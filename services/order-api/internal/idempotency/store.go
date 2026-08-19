package idempotency

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	TTL       = time.Hour
	keyPrefix = "idempotency:"
)

type State string

const (
	StatePending  State = "pending"
	StateComplete State = "complete"
)

var (
	ErrEmptyKey = errors.New("idempotency key is empty")
	ErrNotFound = errors.New("idempotency key not found")
)

type Record struct {
	State   State     `json:"state"`
	OrderID uuid.UUID `json:"order_id,omitempty"`
}

type Backend interface {
	SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error)
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
}

type Store struct {
	backend Backend
}

func NewStore(backend Backend) *Store {
	return &Store{backend: backend}
}

func redisKey(key string) string {
	return keyPrefix + key
}

func (s *Store) Begin(ctx context.Context, key string) (acquired bool, existing Record, err error) {
	if key == "" {
		return false, Record{}, ErrEmptyKey
	}

	rk := redisKey(key)
	pending, err := marshalRecord(Record{State: StatePending})
	if err != nil {
		return false, Record{}, err
	}

	ok, err := s.backend.SetNX(ctx, rk, pending, TTL)
	if err != nil {
		return false, Record{}, fmt.Errorf("reserve idempotency key: %w", err)
	}
	if ok {
		return true, Record{State: StatePending}, nil
	}

	existing, err = s.Get(ctx, key)
	if err != nil {
		return false, Record{}, err
	}
	return false, existing, nil
}

func (s *Store) Complete(ctx context.Context, key string, orderID uuid.UUID) error {
	if key == "" {
		return ErrEmptyKey
	}

	payload, err := marshalRecord(Record{State: StateComplete, OrderID: orderID})
	if err != nil {
		return err
	}
	if err := s.backend.Set(ctx, redisKey(key), payload, TTL); err != nil {
		return fmt.Errorf("complete idempotency key: %w", err)
	}
	return nil
}

func (s *Store) Get(ctx context.Context, key string) (Record, error) {
	if key == "" {
		return Record{}, ErrEmptyKey
	}

	raw, err := s.backend.Get(ctx, redisKey(key))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Record{}, ErrNotFound
		}
		return Record{}, fmt.Errorf("get idempotency key: %w", err)
	}
	return unmarshalRecord(raw)
}

func marshalRecord(rec Record) (string, error) {
	body := recordJSON{State: string(rec.State)}
	if rec.State == StateComplete && rec.OrderID != uuid.Nil {
		body.OrderID = rec.OrderID.String()
	}
	b, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("encode idempotency record: %w", err)
	}
	return string(b), nil
}

func unmarshalRecord(raw string) (Record, error) {
	var body recordJSON
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		return Record{}, fmt.Errorf("decode idempotency record: %w", err)
	}
	rec := Record{State: State(body.State)}
	switch rec.State {
	case StatePending:
		return rec, nil
	case StateComplete:
		if body.OrderID == "" {
			return Record{}, fmt.Errorf("decode idempotency record: complete record missing order_id")
		}
		id, err := uuid.Parse(body.OrderID)
		if err != nil {
			return Record{}, fmt.Errorf("decode idempotency record: %w", err)
		}
		rec.OrderID = id
		return rec, nil
	default:
		return Record{}, fmt.Errorf("decode idempotency record: unknown state %q", body.State)
	}
}

type recordJSON struct {
	State   string `json:"state"`
	OrderID string `json:"order_id,omitempty"`
}
