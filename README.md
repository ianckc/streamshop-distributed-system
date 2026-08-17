# StreamShop

Local-first distributed systems showcase — polyglot microservices, event streaming, and multi-store architecture.

**Documentation:** run the docs site (`cd docs && npm start`) or see [PLAN.md](./PLAN.md) for the roadmap.

```bash
docker compose up --build
```

Traefik is the HTTP entrypoint on **:8080**. Services are also published directly for local debugging. Traefik dashboard: [http://localhost:8081](http://localhost:8081). Redis Insight: [http://localhost:5540](http://localhost:5540).

```bash
curl http://localhost:3001/health                 # catalog-api liveness
curl http://localhost:3001/ready                  # catalog-api Mongo ping
curl http://localhost:3002/health                 # order-api
curl http://localhost:8080/api/catalog/products           # via Traefik (Mongo)
curl http://localhost:8080/api/catalog/products/prod-001  # via Traefik (Mongo)
curl http://localhost:8080/api/orders             # via Traefik (POST with a JSON body)
```
