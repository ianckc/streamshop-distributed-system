#!/usr/bin/env sh
# Phase A ops-agent smoke: admin UI + gateway health + one SSE turn.
set -eu

GATEWAY="${GATEWAY:-http://localhost:8080}"
AGENT_HEALTH="${AGENT_HEALTH_URL:-http://localhost:3010/health}"
SESSION_ID="${AGENT_SMOKE_SESSION:-agent-smoke-1}"
MAX_SECS="${AGENT_SMOKE_TIMEOUT_SECS:-90}"

fail() {
  echo "agent-smoke: FAIL: $1" >&2
  exit 1
}

pass() {
  echo "agent-smoke: ok: $1"
}

hint() {
  echo "hint: make up (LLM: set LLM_* in .env; Ollama on host or Groq/OpenRouter)" >&2
}

if ! curl -sS -o /dev/null --connect-timeout 2 "$GATEWAY/api/analytics/orders/summary" 2>/dev/null; then
  echo "agent-smoke: cannot reach $GATEWAY (analytics)" >&2
  hint
  exit 1
fi
pass "analytics reachable via Traefik"

if ! curl -sS -f -o /dev/null --connect-timeout 2 "$AGENT_HEALTH"; then
  fail "agent-gateway health at $AGENT_HEALTH"
fi
pass "agent-gateway /health"

admin_code="$(curl -sS -o /dev/null -w '%{http_code}' --connect-timeout 2 "$GATEWAY/admin/" || true)"
if [ "$admin_code" != "200" ]; then
  fail "admin UI $GATEWAY/admin/ returned HTTP $admin_code (expected 200)"
fi
pass "admin UI /admin/"

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

# Capture SSE for up to MAX_SECS; do not fail curl on partial read (stream may not close).
set +e
curl -sS -N --max-time "$MAX_SECS" \
  -X POST "$GATEWAY/v1/agent/stream" \
  -H 'Content-Type: application/json' \
  -H 'Accept: text/event-stream' \
  -d "{\"session_id\":\"$SESSION_ID\",\"message\":\"How are orders looking? Use get_orders_summary.\"}" \
  >"$tmp" 2>/dev/null
curl_ec=$?
set -e

# 28 = operation timeout (expected if the server keeps the stream open briefly)
if [ "$curl_ec" -ne 0 ] && [ "$curl_ec" -ne 28 ]; then
  fail "SSE request failed (curl exit $curl_ec). Is LLM_BASE_URL reachable from agent-gateway?"
fi

if ! grep -q 'turn_start\|TurnStart\|token_chunk\|TokenChunk\|tool_call\|ToolCall\|tool_result\|ToolResult' "$tmp"; then
  echo "agent-smoke: SSE body (first 800 chars):" >&2
  head -c 800 "$tmp" >&2 || true
  echo >&2
  fail "no agent SSE events seen (LLM misconfigured or gateway error?)"
fi

pass "SSE turn produced agent events (session=$SESSION_ID)"
echo "agent-smoke: PASS"
echo "agent-smoke: UI → $GATEWAY/admin/"
