# StreamShop storefront

Public StreamShop UI — shopping assistant chat today; browse, purchase, and
accounts later. Served at **`/`** behind Traefik.

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
