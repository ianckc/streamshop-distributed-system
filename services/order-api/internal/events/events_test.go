package events

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ianckc/distributed-systems/services/order-api/internal/model"
)

func TestNewOrderCreated(t *testing.T) {
	order := model.Order{
		ID:     uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		UserID: uuid.MustParse("660e8400-e29b-41d4-a716-446655440001"),
		Status: model.StatusPending,
		Items: []model.OrderItem{
			{ProductID: "prod-001", Qty: 2, PricePence: 1999},
		},
		TotalPence: 3998,
		CreatedAt:  time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC),
	}

	got := NewOrderCreated(order)
	if got.EventType != EventTypeOrderCreated {
		t.Fatalf("event_type = %q", got.EventType)
	}
	if got.OrderID != order.ID.String() {
		t.Fatalf("order_id = %q", got.OrderID)
	}
	if got.CreatedAt != "2026-08-12T10:00:00Z" {
		t.Fatalf("created_at = %q", got.CreatedAt)
	}

	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded["event_type"] != "order.created" {
		t.Fatalf("json event_type = %v", decoded["event_type"])
	}
	if decoded["total_pence"] != float64(3998) {
		t.Fatalf("json total_pence = %v", decoded["total_pence"])
	}
}

func TestSplitBrokers(t *testing.T) {
	got := splitBrokers(" redpanda:9092 , localhost:19092 ")
	if len(got) != 2 || got[0] != "redpanda:9092" || got[1] != "localhost:19092" {
		t.Fatalf("splitBrokers = %#v", got)
	}
}
