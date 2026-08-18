from dataclasses import dataclass
from urllib.parse import urlparse

import clickhouse_connect
from clickhouse_connect.driver.client import Client


@dataclass(frozen=True)
class OrderSummary:
    order_count: int
    total_revenue_pence: int
    avg_order_value_pence: float
    total_items: int


class ClickHouseStore:
    def __init__(
        self,
        url: str,
        user: str,
        password: str,
        database: str,
        *,
        client: Client | None = None,
    ) -> None:
        if client is not None:
            self._client = client
            return

        host, port = _parse_clickhouse_url(url)
        self._client = clickhouse_connect.get_client(
            host=host,
            port=port,
            username=user,
            password=password,
            database=database,
        )

    def ping(self) -> None:
        self._client.query("SELECT 1")

    def order_summary(self) -> OrderSummary:
        result = self._client.query(
            """
            SELECT
                count() AS order_count,
                coalesce(sum(total_pence), 0) AS total_revenue_pence,
                coalesce(avg(total_pence), 0) AS avg_order_value_pence,
                coalesce(sum(item_count), 0) AS total_items
            FROM order_events
            FINAL
            """
        )
        row = result.first_row
        return OrderSummary(
            order_count=int(row[0]),
            total_revenue_pence=int(row[1]),
            avg_order_value_pence=float(row[2]),
            total_items=int(row[3]),
        )

    def close(self) -> None:
        self._client.close()


def _parse_clickhouse_url(url: str) -> tuple[str, int]:
    parsed = urlparse(url)
    host = parsed.hostname or "localhost"
    port = parsed.port or 8123
    return host, port
