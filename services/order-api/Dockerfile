# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /order-api ./cmd/server

# Runtime stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /order-api .

ENV PORT=3002
ENV SERVICE_NAME=order-api
# DATABASE_URL is provided by Compose at runtime
EXPOSE 3002

USER nobody

CMD ["./order-api"]
