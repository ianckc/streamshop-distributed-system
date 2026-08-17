use std::sync::Arc;

use axum::extract::State;
use axum::http::StatusCode;
use axum::response::IntoResponse;
use axum::routing::get;
use axum::{Json, Router};
use serde::Serialize;

use crate::process::{AnalyticsStore, OrderStatusStore};

#[derive(Clone)]
pub struct AppState {
    pub service_name: String,
    pub analytics: Arc<dyn AnalyticsStore>,
    pub orders: Arc<dyn OrderStatusStore>,
}

#[derive(Serialize)]
struct HealthResponse {
    status: &'static str,
    service: String,
}

#[derive(Serialize)]
struct ErrorResponse {
    error: String,
}

pub fn router(state: AppState) -> Router {
    Router::new()
        .route("/health", get(health))
        .route("/ready", get(ready))
        .with_state(state)
}

async fn health(State(state): State<AppState>) -> Json<HealthResponse> {
    Json(HealthResponse {
        status: "ok",
        service: state.service_name,
    })
}

async fn ready(State(state): State<AppState>) -> impl IntoResponse {
    if state.analytics.ping().await.is_err() || state.orders.ping().await.is_err() {
        return (
            StatusCode::SERVICE_UNAVAILABLE,
            Json(ErrorResponse {
                error: "not ready".into(),
            }),
        )
            .into_response();
    }
    Json(HealthResponse {
        status: "ok",
        service: state.service_name,
    })
    .into_response()
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::process::{OrderEventRow, OrderStatusStore};
    use axum::body::Body;
    use axum::http::{Request, StatusCode};
    use tower::ServiceExt;
    use uuid::Uuid;

    struct OkStore;

    #[async_trait::async_trait]
    impl AnalyticsStore for OkStore {
        async fn insert_order_event(&self, _row: OrderEventRow) -> anyhow::Result<()> {
            Ok(())
        }
        async fn ping(&self) -> anyhow::Result<()> {
            Ok(())
        }
    }

    #[async_trait::async_trait]
    impl OrderStatusStore for OkStore {
        async fn mark_processed(&self, _order_id: Uuid) -> anyhow::Result<()> {
            Ok(())
        }
        async fn ping(&self) -> anyhow::Result<()> {
            Ok(())
        }
    }

    struct DownAnalytics;

    #[async_trait::async_trait]
    impl AnalyticsStore for DownAnalytics {
        async fn insert_order_event(&self, _row: OrderEventRow) -> anyhow::Result<()> {
            Ok(())
        }
        async fn ping(&self) -> anyhow::Result<()> {
            anyhow::bail!("down")
        }
    }

    fn state(analytics: Arc<dyn AnalyticsStore>, orders: Arc<dyn OrderStatusStore>) -> AppState {
        AppState {
            service_name: "event-processor".into(),
            analytics,
            orders,
        }
    }

    #[tokio::test]
    async fn health_ok() {
        let app = router(state(Arc::new(OkStore), Arc::new(OkStore)));
        let res = app
            .oneshot(
                Request::builder()
                    .uri("/health")
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(res.status(), StatusCode::OK);
    }

    #[tokio::test]
    async fn ready_ok() {
        let app = router(state(Arc::new(OkStore), Arc::new(OkStore)));
        let res = app
            .oneshot(
                Request::builder()
                    .uri("/ready")
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(res.status(), StatusCode::OK);
    }

    #[tokio::test]
    async fn ready_503_when_clickhouse_down() {
        let app = router(state(Arc::new(DownAnalytics), Arc::new(OkStore)));
        let res = app
            .oneshot(
                Request::builder()
                    .uri("/ready")
                    .body(Body::empty())
                    .unwrap(),
            )
            .await
            .unwrap();
        assert_eq!(res.status(), StatusCode::SERVICE_UNAVAILABLE);
    }
}
