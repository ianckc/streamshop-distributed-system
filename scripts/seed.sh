#!/usr/bin/env sh
set -eu

ROOT="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$ROOT"

hint_down() {
  echo "hint: start the stack with: make up" >&2
}

if ! docker compose exec -T redpanda rpk cluster health >/dev/null 2>&1; then
  echo "seed: stack does not look running (could not reach Redpanda)" >&2
  hint_down
  exit 1
fi

echo "seed: ensuring Redpanda topics"
ensure_topic() {
  if docker compose exec -T redpanda rpk topic list | grep -qw "$1"; then
    return 0
  fi
  docker compose exec -T redpanda rpk topic create "$1"
}
ensure_topic orders.events
ensure_topic orders.events.dlq

echo "seed: upserting catalog products"
docker compose exec -T mongo mongosh streamshop --quiet --eval '
db.products.replaceOne(
  { _id: "prod-001" },
  {
    _id: "prod-001",
    name: "StreamShop Mug",
    price_pence: 1999,
    attributes: { colour: "navy", material: "ceramic" },
  },
  { upsert: true }
);
db.products.replaceOne(
  { _id: "prod-002" },
  {
    _id: "prod-002",
    name: "StreamShop T-shirt",
    price_pence: 2499,
    attributes: { colour: "black", size: "M" },
  },
  { upsert: true }
);
db.products.replaceOne(
  { _id: "prod-003" },
  {
    _id: "prod-003",
    name: "Distributed Systems Notebook",
    price_pence: 1299,
    attributes: { pages: "192", binding: "paperback" },
  },
  { upsert: true }
);
'

echo "seed: uploading sample product images"
docker compose run --rm --no-deps \
  -v "$ROOT/infra/minio/sample-images:/images:ro" \
  --entrypoint /bin/sh \
  minio-init \
  -c '
    set -eu
    mc alias set local http://minio:9000 "${MINIO_ROOT_USER}" "${MINIO_ROOT_PASSWORD}" >/dev/null
    mc mb --ignore-existing local/product-images >/dev/null
    mc mb --ignore-existing local/exports >/dev/null
    mc cp /images/prod-001.png local/product-images/prod-001.png
    mc cp /images/prod-002.png local/product-images/prod-002.png
    mc cp /images/prod-003.png local/product-images/prod-003.png
  '

echo "seed: flushing Redis catalog cache"
docker compose exec -T redis redis-cli FLUSHALL >/dev/null

echo "seed: done"
