package outbox

import (
	"context"
	"log/slog"
)

// Store claims unpublished outbox rows and marks them published after delivery.
type Store interface {
	ProcessNext(ctx context.Context, publish PublishFunc) (processed bool, err error)
}

// publishMessage publishes then marks the row. Returns false without error when
// publish fails so the caller can roll back and retry later.
func publishMessage(ctx context.Context, msg Message, publish PublishFunc, markPublished func(context.Context) error) (bool, error) {
	if err := publish(ctx, msg.Topic, msg.MessageKey, msg.Payload); err != nil {
		slog.WarnContext(ctx, "outbox publish failed, will retry",
			"error", err,
			"outbox_id", msg.ID,
			"topic", msg.Topic,
		)
		return false, nil
	}
	if err := markPublished(ctx); err != nil {
		return false, err
	}
	return true, nil
}
