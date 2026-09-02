package outbox

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) ProcessNext(ctx context.Context, publish PublishFunc) (bool, error) {
	tracer := otel.Tracer("order-api")
	ctx, span := tracer.Start(ctx, "outbox.process_next")
	defer span.End()
	span.SetAttributes(semconv.DBSystemKey.String("postgresql"))

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return false, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var msg Message
	err = tx.QueryRow(ctx, `
		SELECT id, topic, message_key, payload
		FROM outbox
		WHERE published_at IS NULL
		ORDER BY id
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`).Scan(&msg.ID, &msg.Topic, &msg.MessageKey, &msg.Payload)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return false, fmt.Errorf("claim outbox row: %w", err)
	}
	span.SetAttributes(
		attribute.Int64("outbox.id", msg.ID),
		attribute.String("messaging.destination.name", msg.Topic),
	)

	markPublished := func(ctx context.Context) error {
		_, err := tx.Exec(ctx, `
			UPDATE outbox SET published_at = now() WHERE id = $1
		`, msg.ID)
		return err
	}

	processed, err := publishMessage(ctx, msg, publish, markPublished)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return false, fmt.Errorf("mark published: %w", err)
	}
	if !processed {
		return false, nil
	}

	if err := tx.Commit(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return false, fmt.Errorf("commit tx: %w", err)
	}
	return true, nil
}
