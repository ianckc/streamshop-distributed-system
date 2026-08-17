# StreamShop

Local-first distributed systems showcase — polyglot microservices, event streaming, and multi-store architecture.

**Documentation:** run the docs site (`cd docs && npm start`) or see [PLAN.md](./PLAN.md) for the roadmap.

```bash
docker compose up --build
```

Traefik is the HTTP entrypoint on **:8080**. `order-api` is also published on **:3002** for local debugging. Dashboard: [http://localhost:8081](http://localhost:8081).

```bash
curl http://localhost:8080/api/orders   # via Traefik (POST with a JSON body)
curl http://localhost:3002/health       # direct to the service
```
