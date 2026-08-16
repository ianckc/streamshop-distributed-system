# order-api

Go service for order creation, Postgres transactions, and event publishing.

Currently exposes a health check only. Order endpoints will be added in later phases.

## Run locally

```bash
go run ./cmd/server
```

The server listens on `:3002` by default.

## Run with Docker Compose

From the repository root:

```bash
# Optional: override port / service name
cp .env.example .env

docker compose up --build order-api
```

Then:

```bash
curl http://localhost:${ORDER_API_PORT:-3002}/health
```

Compose reads root `.env` for host/container port mapping. Optional service overrides live in `services/order-api/.env` (copy from `.env.example`; not required to start).

## Health check

```bash
curl http://localhost:3002/health
```

Expected response:

```json
{"status":"ok","service":"order-api"}
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `3002` | HTTP listen port (inside the process / container) |
| `SERVICE_NAME` | `order-api` | Included in health response |

| Compose variable (root `.env`) | Default | Description |
|--------------------------------|---------|-------------|
| `ORDER_API_PORT` | `3002` | Published and container port |
| `ORDER_API_SERVICE_NAME` | `order-api` | Sets `SERVICE_NAME` in the container |

For local `go run`, copy `services/order-api/.env.example` to `services/order-api/.env`.

## Project layout

```
cmd/server/          HTTP server entrypoint
internal/config/     Environment configuration
internal/handler/    HTTP handlers
```
