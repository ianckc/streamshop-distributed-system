package events

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/ianckc/distributed-systems/services/order-api/internal/model"
	"github.com/segmentio/kafka-go"
)

type KafkaPublisher struct {
	writer *kafka.Writer
}

func NewKafkaPublisher(brokersCSV string) *KafkaPublisher {
	return &KafkaPublisher{
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(splitBrokers(brokersCSV)...),
			Topic:                  TopicOrdersEvents,
			Balancer:               &kafka.Hash{},
			RequiredAcks:           kafka.RequireOne,
			AllowAutoTopicCreation: true,
		},
	}
}

func (p *KafkaPublisher) PublishOrderCreated(ctx context.Context, order model.Order) error {
	event := NewOrderCreated(order)
	value, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.OrderID),
		Value: value,
	})
}

func (p *KafkaPublisher) Close() error {
	return p.writer.Close()
}

func splitBrokers(csv string) []string {
	parts := strings.Split(csv, ",")
	brokers := make([]string, 0, len(parts))
	for _, part := range parts {
		broker := strings.TrimSpace(part)
		if broker != "" {
			brokers = append(brokers, broker)
		}
	}
	return brokers
}
