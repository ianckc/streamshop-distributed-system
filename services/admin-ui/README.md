# StreamShop Ops / SRE admin chat UI (TanStack Start)

Vendored from `rust-ai-agent-gateway/frontend/`, served at **`/admin`** behind Traefik.

## Local development

```bash
npm ci
npm run dev
# http://localhost:3020/admin/  (proxies /v1 → http://127.0.0.1:8080)
```

Requires StreamShop Traefik (or agent-gateway) on `:8080` for SSE.

## Production (Compose)

`make up` builds this image and routes:

| URL | Purpose |
|-----|---------|
| http://localhost:8080/admin/ | Admin chat UI |
| http://localhost:8080/v1/agent/stream | SSE (agent-gateway) |

Tool call/result events render inline as monospace lines in the transcript.

## How assets are served

The image runs **nginx on :80** (Traefik target) in front of **srvx on :3020** (SSR):

- `/admin/assets/*` → files from `dist/client/assets/` (avoids an srvx + `base: /admin/` 404)
- `/admin` → proxied to TanStack Start for HTML/SSR
- `/health` → nginx liveness for Compose
