# ADR 003: Traefik and nginx for gateway and load balancing

## Status

Accepted

## Context

StreamShop needs both an API gateway (routing, TLS, middleware) and internal load balancing across service replicas. These are distinct concerns that are often conflated.

## Decision

Use **Traefik v3** as the edge API gateway and **nginx** as an internal load balancer for `catalog-api` replicas.

## Rationale

Demonstrating two layers teaches an important production distinction:

1. **Edge gateway (Traefik)** — authentication, rate limiting, path routing, TLS termination, external entrypoint
2. **Internal LB (nginx)** — distribute traffic across replicas of a single service

Traefik integrates with Docker Compose via container labels, eliminating manual config reloads. nginx is universally understood and its upstream/block configuration is a common interview topic.

Kong was considered but rejected for local use due to heavier setup and database dependency for its admin API.

## Consequences

- **Positive:** Two recognizable patterns; Traefik dashboard aids debugging; nginx config is portable to any environment
- **Negative:** Two hop points add latency (~1ms locally, negligible)
- **Mitigation:** Docs include sequence diagram showing both layers explicitly
