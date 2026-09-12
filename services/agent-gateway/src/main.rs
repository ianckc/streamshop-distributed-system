mod agent;
mod gateway;
mod state;
mod streaming;

pub use gateway::router::{ApiKey, AppState};

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

    let state = AppState {
        redis_conn,
        llm_client: reqwest::Client::builder()
            .pool_max_idle_per_host(20)
            .build()
            .expect("Failed to build HTTP client"),
        llm_base_url: std::env::var("LLM_BASE_URL")
            .unwrap_or_else(|_| "https://api.openai.com".into()),
        llm_api_key: ApiKey::new(std::env::var("LLM_API_KEY").expect("LLM_API_KEY must be set")),
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
