package events

import (
	"context"
	"time"

	"github.com/ianckc/distributed-systems/services/order-api/internal/model"
)

const (
	TopicOrdersEvents     = "orders.events"
	EventTypeOrderCreated = "order.created"
)

type Publisher interface {
	PublishOrderCreated(ctx context.Context, order model.Order) error
}

type OrderCreated struct {
	EventType  string            `json:"event_type"`
	OrderID    string            `json:"order_id"`
	UserID     string            `json:"user_id"`
	Items      []model.OrderItem `json:"items"`
	TotalPence int               `json:"total_pence"`
	CreatedAt  string            `json:"created_at"`
}

func NewOrderCreated(order model.Order) OrderCreated {
	return OrderCreated{
		EventType:  EventTypeOrderCreated,
		OrderID:    order.ID.String(),
		UserID:     order.UserID.String(),
		Items:      order.Items,
		TotalPence: order.TotalPence,
		CreatedAt:  order.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}
