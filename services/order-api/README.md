# order-api

Go service for order creation and Postgres transactions.

Currently exposes:

- `GET /health`
- `POST /api/orders`

Event publishing to Redpanda comes in a later phase.

## Run with Docker Compose

From the repository root:

```bash
cp .env.example .env   # optional
docker compose up --build
```

This starts **Postgres** and **order-api**.

### Create an order

```bash
curl -s -X POST http://localhost:3002/api/orders \
  -H 'Content-Type: application/json' \
  -d '{
    "user_id": "660e8400-e29b-41d4-a716-446655440001",
    "items": [
      {"product_id": "prod-001", "qty": 2, "price_pence": 1999}
    ]
  }'
```

### Inspect Postgres

```bash
docker compose exec postgres \
  psql -U streamshop -d streamshop \
  -c 'SELECT id, user_id, status, total_pence FROM orders;'
```

## Run locally (without Compose for the app)

```bash
# Start only Postgres
docker compose up -d postgres

# Point at localhost Postgres
cp .env.example .env
# DATABASE_URL should use @localhost:5432

go run ./cmd/server
```

## Health check

```bash
curl http://localhost:3002/health
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `3002` | HTTP listen port |
| `SERVICE_NAME` | `order-api` | Included in health response |
| `DATABASE_URL` | *(required)* | Postgres connection string |

| Compose variable (root `.env`) | Default | Description |
|--------------------------------|---------|-------------|
| `ORDER_API_PORT` | `3002` | Published and container port |
| `ORDER_API_SERVICE_NAME` | `order-api` | Sets `SERVICE_NAME` in the container |
| `DATABASE_URL` | `postgres://...@postgres:5432/...` | Used inside Compose network |

## Project layout

```
cmd/server/                 HTTP server entrypoint
internal/config/            Environment configuration
internal/handler/           HTTP handlers
internal/model/             Domain types
internal/store/             Store interfaces
internal/store/postgres/    Postgres implementation
```
