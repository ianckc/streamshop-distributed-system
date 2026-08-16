# order-api

Go service for order creation, Postgres transactions, and event publishing.

Currently exposes a health check only. Order endpoints will be added in later phases.

## Run locally

```bash
go run ./cmd/server
```

The server listens on `:3002` by default.

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
| `PORT` | `3002` | HTTP listen port |
| `SERVICE_NAME` | `order-api` | Included in health response |

Copy `.env.example` to `.env` if you use a tool that loads env files locally.

## Project layout

```
cmd/server/          HTTP server entrypoint
internal/config/     Environment configuration
internal/handler/    HTTP handlers
```
