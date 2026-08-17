package model

import (
	"time"

	"github.com/google/uuid"
)

const StatusPending = "pending"

type OrderItem struct {
	ProductID  string `json:"product_id"`
	Qty        int    `json:"qty"`
	PricePence int    `json:"price_pence"`
}

type Order struct {
	ID         uuid.UUID   `json:"id"`
	UserID     uuid.UUID   `json:"user_id"`
	Status     string      `json:"status"`
	TotalPence int         `json:"total_pence"`
	Items      []OrderItem `json:"items"`
	CreatedAt  time.Time   `json:"created_at"`
}

func (o Order) ComputeTotalPence() int {
	total := 0
	for _, item := range o.Items {
		total += item.Qty * item.PricePence
	}
	return total
}
