# ADR 002: Redpanda over Apache Kafka

## Status

Accepted

## Context

StreamShop needs a message broker for the order event pipeline. The broker must support consumer groups, topic partitioning, offset management, and at-least-once delivery semantics.

Options considered:

- **Apache Kafka** (with KRaft)
- **Redpanda** (Kafka-compatible)
- **RabbitMQ** (AMQP)

## Decision

Use **Redpanda** with the Kafka API.

## Rationale

Redpanda provides full Kafka protocol compatibility in a single binary with no Zookeeper dependency. On a laptop:

| | Redpanda | Kafka + KRaft |
|--|----------|---------------|
| RAM | ~512 MB–1 GB | 2–4 GB+ |
| Setup | One container | Multi-container |
| CLI | `rpk` built-in | Separate tooling |
| Industry relevance | Kafka API (same clients) | Native Kafka |

RabbitMQ was rejected because AMQP is less representative of modern event streaming pipelines that employers expect to see.

## Consequences

- **Positive:** kafka-go and rust-rdkafka work without modification; lower local resource use
- **Negative:** Redpanda-specific admin features differ slightly from Confluent Kafka
- **Mitigation:** All client code uses standard Kafka APIs; broker is swappable
