package outbox

import (
	"context"
	"errors"
	"testing"
)

func TestPublishMessage(t *testing.T) {
	msg := Message{
		ID:         1,
		Topic:      "orders.events",
		MessageKey: "550e8400-e29b-41d4-a716-446655440000",
		Payload:    []byte(`{"event_type":"order.created"}`),
	}

	t.Run("publish success marks published", func(t *testing.T) {
		var marked bool
		processed, err := publishMessage(context.Background(), msg, func(_ context.Context, topic, key string, payload []byte) error {
			if topic != msg.Topic {
				t.Fatalf("topic = %q, want %q", topic, msg.Topic)
			}
			if key != msg.MessageKey {
				t.Fatalf("key = %q, want %q", key, msg.MessageKey)
			}
			if string(payload) != string(msg.Payload) {
				t.Fatalf("payload = %q, want %q", payload, msg.Payload)
			}
			return nil
		}, func(context.Context) error {
			marked = true
			return nil
		})
		if err != nil {
			t.Fatalf("publishMessage: %v", err)
		}
		if !processed {
			t.Fatal("expected processed")
		}
		if !marked {
			t.Fatal("expected published_at mark")
		}
	})

	t.Run("publish failure leaves row unpublished", func(t *testing.T) {
		markCalls := 0
		processed, err := publishMessage(context.Background(), msg, func(context.Context, string, string, []byte) error {
			return errors.New("broker down")
		}, func(context.Context) error {
			markCalls++
			return nil
		})
		if err != nil {
			t.Fatalf("publishMessage: %v", err)
		}
		if processed {
			t.Fatal("expected not processed")
		}
		if markCalls != 0 {
			t.Fatalf("mark called %d times, want 0", markCalls)
		}
	})

	t.Run("mark failure returns error", func(t *testing.T) {
		processed, err := publishMessage(context.Background(), msg, func(context.Context, string, string, []byte) error {
			return nil
		}, func(context.Context) error {
			return errors.New("update failed")
		})
		if err == nil {
			t.Fatal("expected error")
		}
		if processed {
			t.Fatal("expected not processed on mark failure")
		}
	})
}
