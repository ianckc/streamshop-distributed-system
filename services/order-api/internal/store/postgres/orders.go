package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/ianckc/distributed-systems/services/order-api/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type OrderStore struct {
	pool *pgxpool.Pool
}

func NewOrderStore(pool *pgxpool.Pool) *OrderStore {
	return &OrderStore{pool: pool}
}

func (s *OrderStore) CreateOrder(ctx context.Context, order model.Order) (model.Order, error) {
	tracer := otel.Tracer("order-api")
	ctx, span := tracer.Start(ctx, "postgres.create_order")
	defer span.End()
	span.SetAttributes(
		semconv.DBSystemKey.String("postgresql"),
		attribute.String("db.operation.name", "insert"),
		attribute.String("order.id", order.ID.String()),
	)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Order{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var createdAt time.Time
	err = tx.QueryRow(ctx, `
		INSERT INTO orders (id, user_id, status, total_pence)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at
	`, order.ID, order.UserID, order.Status, order.TotalPence).Scan(&createdAt)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Order{}, fmt.Errorf("insert order: %w", err)
	}

	_, itemSpan := tracer.Start(ctx, "postgres.insert_order_items")
	itemSpan.SetAttributes(
		semconv.DBSystemKey.String("postgresql"),
		attribute.Int("order.item_count", len(order.Items)),
	)
	batch := &pgx.Batch{}
	for _, item := range order.Items {
		batch.Queue(`
			INSERT INTO order_items (order_id, product_id, qty, price_pence)
			VALUES ($1, $2, $3, $4)
		`, order.ID, item.ProductID, item.Qty, item.PricePence)
	}

	br := tx.SendBatch(ctx, batch)
	if err := br.Close(); err != nil {
		itemSpan.RecordError(err)
		itemSpan.SetStatus(codes.Error, err.Error())
		itemSpan.End()
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Order{}, fmt.Errorf("insert order items: %w", err)
	}
	itemSpan.End()

	if err := tx.Commit(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return model.Order{}, fmt.Errorf("commit tx: %w", err)
	}

	order.CreatedAt = createdAt
	return order, nil
}

func (s *OrderStore) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}
