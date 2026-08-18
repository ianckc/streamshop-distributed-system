use std::str::FromStr;

use async_trait::async_trait;
use tokio_postgres::NoTls;
use uuid::Uuid;

use crate::process::OrderStatusStore;

pub struct PostgresOrders {
    client: tokio_postgres::Client,
}

impl PostgresOrders {
    pub async fn connect(database_url: &str) -> anyhow::Result<Self> {
        let config = tokio_postgres::Config::from_str(database_url)?;
        let (client, connection) = config.connect(NoTls).await?;
        tokio::spawn(async move {
            if let Err(err) = connection.await {
                tracing::error!(error = %err, "postgres connection closed");
            }
        });
        Ok(Self { client })
    }
}

#[async_trait]
impl OrderStatusStore for PostgresOrders {
    async fn mark_processed(&self, order_id: Uuid) -> anyhow::Result<()> {
        let n = self
            .client
            .execute(
                "UPDATE orders SET status = 'processed' WHERE id = $1",
                &[&order_id],
            )
            .await?;
        if n == 0 {
            tracing::warn!(order_id = %order_id, "no order row to mark processed");
        }
        Ok(())
    }

    async fn ping(&self) -> anyhow::Result<()> {
        self.client.simple_query("SELECT 1").await?;
        Ok(())
    }
}
