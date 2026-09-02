package outbox

import "context"

// Message is one unpublished (or in-flight) outbox row.
type Message struct {
	ID         int64
	Topic      string
	MessageKey string
	Payload    []byte
}

// PublishFunc sends a stored outbox payload to the message broker.
type PublishFunc func(ctx context.Context, topic, key string, payload []byte) error
