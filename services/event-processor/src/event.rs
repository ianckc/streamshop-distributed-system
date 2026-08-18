use chrono::{DateTime, Utc};
use serde::Deserialize;
use uuid::Uuid;

pub const EVENT_TYPE_ORDER_CREATED: &str = "order.created";

#[derive(Debug, Clone, PartialEq, Eq, Deserialize)]
pub struct OrderItem {
    pub product_id: String,
    pub qty: i32,
    pub price_pence: i32,
}

#[derive(Debug, Clone, PartialEq, Deserialize)]
pub struct OrderCreated {
    pub event_type: String,
    pub order_id: Uuid,
    pub user_id: Uuid,
    pub items: Vec<OrderItem>,
    pub total_pence: i32,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, thiserror::Error, PartialEq, Eq)]
#[error("{0}")]
pub struct InvalidEvent(pub String);

pub fn parse_order_created(payload: &[u8]) -> Result<OrderCreated, InvalidEvent> {
    let event: OrderCreated = serde_json::from_slice(payload)
        .map_err(|err| InvalidEvent(format!("invalid JSON: {err}")))?;
    if event.event_type != EVENT_TYPE_ORDER_CREATED {
        return Err(InvalidEvent(format!(
            "event_type must be {EVENT_TYPE_ORDER_CREATED}"
        )));
    }
    if event.items.is_empty() {
        return Err(InvalidEvent("items must not be empty".into()));
    }
    if event.total_pence < 0 {
        return Err(InvalidEvent("total_pence must be >= 0".into()));
    }
    Ok(event)
}

#[cfg(test)]
mod tests {
    use super::*;

    const VALID: &str = r#"{
        "event_type": "order.created",
        "order_id": "550e8400-e29b-41d4-a716-446655440000",
        "user_id": "660e8400-e29b-41d4-a716-446655440001",
        "items": [{"product_id": "prod-001", "qty": 2, "price_pence": 1999}],
        "total_pence": 3998,
        "created_at": "2026-08-12T10:00:00.123456789Z"
    }"#;

    #[test]
    fn parses_order_created() {
        let got = parse_order_created(VALID.as_bytes()).expect("parse");
        assert_eq!(got.event_type, EVENT_TYPE_ORDER_CREATED);
        assert_eq!(got.total_pence, 3998);
        assert_eq!(got.items.len(), 1);
        assert_eq!(got.items[0].product_id, "prod-001");
    }

    #[test]
    fn rejects_wrong_event_type() {
        let body = VALID.replace("order.created", "order.cancelled");
        let err = parse_order_created(body.as_bytes()).unwrap_err();
        assert!(err.0.contains("event_type"));
    }

    #[test]
    fn rejects_empty_items() {
        let body = r#"{
            "event_type": "order.created",
            "order_id": "550e8400-e29b-41d4-a716-446655440000",
            "user_id": "660e8400-e29b-41d4-a716-446655440001",
            "items": [],
            "total_pence": 0,
            "created_at": "2026-08-12T10:00:00Z"
        }"#;
        let err = parse_order_created(body.as_bytes()).unwrap_err();
        assert!(err.0.contains("items"));
    }

    #[test]
    fn rejects_invalid_json() {
        let err = parse_order_created(b"not json").unwrap_err();
        assert!(err.0.contains("invalid JSON"));
    }
}
