# ADR 006: At-least-once delivery with idempotency

## Status

Accepted

## Context

The order pipeline spans HTTP (order-api), a message broker (Redpanda), and an async consumer (event-processor). Network failures, restarts, and consumer rebalances can all cause duplicate message delivery.

Options considered:

- **At-most-once** — fire and forget; may lose messages
- **At-least-once with idempotent consumers** — may duplicate; no message loss on the consume path
- **Exactly-once** — Kafka transactions + idempotent producer + distributed coordination; complex

## Decision

Implement **at-least-once delivery** with idempotency at the HTTP, produce, and consumer layers.

## Rationale

Exactly-once end-to-end is valuable in production but disproportionate for a local demo. At-least-once with idempotency is what most production systems actually run.

The **produce path** uses a transactional outbox ([ADR 007](./transactional-outbox)) so orders and event intent commit atomically; an in-process poller publishes to Kafka. That removes lost events from dual-write; duplicates on republish are still possible.

### Idempotency layers

1. **HTTP layer (implemented)** — optional `Idempotency-Key` in Redis (`SET NX EX 3600`). Value is JSON: `pending` while creating, then `complete` plus `order_id`. Replay returns the original order (`200`). In-flight duplicate → `409`. Redis unavailable + header present → fail-closed `503`. No header → new order every time.
2. **Consumer layer** — ClickHouse inserts use `ReplacingMergeTree` or dedup on `order_id`; Postgres status updates are naturally idempotent

### Offset commit strategy

The processor commits the Kafka offset **after** successful ClickHouse write and Postgres update — never before.

## Consequences

- **Positive:** No lost events on the produce path (outbox); realistic production pattern; simpler than exactly-once end-to-end
- **Negative:** Duplicate events possible if the poller republishes or the consumer redelivers (handled by consumer idempotency). HTTP retries with a key do not create duplicate orders.
- **Mitigation:** DLQ (`orders.events.dlq`) for two failure classes:
  - **Poison** — permanently unprocessable messages (bad JSON, wrong `event_type`, empty `items`, negative `total_pence`). Sent to DLQ immediately; offset committed; no retry.
  - **Transient** — storage failures (ClickHouse / Postgres unavailable). Retried up to `STORAGE_MAX_RETRIES` times (default 5, 1 s sleep). If still failing, sent to DLQ with a reason prefixed `storage: exhausted after N retries: …` so operators can distinguish from poison. The trade-off: a prolonged backend outage parks messages in the DLQ rather than blocking the consumer forever.
