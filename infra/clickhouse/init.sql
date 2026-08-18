CREATE DATABASE IF NOT EXISTS streamshop;

CREATE TABLE IF NOT EXISTS streamshop.order_events
(
    order_id UUID,
    user_id UUID,
    total_pence UInt32,
    item_count UInt8,
    event_time DateTime64(3, 'UTC'),
    processed_at DateTime64(3, 'UTC')
)
ENGINE = ReplacingMergeTree(processed_at)
ORDER BY (order_id);
