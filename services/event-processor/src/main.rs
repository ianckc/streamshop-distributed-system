use std::net::SocketAddr;
use std::sync::Arc;

use event_processor::analytics::ClickHouseAnalytics;
use event_processor::config::Config;
use event_processor::consume::{handle_message, send_storage_exhausted_to_dlq_and_commit, KafkaIO};
use event_processor::process::should_dlq;
use event_processor::http::{router, AppState};
use event_processor::orders::PostgresOrders;
use event_processor::telemetry::init_propagation;
use opentelemetry::trace::TracerProvider;
use tokio::net::TcpListener;
use tracing_subscriber::layer::SubscriberExt;
use tracing_subscriber::util::SubscriberInitExt;
use tracing_subscriber::EnvFilter;

fn init_tracing(service_name: &str) -> Option<opentelemetry_sdk::trace::SdkTracerProvider> {
    let endpoint = std::env::var("OTEL_EXPORTER_OTLP_ENDPOINT").ok()?;
    if endpoint.is_empty() {
        return None;
    }

    let exporter = opentelemetry_otlp::SpanExporter::builder()
        .with_tonic()
        .build()
        .ok()?;

    let provider = opentelemetry_sdk::trace::SdkTracerProvider::builder()
        .with_batch_exporter(exporter)
        .with_resource(
            opentelemetry_sdk::Resource::builder()
                .with_service_name(service_name.to_owned())
                .build(),
        )
        .build();

    let telemetry =
        tracing_opentelemetry::layer().with_tracer(provider.tracer(service_name.to_owned()));

    tracing_subscriber::registry()
        .with(EnvFilter::from_default_env().add_directive("info".parse().unwrap()))
        .with(tracing_subscriber::fmt::layer().json())
        .with(telemetry)
        .init();

    Some(provider)
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let cfg = Config::from_env()?;
    init_propagation();
    let _provider = init_tracing(&cfg.service_name);
    if _provider.is_none() {
        tracing_subscriber::fmt()
            .json()
            .with_env_filter(EnvFilter::from_default_env().add_directive("info".parse()?))
            .init();
    }

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

    let max_retries = cfg.storage_max_retries;
    let consume = tokio::spawn(async move {
        loop {
            match kafka.consumer.recv().await {
                Err(err) => {
                    tracing::error!(error = %err, "kafka recv failed");
                    tokio::time::sleep(std::time::Duration::from_secs(1)).await;
                }
                Ok(msg) => {
                    let mut attempt = 0u32;
                    loop {
                        match handle_message(&kafka, &msg, analytics.as_ref(), orders.as_ref())
                            .await
                        {
                            Ok(()) => break,
                            Err(err) => {
                                attempt += 1;
                                if should_dlq(attempt, max_retries) {
                                    tracing::error!(
                                        error = %err,
                                        attempt,
                                        max_retries,
                                        "storage retries exhausted; sending to DLQ",
                                    );
                                    let payload =
                                        rdkafka::Message::payload(&msg).unwrap_or(&[]);
                                    if let Err(dlq_err) =
                                        send_storage_exhausted_to_dlq_and_commit(
                                            &kafka, &msg, payload, attempt, &err,
                                        )
                                        .await
                                    {
                                        tracing::error!(
                                            error = %dlq_err,
                                            "failed to send exhausted message to DLQ; will retry",
                                        );
                                        continue;
                                    }
                                    break;
                                }
                                tracing::error!(
                                    error = %err,
                                    attempt,
                                    max_retries,
                                    "message not committed; retrying",
                                );
                                tokio::time::sleep(std::time::Duration::from_secs(1)).await;
                            }
                        }
                    }
                }
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
