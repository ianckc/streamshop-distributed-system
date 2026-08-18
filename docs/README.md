# StreamShop docs

Built with Docusaurus. Site content lives in `docs/docs/`.

## With Docker Compose (recommended)

```bash
docker compose up --build
```

Browse at [http://localhost:8080/docs/](http://localhost:8080/docs/) via Traefik.

## Local dev (live reload)

```bash
npm install
npm start
```

Opens at [http://localhost:3100/docs/](http://localhost:3100/docs/) so it does not clash with Grafana on `:3000`.

See the repo root [README](../README.md) for running the application.
