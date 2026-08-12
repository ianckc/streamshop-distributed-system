# ADR 004: ClickHouse for OLAP

## Status

Accepted

## Context

StreamShop needs an analytics store for order event aggregates — hourly revenue, order counts by period, top products. OLTP queries against PostgreSQL will not scale for analytical workloads.

Options considered:

- **ClickHouse** — columnar OLAP database
- **DuckDB** — embedded analytical database
- **PostgreSQL with materialized views** — reuse existing store

## Decision

Use **ClickHouse 24** for all analytical queries.

## Rationale

ClickHouse is the industry standard for real-time analytics at scale. It demonstrates understanding of OLTP vs OLAP separation — a core distributed systems concept.

| | ClickHouse | DuckDB | Postgres MVs |
|--|------------|--------|--------------|
| Columnar storage | Yes | Yes | No |
| Separate service | Yes | Embedded | No |
| Industry recognition | High | Growing | Low for analytics |
| Local RAM | ~1 GB | Minimal | Shared with OLTP |

DuckDB is noted as the fallback for the `minimal` Compose profile on RAM-constrained laptops.

## Consequences

- **Positive:** Fast aggregations; clear OLTP/OLAP boundary; impressive in demos
- **Negative:** Additional container (~1 GB RAM); separate schema to maintain
- **Mitigation:** event-processor owns all ClickHouse writes; analytics-api is read-only
