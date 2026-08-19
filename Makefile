.PHONY: up down seed smoke logs

GATEWAY ?= http://localhost:8080

up:
	docker compose up --build -d --wait
	@echo "StreamShop is up at $(GATEWAY) (docs: $(GATEWAY)/docs/)"

down:
	docker compose down

seed:
	./scripts/seed.sh

smoke:
	./scripts/smoke-test.sh

logs:
	docker compose logs -f
