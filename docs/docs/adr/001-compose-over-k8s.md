# ADR 001: Docker Compose over local Kubernetes

## Status

Accepted

## Context

StreamShop must run entirely on a developer laptop for demos and interviews. We need an orchestration layer that supports multiple services, networks, health checks, and profiles without cloud infrastructure costs.

Options considered:

- **Docker Compose** — declarative multi-container orchestration
- **k3d / minikube** — lightweight local Kubernetes clusters
- **Hybrid** — Compose for infra, k8s for services

## Decision

Use **Docker Compose** as the sole orchestration tool.

## Rationale

| Factor | Compose | Local k8s |
|--------|---------|-----------|
| Bootstrap time | Seconds | Minutes |
| RAM overhead | Minimal | +2–4 GB for control plane |
| Learning curve for reviewers | Low | Requires k8s context |
| Service discovery | Docker DNS | CoreDNS |
| Config style | Labels + YAML | Deployments, Services, Ingress |

Employers reviewing a portfolio repo want to run `docker compose up` and see results immediately. Kubernetes YAML adds volume without adding domain value for a local-only demo.

## Consequences

- **Positive:** One-command startup, familiar to most engineers, easy Traefik integration via labels
- **Negative:** Does not demonstrate k8s-specific patterns (Helm, operators, pod autoscaling)
- **Mitigation:** ADR documents the trade-off; production deployment section in docs explains k8s migration path
