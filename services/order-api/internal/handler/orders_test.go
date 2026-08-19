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
	"github.com/ianckc/distributed-systems/services/order-api/internal/idempotency"
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

type fakeIdempotency struct {
	acquired    bool
	existing    idempotency.Record
	beginErr    error
	completeErr error
	begins      int
	completes   []uuid.UUID
}

func (f *fakeIdempotency) Begin(_ context.Context, _ string) (bool, idempotency.Record, error) {
	f.begins++
	if f.beginErr != nil {
		return false, idempotency.Record{}, f.beginErr
	}
	return f.acquired, f.existing, nil
}

func (f *fakeIdempotency) Complete(_ context.Context, _ string, orderID uuid.UUID) error {
	f.completes = append(f.completes, orderID)
	return f.completeErr
}

const validOrderBody = `{
	"user_id": "660e8400-e29b-41d4-a716-446655440001",
	"items": [{"product_id": "prod-001", "qty": 2, "price_pence": 1999}]
}`

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

func TestOrderHandlerIdempotency(t *testing.T) {
	createdAt := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	userID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")
	stored := model.Order{
		ID:         orderID,
		UserID:     userID,
		Status:     model.StatusPending,
		TotalPence: 3998,
		Items:      []model.OrderItem{{ProductID: "prod-001", Qty: 2, PricePence: 1999}},
		CreatedAt:  createdAt,
	}

	t.Run("first request with key creates order", func(t *testing.T) {
		keys := &fakeIdempotency{acquired: true}
		creates := 0
		h := handler.OrderHandler{
			Store: fakeOrderStore{
				createFn: func(_ context.Context, order model.Order) (model.Order, error) {
					creates++
					order.CreatedAt = createdAt
					return order, nil
				},
			},
			Idempotency: keys,
		}

		req := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewBufferString(validOrderBody))
		req.Header.Set("Idempotency-Key", "req-1")
		rec := httptest.NewRecorder()
		h.Create(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusCreated, rec.Body.String())
		}
		if creates != 1 {
			t.Fatalf("creates = %d, want 1", creates)
		}
		if keys.begins != 1 {
			t.Fatalf("begins = %d, want 1", keys.begins)
		}
		if len(keys.completes) != 1 {
			t.Fatalf("completes = %d, want 1", len(keys.completes))
		}
	})

	t.Run("replay returns original order", func(t *testing.T) {
		pub := &recordingPublisher{}
		h := handler.OrderHandler{
			Store: fakeOrderStore{
				createFn: func(context.Context, model.Order) (model.Order, error) {
					t.Fatal("store should not create on replay")
					return model.Order{}, nil
				},
				getFn: func(_ context.Context, id uuid.UUID) (model.Order, error) {
					if id != orderID {
						t.Fatalf("GetOrder id = %s, want %s", id, orderID)
					}
					return stored, nil
				},
			},
			Events: pub,
			Idempotency: &fakeIdempotency{
				acquired: false,
				existing: idempotency.Record{State: idempotency.StateComplete, OrderID: orderID},
			},
		}

		req := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewBufferString(validOrderBody))
		req.Header.Set("Idempotency-Key", "req-1")
		rec := httptest.NewRecorder()
		h.Create(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
		}
		if pub.called {
			t.Fatal("replay should not publish")
		}
		var got map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got["id"] != orderID.String() {
			t.Fatalf("id = %v, want %s", got["id"], orderID)
		}
	})

	t.Run("in-flight key returns 409", func(t *testing.T) {
		h := handler.OrderHandler{
			Store: fakeOrderStore{
				createFn: func(context.Context, model.Order) (model.Order, error) {
					t.Fatal("store should not create while in progress")
					return model.Order{}, nil
				},
			},
			Idempotency: &fakeIdempotency{
				acquired: false,
				existing: idempotency.Record{State: idempotency.StatePending},
			},
		}

		req := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewBufferString(validOrderBody))
		req.Header.Set("Idempotency-Key", "req-1")
		rec := httptest.NewRecorder()
		h.Create(rec, req)

		if rec.Code != http.StatusConflict {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusConflict, rec.Body.String())
		}
	})

	t.Run("redis error returns 503", func(t *testing.T) {
		h := handler.OrderHandler{
			Store: fakeOrderStore{
				createFn: func(context.Context, model.Order) (model.Order, error) {
					t.Fatal("store should not create when redis fails")
					return model.Order{}, nil
				},
			},
			Idempotency: &fakeIdempotency{beginErr: errors.New("redis down")},
		}

		req := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewBufferString(validOrderBody))
		req.Header.Set("Idempotency-Key", "req-1")
		rec := httptest.NewRecorder()
		h.Create(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
		}
		var body struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body.Error != "idempotency unavailable" {
			t.Fatalf("error = %q", body.Error)
		}
	})

	t.Run("header without store returns 503", func(t *testing.T) {
		h := handler.OrderHandler{
			Store: fakeOrderStore{
				createFn: func(context.Context, model.Order) (model.Order, error) {
					t.Fatal("store should not create when idempotency is unset")
					return model.Order{}, nil
				},
			},
		}

		req := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewBufferString(validOrderBody))
		req.Header.Set("Idempotency-Key", "req-1")
		rec := httptest.NewRecorder()
		h.Create(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
		}
	})

	t.Run("missing header still creates without keys", func(t *testing.T) {
		h := handler.OrderHandler{
			Store: fakeOrderStore{
				createFn: func(_ context.Context, order model.Order) (model.Order, error) {
					order.CreatedAt = createdAt
					return order, nil
				},
			},
		}

		req := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewBufferString(validOrderBody))
		rec := httptest.NewRecorder()
		h.Create(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusCreated, rec.Body.String())
		}
	})
}
