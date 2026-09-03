use std::time::Duration;

use rdkafka::config::ClientConfig;
use rdkafka::consumer::{Consumer, StreamConsumer};
use rdkafka::message::BorrowedMessage;
use rdkafka::producer::{FutureProducer, FutureRecord};
use rdkafka::Message;
use tracing::Instrument;
use tracing_opentelemetry::OpenTelemetrySpanExt;

use crate::config::Config;
use crate::process::{process_order_created, AnalyticsStore, OrderStatusStore, ProcessError};
use crate::telemetry::extract_context;

pub struct KafkaIO {
    pub consumer: StreamConsumer,
    pub producer: FutureProducer,
    pub dlq_topic: String,
}

impl KafkaIO {
    pub fn connect(cfg: &Config) -> anyhow::Result<Self> {
        let consumer: StreamConsumer = ClientConfig::new()
            .set("bootstrap.servers", &cfg.kafka_brokers)
            .set("group.id", &cfg.kafka_group_id)
            .set("enable.auto.commit", "false")
            .set("auto.offset.reset", "earliest")
            .set("session.timeout.ms", "6000")
            .create()?;
        consumer.subscribe(&[&cfg.kafka_topic])?;

        let producer: FutureProducer = ClientConfig::new()
            .set("bootstrap.servers", &cfg.kafka_brokers)
            .create()?;

        Ok(Self {
            consumer,
            producer,
            dlq_topic: cfg.kafka_dlq_topic.clone(),
        })
    }
}

pub async fn handle_message(
    kafka: &KafkaIO,
    msg: &BorrowedMessage<'_>,
    analytics: &dyn AnalyticsStore,
    orders: &dyn OrderStatusStore,
) -> anyhow::Result<()> {
    let parent_cx = extract_context(msg.headers());
    let span = tracing::info_span!(
        "kafka.process order.created",
        messaging.system = "kafka",
        messaging.destination.name = "orders.events",
    );
    span.set_parent(parent_cx);

    async {
        let payload = msg.payload().unwrap_or(&[]);
        match process_order_created(payload, analytics, orders).await {
            Ok(row) => {
                tracing::info!(order_id = %row.order_id, "processed order.created");
                kafka
                    .consumer
                    .commit_message(msg, rdkafka::consumer::CommitMode::Sync)?;
                Ok(())
            }
            Err(ProcessError::Invalid(reason)) => {
                tracing::warn!(error = %reason, "invalid order.created; sending to DLQ");
                send_dlq_and_commit(kafka, msg, payload, &reason).await
            }
            Err(ProcessError::Storage(err)) => {
                tracing::error!(error = %err, "storage failure; will retry");
                Err(err)
            }
        }
    }
    .instrument(span)
    .await
}

pub async fn send_storage_exhausted_to_dlq_and_commit(
    kafka: &KafkaIO,
    msg: &BorrowedMessage<'_>,
    payload: &[u8],
    attempts: u32,
    err: &anyhow::Error,
) -> anyhow::Result<()> {
    let reason = format!("storage: exhausted after {attempts} retries: {err}");
    send_dlq_and_commit(kafka, msg, payload, &reason).await
}

pub async fn send_dlq_and_commit(
    kafka: &KafkaIO,
    msg: &BorrowedMessage<'_>,
    payload: &[u8],
    reason: &str,
) -> anyhow::Result<()> {
    send_dlq(kafka, payload, reason).await?;
    kafka
        .consumer
        .commit_message(msg, rdkafka::consumer::CommitMode::Sync)?;
    Ok(())
}

async fn send_dlq(kafka: &KafkaIO, payload: &[u8], reason: &str) -> anyhow::Result<()> {
    let key = reason.as_bytes();
    kafka
        .producer
        .send(
            FutureRecord::to(&kafka.dlq_topic).payload(payload).key(key),
            Duration::from_secs(5),
        )
        .await
        .map_err(|(err, _)| anyhow::anyhow!("dlq produce: {err}"))?;
    Ok(())
}
