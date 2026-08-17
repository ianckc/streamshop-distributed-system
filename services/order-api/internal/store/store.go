package store

import (
	"context"

	"github.com/ianckc/distributed-systems/services/order-api/internal/model"
)

type OrderStore interface {
	CreateOrder(ctx context.Context, order model.Order) (model.Order, error)
	Ping(ctx context.Context) error
}
