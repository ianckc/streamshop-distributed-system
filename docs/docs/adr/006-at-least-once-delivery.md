# ADR 006: At-least-once delivery with idempotency

## Status

Accepted

## Context

The order pipeline spans HTTP (order-api), a message broker (Redpanda), and an async consumer (event-processor). Network failures, restarts, and consumer rebalances can all cause duplicate message delivery.

Options considered:

- **At-most-once** — fire and forget; may lose messages
- **At-least-once with idempotent consumers** — may duplicate; no message loss
- **Exactly-once** — Kafka transactions + idempotent producer; complex

## Decision

Implement **at-least-once delivery** with idempotency at both the HTTP and consumer layers.

## Rationale

Exactly-once semantics require Kafka transactions, an outbox pattern, and distributed coordination — valuable but disproportionate for a local demo. At-least-once with idempotency is what most production systems actually run.

### Idempotency layers

1. **HTTP layer (implemented)** — optional `Idempotency-Key` in Redis (`SET NX EX 3600`). Value is JSON: `pending` while creating, then `complete` plus `order_id`. Replay returns the original order (`200`). In-flight duplicate → `409`. Redis unavailable + header present → fail-closed `503`. No header → new order every time.
2. **Consumer layer** — ClickHouse inserts use `ReplacingMergeTree` or dedup on `order_id`; Postgres status updates are naturally idempotent

### Offset commit strategy

The processor commits the Kafka offset **after** successful ClickHouse write and Postgres update — never before.

## Consequences

- **Positive:** No message loss; realistic production pattern; simpler than exactly-once
- **Negative:** Duplicate events possible during failures (handled by consumer idempotency). HTTP retries with a key do not create duplicate orders.
- **Mitigation:** DLQ (`orders.events.dlq`) for messages that fail schema validation. Bounded retries then DLQ for storage failures are not built yet. A transactional outbox for the produce path is not built yet.
