.PHONY: up down seed smoke agent-smoke logs

GATEWAY ?= http://localhost:8080

up:
	docker compose up --build -d --wait
	@echo "StreamShop is up at $(GATEWAY) (docs: $(GATEWAY)/docs/, admin agent: $(GATEWAY)/admin/)"

down:
	docker compose down

seed:
	./scripts/seed.sh

smoke:
	./scripts/smoke-test.sh

agent-smoke:
	./scripts/agent-smoke.sh

logs:
	docker compose logs -f
