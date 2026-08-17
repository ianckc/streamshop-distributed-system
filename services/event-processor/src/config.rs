use std::env;

#[derive(Clone, Debug)]
pub struct Config {
    pub port: u16,
    pub service_name: String,
    pub kafka_brokers: String,
    pub kafka_topic: String,
    pub kafka_dlq_topic: String,
    pub kafka_group_id: String,
    pub database_url: String,
    pub clickhouse_url: String,
    pub clickhouse_user: String,
    pub clickhouse_password: String,
    pub clickhouse_database: String,
}

impl Config {
    pub fn from_env() -> anyhow::Result<Self> {
        Ok(Self {
            port: env::var("PORT")
                .ok()
                .filter(|v| !v.is_empty())
                .unwrap_or_else(|| "3004".into())
                .parse()
                .map_err(|_| anyhow::anyhow!("PORT must be a number"))?,
            service_name: env::var("SERVICE_NAME")
                .ok()
                .filter(|v| !v.is_empty())
                .unwrap_or_else(|| "event-processor".into()),
            kafka_brokers: required("KAFKA_BROKERS")?,
            kafka_topic: env::var("KAFKA_TOPIC")
                .ok()
                .filter(|v| !v.is_empty())
                .unwrap_or_else(|| "orders.events".into()),
            kafka_dlq_topic: env::var("KAFKA_DLQ_TOPIC")
                .ok()
                .filter(|v| !v.is_empty())
                .unwrap_or_else(|| "orders.events.dlq".into()),
            kafka_group_id: env::var("KAFKA_GROUP_ID")
                .ok()
                .filter(|v| !v.is_empty())
                .unwrap_or_else(|| "event-processor".into()),
            database_url: required("DATABASE_URL")?,
            clickhouse_url: required("CLICKHOUSE_URL")?,
            clickhouse_user: env::var("CLICKHOUSE_USER")
                .ok()
                .filter(|v| !v.is_empty())
                .unwrap_or_else(|| "streamshop".into()),
            clickhouse_password: env::var("CLICKHOUSE_PASSWORD")
                .ok()
                .filter(|v| !v.is_empty())
                .unwrap_or_else(|| "streamshop".into()),
            clickhouse_database: env::var("CLICKHOUSE_DATABASE")
                .ok()
                .filter(|v| !v.is_empty())
                .unwrap_or_else(|| "streamshop".into()),
        })
    }
}

fn required(name: &str) -> anyhow::Result<String> {
    env::var(name)
        .ok()
        .filter(|v| !v.is_empty())
        .ok_or_else(|| anyhow::anyhow!("{name} is required"))
}
