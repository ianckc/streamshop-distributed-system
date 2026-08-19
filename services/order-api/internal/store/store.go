package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/ianckc/distributed-systems/services/order-api/internal/model"
)

var ErrNotFound = errors.New("order not found")

type OrderStore interface {
	CreateOrder(ctx context.Context, order model.Order) (model.Order, error)
	GetOrder(ctx context.Context, id uuid.UUID) (model.Order, error)
	Ping(ctx context.Context) error
}
