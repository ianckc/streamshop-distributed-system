use axum::Router;
use axum::error_handling::HandleErrorLayer;
use axum::http::StatusCode;
use redis::aio::MultiplexedConnection;
use std::fmt;
use tower::{BoxError, ServiceBuilder};
use tower_http::trace::TraceLayer;

/// Wraps the LLM API key with a redacted Debug impl
/// to prevent accidental key leakage in logs or crash dumps.
#[derive(Clone)]
pub struct ApiKey(String);

impl ApiKey {
    pub fn new(key: String) -> Self {
        Self(key)
    }

    /// Returns the raw key value for use in auth headers only.
    pub(crate) fn api_key(&self) -> &str {
        &self.0
    }
}

impl fmt::Debug for ApiKey {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        write!(f, "[REDACTED]")
    }
}

#[derive(Clone)]
pub struct AppState {
    pub redis_conn: MultiplexedConnection,
    pub llm_client: reqwest::Client,
    pub llm_base_url: String,
    pub llm_api_key: ApiKey,
}

pub fn app(state: AppState) -> Router {
    Router::new()
        .route(
            "/v1/agent/stream",
            axum::routing::post(crate::streaming::sse_handler::handle_stream),
        )
        .route("/health", axum::routing::get(|| async { "ok" }))
        .layer(axum::extract::DefaultBodyLimit::max(1_048_576)) // 1 MB request body limit
        .layer(
            ServiceBuilder::new()
                .layer(HandleErrorLayer::new(|err: BoxError| async move {
                    if err.is::<tower::load_shed::error::Overloaded>() {
                        (
                            StatusCode::SERVICE_UNAVAILABLE,
                            "server at capacity".to_string(),
                        )
                    } else {
                        (
                            StatusCode::INTERNAL_SERVER_ERROR,
                            format!("internal error: {err}"),
                        )
                    }
                }))
                .layer(TraceLayer::new_for_http())
                .layer(tower::timeout::TimeoutLayer::new(
                    std::time::Duration::from_secs(120),
                ))
                .layer(tower::limit::ConcurrencyLimitLayer::new(256)),
        )
        .with_state(state)
}
