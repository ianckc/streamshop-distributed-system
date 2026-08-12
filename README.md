# StreamShop

A local-first distributed systems showcase — polyglot microservices, event streaming, multi-store architecture, and full observability, runnable with Docker Compose.

## Documentation

Full architecture docs, ADRs, runbooks, and sequence diagrams live in the Docusaurus site:

```bash
cd docs
npm install
npm start
```

Open [http://localhost:3000](http://localhost:3000).

## What's inside

| Component | Technology |
|-----------|------------|
| API Gateway | Traefik v3 |
| Load Balancer | nginx |
| Cache | Redis 7 |
| Message Queue | Redpanda (Kafka-compatible) |
| SQL (OLTP) | PostgreSQL 16 |
| NoSQL | MongoDB 7 |
| OLAP | ClickHouse 24 |
| Object Storage | MinIO |
| Observability | OpenTelemetry, Jaeger, Prometheus, Grafana |

## Microservices

| Service | Language | Responsibility |
|---------|----------|----------------|
| `catalog-api` | Node.js | Product catalog, caching, image uploads |
| `order-api` | Go | Orders, Postgres transactions, event publishing |
| `analytics-api` | Python | OLAP queries and reporting |
| `event-processor` | Rust | Async event consumption and enrichment |

## Quick start (coming soon)

```bash
docker compose --profile full up -d
```

## License

MIT
