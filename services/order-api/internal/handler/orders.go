package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/ianckc/distributed-systems/services/order-api/internal/events"
	"github.com/ianckc/distributed-systems/services/order-api/internal/model"
	"github.com/ianckc/distributed-systems/services/order-api/internal/store"
)

type OrderHandler struct {
	Store  store.OrderStore
	Events events.Publisher
}

type createOrderRequest struct {
	UserID string            `json:"user_id"`
	Items  []createOrderItem `json:"items"`
}

type createOrderItem struct {
	ProductID  string `json:"product_id"`
	Qty        int    `json:"qty"`
	PricePence int    `json:"price_pence"`
}

type createOrderResponse struct {
	ID         string            `json:"id"`
	UserID     string            `json:"user_id"`
	Status     string            `json:"status"`
	TotalPence int               `json:"total_pence"`
	Items      []model.OrderItem `json:"items"`
	CreatedAt  string            `json:"created_at"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createOrderRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}

	order, err := buildOrder(req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	created, err := h.Store.CreateOrder(r.Context(), order)
	if err != nil {
		slog.Error("failed to create order", "error", err)
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "failed to create order"})
		return
	}

	if h.Events != nil {
		pubCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := h.Events.PublishOrderCreated(pubCtx, created); err != nil {
			slog.Error("failed to publish order.created", "error", err, "order_id", created.ID)
		}
	}

	writeJSON(w, http.StatusCreated, createOrderResponse{
		ID:         created.ID.String(),
		UserID:     created.UserID.String(),
		Status:     created.Status,
		TotalPence: created.TotalPence,
		Items:      created.Items,
		CreatedAt:  created.CreatedAt.UTC().Format(time.RFC3339Nano),
	})
}

func buildOrder(req createOrderRequest) (model.Order, error) {
	if req.UserID == "" {
		return model.Order{}, errors.New("user_id is required")
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return model.Order{}, errors.New("user_id must be a valid UUID")
	}
	if len(req.Items) == 0 {
		return model.Order{}, errors.New("items must not be empty")
	}

	items := make([]model.OrderItem, 0, len(req.Items))
	for i, item := range req.Items {
		if item.ProductID == "" {
			return model.Order{}, fmt.Errorf("items[%d].product_id is required", i)
		}
		if item.Qty <= 0 {
			return model.Order{}, fmt.Errorf("items[%d].qty must be greater than 0", i)
		}
		if item.PricePence < 0 {
			return model.Order{}, fmt.Errorf("items[%d].price_pence must be >= 0", i)
		}
		items = append(items, model.OrderItem{
			ProductID:  item.ProductID,
			Qty:        item.Qty,
			PricePence: item.PricePence,
		})
	}

	order := model.Order{
		ID:     uuid.New(),
		UserID: userID,
		Status: model.StatusPending,
		Items:  items,
	}
	order.TotalPence = order.ComputeTotalPence()
	return order, nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
