# ADR 007: Transactional outbox for order events

## Status

Accepted

## Context

`order-api` must persist orders in PostgreSQL and publish `order.created` to Redpanda. Writing to Postgres and publishing to Kafka in separate steps is a **dual-write**: if the DB commit succeeds and the produce fails, the order exists but no event is emitted (fail-open `201`).

Options considered:

- **Dual-write (produce after commit)** — simple; may lose events when the broker is down
- **Transactional outbox** — insert an outbox row in the same DB transaction; a separate process publishes and marks rows
- **Kafka transactions + idempotent producer** — exactly-once produce; heavy for a local demo

## Decision

Use a **transactional outbox** in PostgreSQL plus an **in-process poller** in order-api.

On `POST /api/orders`, order-api inserts `orders`, `order_items`, and one `outbox` row in a **single transaction**, then returns `201`. It does **not** call Kafka in the handler.

A background loop (~300 ms) claims unpublished rows (`SELECT … FOR UPDATE SKIP LOCKED`), publishes the stored JSON to `orders.events`, and sets `published_at`.

## Rationale

The outbox makes the order and the intent to publish atomic — no lost events when Redpanda is temporarily unavailable. The handler stays fast and decoupled from broker latency.

At scale, the poller would run as a **separate worker service** (or use a relay like Debezium). For StreamShop, an in-process loop keeps the demo simple while showing the pattern.

This does **not** provide exactly-once delivery: if the process dies after Kafka ack but before `published_at`, the row may be republished. Consumers remain at-least-once with idempotent handling (see [ADR 006](./at-least-once-delivery)).

## Consequences

- **Positive:** No dual-write gap on the produce path; broker outages do not lose events; `201` reflects DB success only
- **Negative:** Event delivery is asynchronous (typically sub-second locally); duplicate publishes still possible on poller crash
- **Mitigation:** Consumer idempotency (ReplacingMergeTree, idempotent status updates); partial index on unpublished rows; publish failures logged and retried

## Schema

Defined in `infra/postgres/init.sql` (fresh volumes) and `infra/postgres/outbox.sql` (apply on existing volumes):

```sql
CREATE TABLE outbox (
  id           BIGSERIAL PRIMARY KEY,
  topic        TEXT NOT NULL,
  message_key  TEXT NOT NULL,
  payload      JSONB NOT NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  published_at TIMESTAMPTZ
);
```

Topic `orders.events`, key = order id, payload = `order.created` JSON.
