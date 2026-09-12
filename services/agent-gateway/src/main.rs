mod agent;
mod gateway;
mod state;
mod streaming;

pub use gateway::router::{ApiKey, AppState, StreamShopConfig};

use tracing_subscriber::EnvFilter;

#[tokio::main]
async fn main() {
    dotenvy::dotenv().ok();

    tracing_subscriber::fmt()
        .with_env_filter(EnvFilter::from_default_env())
        .init();

    let redis_url = std::env::var("REDIS_URL").unwrap_or_else(|_| "redis://127.0.0.1:6379".into());
    let redis_client = redis::Client::open(redis_url.as_str()).expect("Invalid Redis URL");
    let redis_conn = redis_client
        .get_multiplexed_async_connection()
        .await
        .expect("Failed to connect to Redis");

    let llm_model = std::env::var("LLM_MODEL").unwrap_or_else(|_| "qwen3:8b".into());
    tracing::info!(%llm_model, "LLM model configured");

    let http_client = reqwest::Client::builder()
        .pool_max_idle_per_host(20)
        .build()
        .expect("Failed to build HTTP client");

    let streamshop = StreamShopConfig {
        analytics_url: std::env::var("STREAMSHOP_ANALYTICS_URL")
            .unwrap_or_else(|_| "http://127.0.0.1:3003".into()),
        order_url: std::env::var("STREAMSHOP_ORDER_URL")
            .unwrap_or_else(|_| "http://127.0.0.1:3002".into()),
        catalog_url: std::env::var("STREAMSHOP_CATALOG_URL")
            .unwrap_or_else(|_| "http://127.0.0.1:3001".into()),
        event_processor_url: std::env::var("STREAMSHOP_EVENT_PROCESSOR_URL")
            .unwrap_or_else(|_| "http://127.0.0.1:3004".into()),
    };
    tracing::info!(
        analytics = %streamshop.analytics_url,
        order = %streamshop.order_url,
        catalog = %streamshop.catalog_url,
        event_processor = %streamshop.event_processor_url,
        "StreamShop tool base URLs"
    );

    let state = AppState {
        redis_conn,
        http_client,
        llm_base_url: std::env::var("LLM_BASE_URL")
            .unwrap_or_else(|_| "https://api.openai.com".into()),
        llm_api_key: ApiKey::new(std::env::var("LLM_API_KEY").expect("LLM_API_KEY must be set")),
        llm_model,
        streamshop,
    };

    let app = gateway::router::app(state);

    let port = std::env::var("PORT").unwrap_or_else(|_| "8080".into());
    let addr = format!("0.0.0.0:{port}");
    let listener = tokio::net::TcpListener::bind(&addr)
        .await
        .expect("Failed to bind");
    tracing::info!("Agent gateway listening on :{port}");
    axum::serve(listener, app).await.expect("Server error");
}
