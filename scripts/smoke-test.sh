#!/usr/bin/env sh
set -eu

GATEWAY="${GATEWAY:-http://localhost:8080}"
TIMEOUT_SECS="${SMOKE_TIMEOUT_SECS:-10}"

fail() {
  echo "smoke: FAIL: $1" >&2
  exit 1
}

pass() {
  echo "smoke: ok: $1"
}

hint_down() {
  echo "hint: start the stack with: make up && make seed" >&2
}

json_field() {
  python3 -c "import json,sys; print(json.load(sys.stdin)['$1'])"
}

curl_json() {
  _method="$1"
  _url="$2"
  _data="${3-}"
  if [ "$_method" = "POST" ]; then
    curl -sS -f -X POST "$_url" \
      -H "Content-Type: application/json" \
      -d "$_data"
  else
    curl -sS -f "$_url"
  fi
}

if ! curl -sS -o /dev/null --connect-timeout 2 "$GATEWAY/api/catalog/products" 2>/dev/null; then
  echo "smoke: cannot reach $GATEWAY" >&2
  hint_down
  exit 1
fi

products="$(curl_json GET "$GATEWAY/api/catalog/products")" || fail "GET /api/catalog/products"
echo "$products" | grep -q "prod-001" || fail "catalog list missing prod-001"
pass "GET /api/catalog/products includes prod-001"

order_body='{
  "user_id": "660e8400-e29b-41d4-a716-446655440001",
  "items": [{"product_id": "prod-001", "qty": 2, "price_pence": 1999}]
}'
order_json="$(curl_json POST "$GATEWAY/api/orders" "$order_body")" || fail "POST /api/orders"
order_id="$(printf '%s' "$order_json" | json_field id)"
order_status="$(printf '%s' "$order_json" | json_field status)"
[ -n "$order_id" ] || fail "order response missing id"
[ "$order_status" = "pending" ] || fail "expected pending order, got $order_status"
pass "POST /api/orders -> $order_id (pending)"

deadline=$(( $(date +%s) + TIMEOUT_SECS ))
processed=""
while [ "$(date +%s)" -lt "$deadline" ]; do
  detail="$(curl -sS -f "$GATEWAY/api/analytics/orders/$order_id" || true)"
  status="$(printf '%s' "$detail" | json_field status 2>/dev/null || true)"
  if [ "$status" = "processed" ]; then
    processed=1
    break
  fi
  sleep 0.5
done
[ -n "$processed" ] || fail "order $order_id did not become processed within ${TIMEOUT_SECS}s"
pass "order $order_id processed"

summary="$(curl_json GET "$GATEWAY/api/analytics/orders/summary")" || fail "GET /api/analytics/orders/summary"
order_count="$(printf '%s' "$summary" | json_field order_count)"
[ "$order_count" -ge 1 ] || fail "expected order_count >= 1, got $order_count"
pass "GET /api/analytics/orders/summary order_count=$order_count"

echo "smoke: all checks passed"
