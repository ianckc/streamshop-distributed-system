#!/usr/bin/env sh
# CI-parity quality gate for the software factory.
# Usage:
#   just gate order-api,catalog-api
#   just gate                    # all supported services
#   SERVICES=order-api,catalog-api ./scripts/factory-gate.sh
set -eu

ROOT="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

ALL_SERVICES="order-api catalog-api analytics-api event-processor agent-gateway"

fail() {
  echo "gate: FAIL: $1" >&2
  exit 1
}

pass() {
  echo "gate: ok: $1"
}

info() {
  echo "gate: $1"
}

if [ -z "${SERVICES:-}" ]; then
  fail "SERVICES is empty — pass SERVICES=order-api,catalog-api (from the ticket) or SERVICES=all"
fi

# Normalize commas/spaces → newline list
services_raw="$(printf '%s' "$SERVICES" | tr ',' ' ')"
services=""
for s in $services_raw; do
  [ -n "$s" ] || continue
  if [ "$s" = "all" ]; then
    services="$ALL_SERVICES"
    break
  fi
  services="$services $s"
done

services="$(printf '%s' "$services" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"
[ -n "$services" ] || fail "SERVICES resolved to empty"

info "running for: $services"

gate_order_api() {
  info "order-api — gofmt + go test"
  cd "$ROOT/services/order-api"
  unformatted="$(gofmt -l .)"
  if [ -n "$unformatted" ]; then
    echo "$unformatted" >&2
    fail "order-api: gofmt needed on files above"
  fi
  go test ./...
  pass "order-api"
}

gate_catalog_api() {
  info "catalog-api — lint + typecheck + test"
  cd "$ROOT/services/catalog-api"
  if [ ! -d node_modules ]; then
    info "catalog-api — npm ci (node_modules missing)"
    npm ci
  fi
  npm run lint
  npm run typecheck
  npm test
  pass "catalog-api"
}

gate_analytics_api() {
  info "analytics-api — pytest"
  cd "$ROOT/services/analytics-api"
  # Prefer uv: needs Python >=3.12; bare `pip` is often missing on macOS.
  if command -v uv >/dev/null 2>&1; then
    info "analytics-api — uv run --python 3.12 --extra dev pytest"
    uv run --python 3.12 --extra dev pytest
  else
    PY=""
    for cand in python3.12 python3; do
      if command -v "$cand" >/dev/null 2>&1; then
        PY="$cand"
        break
      fi
    done
    [ -n "$PY" ] || fail "analytics-api: need uv or python3.12+ (install uv, or match CI Python 3.12)"
    info "analytics-api — $PY -m pip install -e '.[dev]'"
    "$PY" -m pip install -e ".[dev]"
    "$PY" -m pytest
  fi
  pass "analytics-api"
}

gate_event_processor() {
  info "event-processor — cargo fmt --check + cargo test"
  cd "$ROOT/services/event-processor"
  cargo fmt --check
  cargo test
  pass "event-processor"
}

gate_agent_gateway() {
  info "agent-gateway — cargo fmt --check + cargo test"
  cd "$ROOT/services/agent-gateway"
  cargo fmt --check
  cargo test
  pass "agent-gateway"
}

for svc in $services; do
  dir="$ROOT/services/$svc"
  [ -d "$dir" ] || fail "unknown or missing service directory: services/$svc"

  case "$svc" in
    order-api) gate_order_api ;;
    catalog-api) gate_catalog_api ;;
    analytics-api) gate_analytics_api ;;
    event-processor) gate_event_processor ;;
    agent-gateway) gate_agent_gateway ;;
    storefront-ui|admin-ui)
      fail "$svc has no local factory gate yet (no CI-parity suite wired) — pick a backend service or add a gate"
      ;;
    *)
      fail "no gate recipe for service: $svc"
      ;;
  esac
done

cd "$ROOT"
echo "gate: all checks passed"
