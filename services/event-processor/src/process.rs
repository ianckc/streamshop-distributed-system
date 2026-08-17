use async_trait::async_trait;
use uuid::Uuid;

use crate::event::OrderCreated;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct OrderEventRow {
    pub order_id: Uuid,
    pub user_id: Uuid,
    pub total_pence: u32,
    pub item_count: u8,
    pub event_time: chrono::DateTime<chrono::Utc>,
}

impl From<&OrderCreated> for OrderEventRow {
    fn from(event: &OrderCreated) -> Self {
        Self {
            order_id: event.order_id,
            user_id: event.user_id,
            total_pence: u32::try_from(event.total_pence).unwrap_or(0),
            item_count: u8::try_from(event.items.len()).unwrap_or(u8::MAX),
            event_time: event.created_at,
        }
    }
}

#[async_trait]
pub trait AnalyticsStore: Send + Sync {
    async fn insert_order_event(&self, row: OrderEventRow) -> anyhow::Result<()>;
    async fn ping(&self) -> anyhow::Result<()>;
}

#[async_trait]
pub trait OrderStatusStore: Send + Sync {
    async fn mark_processed(&self, order_id: Uuid) -> anyhow::Result<()>;
    async fn ping(&self) -> anyhow::Result<()>;
}

#[derive(Debug, thiserror::Error)]
pub enum ProcessError {
    #[error("invalid event: {0}")]
    Invalid(String),
    #[error(transparent)]
    Storage(#[from] anyhow::Error),
}

pub async fn process_order_created(
    payload: &[u8],
    analytics: &dyn AnalyticsStore,
    orders: &dyn OrderStatusStore,
) -> Result<OrderEventRow, ProcessError> {
    let event =
        crate::event::parse_order_created(payload).map_err(|err| ProcessError::Invalid(err.0))?;
    let row = OrderEventRow::from(&event);
    analytics.insert_order_event(row.clone()).await?;
    orders.mark_processed(event.order_id).await?;
    Ok(row)
}

#[cfg(test)]
mod tests {
    use std::sync::Mutex;

    use super::*;

    struct MemoryAnalytics {
        rows: Mutex<Vec<OrderEventRow>>,
        fail: bool,
    }

    #[async_trait]
    impl AnalyticsStore for MemoryAnalytics {
        async fn insert_order_event(&self, row: OrderEventRow) -> anyhow::Result<()> {
            if self.fail {
                anyhow::bail!("clickhouse down");
            }
            self.rows.lock().unwrap().push(row);
            Ok(())
        }

        async fn ping(&self) -> anyhow::Result<()> {
            Ok(())
        }
    }

    struct MemoryOrders {
        ids: Mutex<Vec<Uuid>>,
        fail: bool,
    }

    #[async_trait]
    impl OrderStatusStore for MemoryOrders {
        async fn mark_processed(&self, order_id: Uuid) -> anyhow::Result<()> {
            if self.fail {
                anyhow::bail!("postgres down");
            }
            self.ids.lock().unwrap().push(order_id);
            Ok(())
        }

        async fn ping(&self) -> anyhow::Result<()> {
            Ok(())
        }
    }

    const VALID: &str = r#"{
        "event_type": "order.created",
        "order_id": "550e8400-e29b-41d4-a716-446655440000",
        "user_id": "660e8400-e29b-41d4-a716-446655440001",
        "items": [{"product_id": "prod-001", "qty": 2, "price_pence": 1999}],
        "total_pence": 3998,
        "created_at": "2026-08-12T10:00:00Z"
    }"#;

    #[tokio::test]
    async fn writes_clickhouse_then_marks_processed() {
        let analytics = MemoryAnalytics {
            rows: Mutex::new(Vec::new()),
            fail: false,
        };
        let orders = MemoryOrders {
            ids: Mutex::new(Vec::new()),
            fail: false,
        };

        let row = process_order_created(VALID.as_bytes(), &analytics, &orders)
            .await
            .expect("process");
        assert_eq!(row.item_count, 1);
        assert_eq!(row.total_pence, 3998);
        assert_eq!(analytics.rows.lock().unwrap().len(), 1);
        assert_eq!(orders.ids.lock().unwrap().len(), 1);
    }

    #[tokio::test]
    async fn invalid_payload_does_not_write() {
        let analytics = MemoryAnalytics {
            rows: Mutex::new(Vec::new()),
            fail: false,
        };
        let orders = MemoryOrders {
            ids: Mutex::new(Vec::new()),
            fail: false,
        };

        let err = process_order_created(b"{}", &analytics, &orders)
            .await
            .unwrap_err();
        assert!(matches!(err, ProcessError::Invalid(_)));
        assert!(analytics.rows.lock().unwrap().is_empty());
        assert!(orders.ids.lock().unwrap().is_empty());
    }

    #[tokio::test]
    async fn storage_failure_is_not_invalid() {
        let analytics = MemoryAnalytics {
            rows: Mutex::new(Vec::new()),
            fail: true,
        };
        let orders = MemoryOrders {
            ids: Mutex::new(Vec::new()),
            fail: false,
        };

        let err = process_order_created(VALID.as_bytes(), &analytics, &orders)
            .await
            .unwrap_err();
        assert!(matches!(err, ProcessError::Storage(_)));
        assert!(orders.ids.lock().unwrap().is_empty());
    }
}
