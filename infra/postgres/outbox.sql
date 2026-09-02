-- Transactional outbox (order-api). Apply on existing Postgres volumes;
-- init.sql only runs on first start of an empty data directory.
--
-- From repo root:
--   docker compose exec -T postgres psql -U streamshop -d streamshop < infra/postgres/outbox.sql
--
-- Or wipe volumes and recreate: docker compose down -v && make up

CREATE TABLE IF NOT EXISTS outbox (
    id           BIGSERIAL PRIMARY KEY,
    topic        TEXT NOT NULL,
    message_key  TEXT NOT NULL,
    payload      JSONB NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_outbox_unpublished
    ON outbox (id) WHERE published_at IS NULL;
