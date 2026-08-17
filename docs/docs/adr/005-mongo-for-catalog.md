# ADR 005: MongoDB for product catalog

## Status

Accepted

## Context

Products in StreamShop have variable attributes — colour, size, material, weight — that differ per category. A running shoe has sizes; a book has ISBN and author. Forcing these into rigid relational columns leads to sparse tables or excessive EAV patterns.

Options considered:

- **MongoDB** — document store with flexible schema
- **PostgreSQL JSONB** — relational store with JSON column
- **Separate Postgres tables per category** — normalized approach

## Decision

Use **MongoDB 7** for the product catalog.

## Rationale

MongoDB documents naturally represent products with variable `attributes` maps. This creates a clear datastore ownership boundary:

- MongoDB → catalog (flexible, read-heavy)
- PostgreSQL → orders (transactional, write-heavy)

PostgreSQL JSONB would work but obscures the polyglot storage story. Separate tables per category does not scale with category count.

## Consequences

- **Positive:** Schema flexibility without migrations; clear NoSQL vs SQL demonstration
- **Negative:** No ACID joins between products and orders (by design — services communicate via IDs)
- **Mitigation:** Product IDs are referenced as strings in Postgres order_items; catalog-api is the source of truth for product data
