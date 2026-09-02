//go:build integration

package outbox_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ianckc/distributed-systems/services/order-api/internal/events"
	"github.com/ianckc/distributed-systems/services/order-api/internal/outbox"
	"github.com/ianckc/distributed-systems/services/order-api/internal/store/postgres"
)

func TestPostgresStoreProcessNext(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if databaseURL == "" || kafkaBrokers == "" {
		t.Skip("DATABASE_URL and KAFKA_BROKERS required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	publisher := events.NewKafkaPublisher(kafkaBrokers)
	defer publisher.Close()

	store := outbox.NewPostgresStore(pool)

	publish := func(ctx context.Context, topic, key string, payload []byte) error {
		err := publisher.Publish(ctx, topic, key, payload)
		t.Logf("publish topic=%q key=%q payload_len=%d err=%v", topic, key, len(payload), err)
		return err
	}

	processed, err := store.ProcessNext(ctx, publish)
	t.Logf("processed=%v err=%v", processed, err)
	if err != nil {
		t.Fatalf("ProcessNext: %v", err)
	}
	if !processed {
		t.Fatal("expected a row to be processed")
	}
}
