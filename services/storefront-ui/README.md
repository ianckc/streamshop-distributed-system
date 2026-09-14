# StreamShop storefront UI (TanStack Start)

Public-facing UI scaffold (eventually browse, purchase, accounts). Served at
**`/`** behind Traefik. Pattern matches `admin-ui`.

## Local development

```bash
npm ci
npm run dev
# http://localhost:3030/  (proxies /v1 → http://127.0.0.1:8080)
```

## Production (Compose)

Wired in a later step. Image layout:

| Piece | Role |
|-------|------|
| nginx `:80` | `/health`, `/assets/*`, proxy to SSR |
| srvx `:3030` | TanStack Start SSR |

Build arg: `STOREFRONT_UI_BASE=/`
