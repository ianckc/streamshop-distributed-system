import os
from dataclasses import dataclass
from urllib.parse import urlparse


@dataclass(frozen=True)
class Config:
    port: int
    service_name: str
    database_url: str
    clickhouse_host: str
    clickhouse_port: int
    clickhouse_user: str
    clickhouse_password: str
    clickhouse_database: str

    @classmethod
    def from_env(cls) -> "Config":
        port_raw = os.getenv("PORT", "3003")
        try:
            port = int(port_raw)
        except ValueError as exc:
            raise ValueError("PORT must be a number") from exc

        clickhouse_url = os.getenv("CLICKHOUSE_URL", "http://localhost:8123")
        host, ch_port = _parse_clickhouse_url(clickhouse_url)

        return cls(
            port=port,
            service_name=os.getenv("SERVICE_NAME", "analytics-api") or "analytics-api",
            database_url=_required("DATABASE_URL"),
            clickhouse_host=host,
            clickhouse_port=ch_port,
            clickhouse_user=os.getenv("CLICKHOUSE_USER", "streamshop") or "streamshop",
            clickhouse_password=os.getenv("CLICKHOUSE_PASSWORD", "streamshop")
            or "streamshop",
            clickhouse_database=os.getenv("CLICKHOUSE_DATABASE", "streamshop")
            or "streamshop",
        )


def _required(name: str) -> str:
    value = os.getenv(name, "")
    if not value:
        raise ValueError(f"{name} is required")
    return value


def _parse_clickhouse_url(url: str) -> tuple[str, int]:
    parsed = urlparse(url)
    host = parsed.hostname or "localhost"
    port = parsed.port or 8123
    return host, port
