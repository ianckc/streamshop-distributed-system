# StreamShop storefront

Public StreamShop UI — shopping assistant chat today; browse, purchase, and
accounts later.

## Local development

```bash
npm ci
npm run dev
# http://localhost:3030/  (proxies /v1 → http://127.0.0.1:8080)
```

## Production (Compose)

`make up` builds this image and routes:

| URL | Purpose |
|-----|---------|
| http://localhost:8080/ | Storefront chat (this service) |
| http://localhost:8080/admin/ | Ops admin chat (admin-ui) |
| http://localhost:8080/v1/agent/stream | SSE (agent-gateway) |

Traefik uses `PathPrefix(/)` with **priority 1** so `/admin`, `/api/*`, `/v1/agent`, and `/docs` keep their routers.

Image layout: nginx `:80` (`/health`, `/assets/*`, proxy to SSR) + srvx `:3030`.
Build arg: `STOREFRONT_UI_BASE=/`
