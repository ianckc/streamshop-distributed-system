use async_trait::async_trait;
use chrono::{DateTime, Utc};
use clickhouse::Client;
use serde::Serialize;
use uuid::Uuid;

use crate::process::{AnalyticsStore, OrderEventRow};

pub struct ClickHouseAnalytics {
    client: Client,
}

#[derive(clickhouse::Row, Serialize)]
struct InsertRow {
    #[serde(with = "clickhouse::serde::uuid")]
    order_id: Uuid,
    #[serde(with = "clickhouse::serde::uuid")]
    user_id: Uuid,
    total_pence: u32,
    item_count: u8,
    #[serde(with = "clickhouse::serde::chrono::datetime64::millis")]
    event_time: DateTime<Utc>,
    #[serde(with = "clickhouse::serde::chrono::datetime64::millis")]
    processed_at: DateTime<Utc>,
}

impl ClickHouseAnalytics {
    pub fn connect(url: &str, user: &str, password: &str, database: &str) -> Self {
        let client = Client::default()
            .with_url(url)
            .with_user(user)
            .with_password(password)
            .with_database(database);
        Self { client }
    }
}

#[async_trait]
impl AnalyticsStore for ClickHouseAnalytics {
    async fn insert_order_event(&self, row: OrderEventRow) -> anyhow::Result<()> {
        let mut insert = self.client.insert("order_events")?;
        insert
            .write(&InsertRow {
                order_id: row.order_id,
                user_id: row.user_id,
                total_pence: row.total_pence,
                item_count: row.item_count,
                event_time: row.event_time,
                processed_at: Utc::now(),
            })
            .await?;
        insert.end().await?;
        Ok(())
    }

    async fn ping(&self) -> anyhow::Result<()> {
        let _: u8 = self.client.query("SELECT 1").fetch_one().await?;
        Ok(())
    }
}
