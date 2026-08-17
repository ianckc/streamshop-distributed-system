package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/ianckc/distributed-systems/services/order-api/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderStore struct {
	pool *pgxpool.Pool
}

func NewOrderStore(pool *pgxpool.Pool) *OrderStore {
	return &OrderStore{pool: pool}
}

func (s *OrderStore) CreateOrder(ctx context.Context, order model.Order) (model.Order, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
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
		return model.Order{}, fmt.Errorf("insert order: %w", err)
	}

	batch := &pgx.Batch{}
	for _, item := range order.Items {
		batch.Queue(`
			INSERT INTO order_items (order_id, product_id, qty, price_pence)
			VALUES ($1, $2, $3, $4)
		`, order.ID, item.ProductID, item.Qty, item.PricePence)
	}

	br := tx.SendBatch(ctx, batch)
	if err := br.Close(); err != nil {
		return model.Order{}, fmt.Errorf("insert order items: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return model.Order{}, fmt.Errorf("commit tx: %w", err)
	}

	order.CreatedAt = createdAt
	return order, nil
}
