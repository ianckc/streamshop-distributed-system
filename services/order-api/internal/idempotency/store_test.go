package idempotency

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

type memBackend struct {
	mu   sync.Mutex
	data map[string]string
	ttls map[string]time.Duration
	err  error
}

func newMemBackend() *memBackend {
	return &memBackend{
		data: make(map[string]string),
		ttls: make(map[string]time.Duration),
	}
}

func (m *memBackend) SetNX(_ context.Context, key, value string, ttl time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return false, m.err
	}
	if _, exists := m.data[key]; exists {
		return false, nil
	}
	m.data[key] = value
	m.ttls[key] = ttl
	return true, nil
}

func (m *memBackend) Get(_ context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return "", m.err
	}
	val, ok := m.data[key]
	if !ok {
		return "", ErrNotFound
	}
	return val, nil
}

func (m *memBackend) Set(_ context.Context, key, value string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.data[key] = value
	m.ttls[key] = ttl
	return nil
}

func TestBeginAcquiresPendingKey(t *testing.T) {
	backend := newMemBackend()
	store := NewStore(backend)
	ctx := context.Background()

	acquired, rec, err := store.Begin(ctx, "req-1")
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if !acquired {
		t.Fatal("expected to acquire key")
	}
	if rec.State != StatePending {
		t.Fatalf("state = %q, want pending", rec.State)
	}

	raw, ok := backend.data["idempotency:req-1"]
	if !ok {
		t.Fatal("expected redis key idempotency:req-1")
	}
	if backend.ttls["idempotency:req-1"] != TTL {
		t.Fatalf("ttl = %s, want %s", backend.ttls["idempotency:req-1"], TTL)
	}
	got, err := unmarshalRecord(raw)
	if err != nil {
		t.Fatalf("stored value: %v", err)
	}
	if got.State != StatePending {
		t.Fatalf("stored state = %q", got.State)
	}
}

func TestBeginReturnsExistingPending(t *testing.T) {
	store := NewStore(newMemBackend())
	ctx := context.Background()

	if _, _, err := store.Begin(ctx, "req-1"); err != nil {
		t.Fatalf("first Begin: %v", err)
	}
	acquired, rec, err := store.Begin(ctx, "req-1")
	if err != nil {
		t.Fatalf("second Begin: %v", err)
	}
	if acquired {
		t.Fatal("second Begin should not acquire")
	}
	if rec.State != StatePending {
		t.Fatalf("state = %q, want pending", rec.State)
	}
}

func TestCompleteThenBeginReturnsOriginalOrderID(t *testing.T) {
	store := NewStore(newMemBackend())
	ctx := context.Background()
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	if _, _, err := store.Begin(ctx, "req-1"); err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if err := store.Complete(ctx, "req-1", orderID); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	acquired, rec, err := store.Begin(ctx, "req-1")
	if err != nil {
		t.Fatalf("replay Begin: %v", err)
	}
	if acquired {
		t.Fatal("replay should not acquire")
	}
	if rec.State != StateComplete {
		t.Fatalf("state = %q, want complete", rec.State)
	}
	if rec.OrderID != orderID {
		t.Fatalf("order_id = %s, want %s", rec.OrderID, orderID)
	}
}

func TestGetCompleteRecord(t *testing.T) {
	store := NewStore(newMemBackend())
	ctx := context.Background()
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	if _, _, err := store.Begin(ctx, "req-1"); err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if err := store.Complete(ctx, "req-1", orderID); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	rec, err := store.Get(ctx, "req-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if rec.State != StateComplete || rec.OrderID != orderID {
		t.Fatalf("record = %+v", rec)
	}
}

func TestGetMissingKey(t *testing.T) {
	_, err := NewStore(newMemBackend()).Get(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestEmptyKey(t *testing.T) {
	store := NewStore(newMemBackend())
	ctx := context.Background()

	if _, _, err := store.Begin(ctx, ""); !errors.Is(err, ErrEmptyKey) {
		t.Fatalf("Begin err = %v, want ErrEmptyKey", err)
	}
	if err := store.Complete(ctx, "", uuid.New()); !errors.Is(err, ErrEmptyKey) {
		t.Fatalf("Complete err = %v, want ErrEmptyKey", err)
	}
	if _, err := store.Get(ctx, ""); !errors.Is(err, ErrEmptyKey) {
		t.Fatalf("Get err = %v, want ErrEmptyKey", err)
	}
}

func TestBackendErrorsPropagate(t *testing.T) {
	backend := newMemBackend()
	backend.err = errors.New("redis down")
	store := NewStore(backend)
	ctx := context.Background()

	if _, _, err := store.Begin(ctx, "req-1"); err == nil {
		t.Fatal("expected Begin error")
	}
	if err := store.Complete(ctx, "req-1", uuid.New()); err == nil {
		t.Fatal("expected Complete error")
	}
}

func TestRejectsUnknownState(t *testing.T) {
	backend := newMemBackend()
	backend.data["idempotency:req-1"] = `{"state":"weird"}`
	_, err := NewStore(backend).Get(context.Background(), "req-1")
	if err == nil {
		t.Fatal("expected decode error")
	}
}
