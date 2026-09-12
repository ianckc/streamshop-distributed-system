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

Default listen port in this service is **3010** (StreamShop Traefik uses host `:8080`).

Compose wiring and tool calling land in later plan chunks (A2+).
