# agent-gateway

Rust SSE agent gateway (ops / SRE copilot). Vendored from `rust-ai-agent-gateway/` for StreamShop.

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Liveness |
| POST | `/v1/agent/stream` | SSE agent turn (`session_id` + `message`) |

## Local development

```bash
cp .env.example .env
# Point REDIS_URL / LLM_* at your local Redis and Ollama (or other provider)
cargo run
```

`LLM_MODEL` selects the chat-completions model (default `qwen3:8b`). Phase A tool schemas (`get_order`, `get_orders_summary`, `check_service_health`) are advertised to the LLM; HTTP handlers land in A4.

Default listen port is **3010** (StreamShop Traefik owns host `:8080`).

## Compose / Traefik

With `make up`, Traefik routes:

| URL | Target |
|-----|--------|
| `http://localhost:8080/v1/agent/stream` | SSE agent turns |
| `http://localhost:3010/health` | Direct health (debug port) |

```bash
curl -s http://localhost:8080/v1/agent/stream -o /dev/null -w '%{http_code}\n' \
  -H 'Content-Type: application/json' \
  -d '{"session_id":"smoke","message":"ping"}'
# Expect 200 once LLM + Redis are reachable (tools still stubbed until A4)
```

LLM defaults assume Ollama on the host (`host.docker.internal:11434`). Override `LLM_*` in root `.env` or `services/agent-gateway/.env`.
