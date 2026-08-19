package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ianckc/distributed-systems/services/order-api/internal/events"
	"github.com/ianckc/distributed-systems/services/order-api/internal/handler"
	"github.com/ianckc/distributed-systems/services/order-api/internal/model"
	"github.com/ianckc/distributed-systems/services/order-api/internal/store"
)

type fakeOrderStore struct {
	createFn func(ctx context.Context, order model.Order) (model.Order, error)
	getFn    func(ctx context.Context, id uuid.UUID) (model.Order, error)
	pingFn   func(ctx context.Context) error
}

func (f fakeOrderStore) CreateOrder(ctx context.Context, order model.Order) (model.Order, error) {
	return f.createFn(ctx, order)
}

func (f fakeOrderStore) GetOrder(ctx context.Context, id uuid.UUID) (model.Order, error) {
	if f.getFn != nil {
		return f.getFn(ctx, id)
	}
	return model.Order{}, store.ErrNotFound
}

func (f fakeOrderStore) Ping(ctx context.Context) error {
	if f.pingFn != nil {
		return f.pingFn(ctx)
	}
	return nil
}

type recordingPublisher struct {
	orders []model.Order
	err    error
	called bool
}

func (p *recordingPublisher) PublishOrderCreated(_ context.Context, order model.Order) error {
	p.called = true
	p.orders = append(p.orders, order)
	return p.err
}

func TestOrderHandlerCreate(t *testing.T) {
	userID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")

	t.Run("creates order", func(t *testing.T) {
		h := handler.OrderHandler{
			Store: fakeOrderStore{
				createFn: func(_ context.Context, order model.Order) (model.Order, error) {
					order.CreatedAt = time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
					return order, nil
				},
			},
		}

		body := `{
			"user_id": "660e8400-e29b-41d4-a716-446655440001",
			"items": [{"product_id": "prod-001", "qty": 2, "price_pence": 1999}]
		}`
		req := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()

		h.Create(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusCreated, rec.Body.String())
		}

		var got map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got["status"] != "pending" {
			t.Fatalf("status = %v, want pending", got["status"])
		}
		if got["total_pence"] != float64(3998) {
			t.Fatalf("total_pence = %v, want 3998", got["total_pence"])
		}
		if got["user_id"] != userID.String() {
			t.Fatalf("user_id = %v", got["user_id"])
		}
		if got["id"] == "" {
			t.Fatal("expected order id")
		}
	})

	t.Run("rejects invalid body", func(t *testing.T) {
		h := handler.OrderHandler{
			Store: fakeOrderStore{
				createFn: func(context.Context, model.Order) (model.Order, error) {
					t.Fatal("store should not be called")
					return model.Order{}, nil
				},
			},
		}

		req := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewBufferString(`{"user_id":"not-a-uuid","items":[]}`))
		rec := httptest.NewRecorder()

		h.Create(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("returns 503 when store fails", func(t *testing.T) {
		h := handler.OrderHandler{
			Store: fakeOrderStore{
				createFn: func(context.Context, model.Order) (model.Order, error) {
					return model.Order{}, errors.New("db down")
				},
			},
		}

		body := `{
			"user_id": "660e8400-e29b-41d4-a716-446655440001",
			"items": [{"product_id": "prod-001", "qty": 1, "price_pence": 999}]
		}`
		req := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()

		h.Create(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
		}
	})

	t.Run("publishes order.created after persist", func(t *testing.T) {
		pub := &recordingPublisher{}
		createdAt := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
		h := handler.OrderHandler{
			Store: fakeOrderStore{
				createFn: func(_ context.Context, order model.Order) (model.Order, error) {
					order.CreatedAt = createdAt
					return order, nil
				},
			},
			Events: pub,
		}

		body := `{
			"user_id": "660e8400-e29b-41d4-a716-446655440001",
			"items": [{"product_id": "prod-001", "qty": 2, "price_pence": 1999}]
		}`
		req := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()

		h.Create(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusCreated, rec.Body.String())
		}
		if !pub.called {
			t.Fatal("expected publish")
		}
		if len(pub.orders) != 1 {
			t.Fatalf("published %d events, want 1", len(pub.orders))
		}
		got := events.NewOrderCreated(pub.orders[0])
		if got.EventType != events.EventTypeOrderCreated {
			t.Fatalf("event_type = %q", got.EventType)
		}
		if got.TotalPence != 3998 {
			t.Fatalf("total_pence = %d", got.TotalPence)
		}
		if got.UserID != userID.String() {
			t.Fatalf("user_id = %q", got.UserID)
		}
	})

	t.Run("returns 201 when publish fails", func(t *testing.T) {
		h := handler.OrderHandler{
			Store: fakeOrderStore{
				createFn: func(_ context.Context, order model.Order) (model.Order, error) {
					order.CreatedAt = time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
					return order, nil
				},
			},
			Events: &recordingPublisher{err: errors.New("redpanda down")},
		}

		body := `{
			"user_id": "660e8400-e29b-41d4-a716-446655440001",
			"items": [{"product_id": "prod-001", "qty": 1, "price_pence": 999}]
		}`
		req := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()

		h.Create(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusCreated, rec.Body.String())
		}
	})

	t.Run("does not publish when store fails", func(t *testing.T) {
		pub := &recordingPublisher{}
		h := handler.OrderHandler{
			Store: fakeOrderStore{
				createFn: func(context.Context, model.Order) (model.Order, error) {
					return model.Order{}, errors.New("db down")
				},
			},
			Events: pub,
		}

		body := `{
			"user_id": "660e8400-e29b-41d4-a716-446655440001",
			"items": [{"product_id": "prod-001", "qty": 1, "price_pence": 999}]
		}`
		req := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()

		h.Create(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
		}
		if pub.called {
			t.Fatal("should not publish when persist fails")
		}
	})
}
