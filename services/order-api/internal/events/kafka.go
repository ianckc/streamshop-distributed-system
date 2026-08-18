package events

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/ianckc/distributed-systems/services/order-api/internal/model"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
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
	tracer := otel.Tracer("order-api")
	ctx, span := tracer.Start(ctx, "kafka.publish order.created",
		trace.WithSpanKind(trace.SpanKindProducer),
	)
	defer span.End()
	span.SetAttributes(
		semconv.MessagingSystemKey.String("kafka"),
		semconv.MessagingDestinationNameKey.String(TopicOrdersEvents),
		attribute.String("order.id", order.ID.String()),
	)

	event := NewOrderCreated(order)
	value, err := json.Marshal(event)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	headers := make([]kafka.Header, 0, len(carrier))
	for key, val := range carrier {
		headers = append(headers, kafka.Header{Key: key, Value: []byte(val)})
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:     []byte(event.OrderID),
		Value:   value,
		Headers: headers,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
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
