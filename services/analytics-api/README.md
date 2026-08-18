# analytics-api

Python FastAPI service for OLAP queries (ClickHouse) and order detail reads (PostgreSQL).

See the **analytics-api** page in the docs site (`cd docs && npm start` → Services → analytics-api).

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Liveness |
| GET | `/ready` | Postgres + ClickHouse connectivity |
| GET | `/api/analytics/orders/summary` | Aggregate metrics from ClickHouse |
| GET | `/api/analytics/orders/{id}` | Order detail from Postgres |

## Local development

```bash
cp .env.example .env
pip install -e ".[dev]"
pytest
python -m analytics_api.main
```

Requires Postgres and ClickHouse (e.g. `docker compose up postgres clickhouse` from repo root).
