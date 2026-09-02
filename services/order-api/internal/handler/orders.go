package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ianckc/distributed-systems/services/order-api/internal/catalog"
	"github.com/ianckc/distributed-systems/services/order-api/internal/idempotency"
	"github.com/ianckc/distributed-systems/services/order-api/internal/model"
	"github.com/ianckc/distributed-systems/services/order-api/internal/store"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type IdempotencyStore interface {
	Begin(ctx context.Context, key string) (acquired bool, existing idempotency.Record, err error)
	Complete(ctx context.Context, key string, orderID uuid.UUID) error
}

type OrderHandler struct {
	Store       store.OrderStore
	Catalog     catalog.ProductGetter
	Idempotency IdempotencyStore
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
	ctx := r.Context()
	tracer := otel.Tracer("order-api")
	ctx, span := tracer.Start(ctx, "orders.create")
	defer span.End()

	var req createOrderRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}

	if h.Catalog != nil {
		if err := validateProducts(ctx, h.Catalog, req.Items); err != nil {
			var validationErr *validationError
			if errors.As(err, &validationErr) {
				writeJSON(w, http.StatusBadRequest, errorResponse{Error: validationErr.Error()})
				return
			}
			slog.ErrorContext(ctx, "catalog validation failed", "error", err)
			writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "catalog unavailable"})
			return
		}
	}

	order, err := buildOrder(req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey != "" {
		if h.Idempotency == nil {
			writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "idempotency unavailable"})
			return
		}
		acquired, existing, err := h.Idempotency.Begin(ctx, idempotencyKey)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			slog.ErrorContext(ctx, "idempotency begin failed", "error", err)
			writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "idempotency unavailable"})
			return
		}
		if !acquired {
			h.replayOrConflict(ctx, w, span, existing)
			return
		}
	}

	span.SetAttributes(attribute.String("order.id", order.ID.String()))

	created, err := h.Store.CreateOrder(ctx, order)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		slog.ErrorContext(ctx, "failed to create order", "error", err)
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "failed to create order"})
		return
	}

	if idempotencyKey != "" {
		if err := h.Idempotency.Complete(ctx, idempotencyKey, created.ID); err != nil {
			span.RecordError(err)
			slog.ErrorContext(ctx, "idempotency complete failed", "error", err, "order_id", created.ID)
		}
	}

	writeJSON(w, http.StatusCreated, orderResponse(created))
}

func (h OrderHandler) replayOrConflict(ctx context.Context, w http.ResponseWriter, span trace.Span, existing idempotency.Record) {
	if existing.State == idempotency.StatePending {
		writeJSON(w, http.StatusConflict, errorResponse{Error: "idempotency key in progress"})
		return
	}
	if existing.State != idempotency.StateComplete {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "idempotency unavailable"})
		return
	}

	span.SetAttributes(
		attribute.Bool("idempotency.replay", true),
		attribute.String("order.id", existing.OrderID.String()),
	)
	order, err := h.Store.GetOrder(ctx, existing.OrderID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		slog.ErrorContext(ctx, "idempotency replay failed", "error", err, "order_id", existing.OrderID)
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "failed to load order"})
		return
	}
	writeJSON(w, http.StatusOK, orderResponse(order))
}

func orderResponse(order model.Order) createOrderResponse {
	return createOrderResponse{
		ID:         order.ID.String(),
		UserID:     order.UserID.String(),
		Status:     order.Status,
		TotalPence: order.TotalPence,
		Items:      order.Items,
		CreatedAt:  order.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

type validationError struct {
	msg string
}

func (e *validationError) Error() string { return e.msg }

func validateProducts(ctx context.Context, catalogClient catalog.ProductGetter, items []createOrderItem) error {
	tracer := otel.Tracer("order-api")
	ctx, span := tracer.Start(ctx, "orders.validate_products")
	defer span.End()
	span.SetAttributes(attribute.Int("order.item_count", len(items)))

	for i, item := range items {
		product, err := catalogClient.GetProduct(ctx, item.ProductID)
		if err != nil {
			if errors.Is(err, catalog.ErrProductNotFound) {
				msg := fmt.Sprintf("product %q not found", item.ProductID)
				err := &validationError{msg: msg}
				span.RecordError(err)
				span.SetStatus(codes.Error, msg)
				return err
			}
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return fmt.Errorf("catalog lookup %q: %w", item.ProductID, err)
		}
		if item.PricePence != product.PricePence {
			msg := fmt.Sprintf("items[%d].price_pence does not match catalog price for %q", i, item.ProductID)
			err := &validationError{msg: msg}
			span.RecordError(err)
			span.SetStatus(codes.Error, msg)
			return err
		}
	}
	return nil
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
