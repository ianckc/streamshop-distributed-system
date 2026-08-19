# StreamShop

Local-first distributed systems showcase — polyglot microservices, event streaming, and multi-store architecture.

```bash
make up && make seed && make smoke
```

That starts the stack (waits until it is healthy), loads sample catalog data and product images, then walks the happy path: list products → place an order → wait until it is `processed` → check analytics.

**Documentation:** [http://localhost:8080/docs/](http://localhost:8080/docs/) after `make up`, or `cd docs && npm start` on [http://localhost:3100/docs/](http://localhost:3100/docs/). Roadmap: [PLAN.md](./PLAN.md).

## 30-second walkthrough

With the stack up:

1. **Browse products** — [http://localhost:8080/api/catalog/products](http://localhost:8080/api/catalog/products)
2. **Place an order** — `POST http://localhost:8080/api/orders` (or `make smoke`)
3. **Watch the event** — [Redpanda Console](http://localhost:8082) → topic `orders.events`
4. **See it processed** — Postgres `orders.status` becomes `processed` (event-processor)
5. **Read analytics** — [http://localhost:8080/api/analytics/orders/summary](http://localhost:8080/api/analytics/orders/summary)

## Make targets

| Target | What it does |
|--------|----------------|
| `make up` | `docker compose up --build -d --wait` |
| `make seed` | Topics, catalog upsert, sample MinIO images, Redis flush |
| `make smoke` | End-to-end check through Traefik (`:8080`) |
| `make logs` | `docker compose logs -f` |
| `make down` | `docker compose down` |

Traefik is the HTTP entrypoint on **:8080** (APIs and docs at `/docs/`). Other consoles: Traefik [dashboard](http://localhost:8081), [Redis Insight](http://localhost:5540), [MinIO](http://localhost:9001), [Redpanda Console](http://localhost:8082). Redpanda Kafka API: `localhost:19092`. ClickHouse HTTP: `localhost:8123`.

### Observability stack (optional)

```bash
docker compose --profile observability up --build
```

Adds Jaeger, Prometheus, Grafana, and the OTel Collector. Jaeger UI: [http://localhost:16686](http://localhost:16686). Prometheus: [http://localhost:9090](http://localhost:9090). Grafana: [http://localhost:3000](http://localhost:3000) (admin/admin).

### Manual checks

`make smoke` covers the pipeline. Direct health/debug ports:

```bash
curl http://localhost:3001/health                 # catalog-api-1 liveness
curl http://localhost:3001/ready                  # catalog-api-1 Mongo ping
curl http://localhost:3011/health                 # catalog-api-2 liveness
curl http://localhost:3002/health                 # order-api liveness
curl http://localhost:3002/ready                  # order-api Postgres ping
curl http://localhost:3004/health                 # event-processor liveness
curl http://localhost:3004/ready                  # event-processor Postgres + ClickHouse
curl http://localhost:3003/health                 # analytics-api liveness
curl http://localhost:3003/ready                  # analytics-api Postgres + ClickHouse
curl http://localhost:8080/api/catalog/products           # via Traefik → nginx
curl http://localhost:8080/api/catalog/products/prod-001  # via Traefik → nginx
curl http://localhost:8080/api/analytics/orders/summary   # via Traefik (ClickHouse aggregates)
```

Place an order:

```bash
curl -s -X POST http://localhost:8080/api/orders \
  -H 'Content-Type: application/json' \
  -d '{
    "user_id": "660e8400-e29b-41d4-a716-446655440001",
    "items": [{"product_id": "prod-001", "qty": 2, "price_pence": 1999}]
  }'
```
