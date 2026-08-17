# Build stage
FROM rust:1-bookworm AS builder

RUN apt-get update \
    && apt-get install -y --no-install-recommends cmake g++ libssl-dev libsasl2-dev zlib1g-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src
COPY Cargo.toml Cargo.lock ./
COPY src ./src
RUN cargo build --release --locked --features kafka

# Runtime stage
FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates libssl3 libsasl2-2 wget \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /src/target/release/event-processor /app/event-processor

ENV PORT=3004
ENV SERVICE_NAME=event-processor
EXPOSE 3004

USER nobody
CMD ["./event-processor"]
