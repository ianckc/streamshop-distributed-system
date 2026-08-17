-- StreamShop OLTP schema (order-api)
-- Monetary amounts are stored in pence (GBP minor units).

CREATE TABLE IF NOT EXISTS orders (
    id          UUID PRIMARY KEY,
    user_id     UUID NOT NULL,
    status      TEXT NOT NULL DEFAULT 'pending',
    total_pence INTEGER NOT NULL CHECK (total_pence >= 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS order_items (
    id          BIGSERIAL PRIMARY KEY,
    order_id    UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id  TEXT NOT NULL,
    qty         INTEGER NOT NULL CHECK (qty > 0),
    price_pence INTEGER NOT NULL CHECK (price_pence >= 0)
);

CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders (user_id);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders (created_at DESC);
