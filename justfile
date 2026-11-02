# StreamShop local commands — https://github.com/casey/just
# Run `just` to list recipes.

export GATEWAY := env_var_or_default("GATEWAY", "http://localhost:8080")
# Comma-separated service names, or `all`. Ticket runs should pass affected services.
services := env_var_or_default("SERVICES", "all")

default:
    @just --list

# Build and start the stack (waits until healthy)
up:
    docker compose up --build -d --wait
    @echo "StreamShop is up at {{GATEWAY}} (docs: {{GATEWAY}}/docs/, admin agent: {{GATEWAY}}/admin/)"

# Stop the stack
down:
    docker compose down

# Seed topics, catalog, sample images, Redis flush
seed:
    ./scripts/seed.sh

# End-to-end happy path through Traefik
smoke:
    ./scripts/smoke-test.sh

# Ops agent: gateway health, /admin/, one SSE turn
agent-smoke:
    ./scripts/agent-smoke.sh

# CI-parity factory gate (SERVICES env or: just gate order-api,catalog-api)
gate services=services:
    SERVICES="{{services}}" ./scripts/factory-gate.sh

# Follow compose logs
logs:
    docker compose logs -f
