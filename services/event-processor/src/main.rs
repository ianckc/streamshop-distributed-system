use std::net::SocketAddr;
use std::sync::Arc;

use event_processor::analytics::ClickHouseAnalytics;
use event_processor::config::Config;
use event_processor::consume::{handle_message, KafkaIO};
use event_processor::http::{router, AppState};
use event_processor::orders::PostgresOrders;
use tokio::net::TcpListener;
use tracing_subscriber::EnvFilter;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    tracing_subscriber::fmt()
        .json()
        .with_env_filter(EnvFilter::from_default_env().add_directive("info".parse()?))
        .init();

    let cfg = Config::from_env()?;
    let analytics = Arc::new(ClickHouseAnalytics::connect(
        &cfg.clickhouse_url,
        &cfg.clickhouse_user,
        &cfg.clickhouse_password,
        &cfg.clickhouse_database,
    ));
    let orders = Arc::new(PostgresOrders::connect(&cfg.database_url).await?);
    let kafka = KafkaIO::connect(&cfg)?;

    let state = AppState {
        service_name: cfg.service_name.clone(),
        analytics: analytics.clone(),
        orders: orders.clone(),
    };
    let app = router(state);
    let addr = SocketAddr::from(([0, 0, 0, 0], cfg.port));
    let listener = TcpListener::bind(addr).await?;
    tracing::info!(%addr, "http listening");

    let http = tokio::spawn(async move {
        if let Err(err) = axum::serve(listener, app).await {
            tracing::error!(error = %err, "http server failed");
        }
    });

    let consume = tokio::spawn(async move {
        loop {
            match kafka.consumer.recv().await {
                Err(err) => {
                    tracing::error!(error = %err, "kafka recv failed");
                    tokio::time::sleep(std::time::Duration::from_secs(1)).await;
                }
                Ok(msg) => loop {
                    match handle_message(&kafka, &msg, analytics.as_ref(), orders.as_ref()).await {
                        Ok(()) => break,
                        Err(err) => {
                            tracing::error!(error = %err, "message not committed; retrying");
                            tokio::time::sleep(std::time::Duration::from_secs(1)).await;
                        }
                    }
                },
            }
        }
    });

    tokio::select! {
        _ = http => {}
        _ = consume => {}
        _ = tokio::signal::ctrl_c() => {
            tracing::info!("shutting down");
        }
    }
    Ok(())
}
