package telemetry

import (
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func HTTPMiddleware(next http.Handler) http.Handler {
	tracer := otel.Tracer("order-api")
	meter := otel.Meter("order-api")

	duration, _ := meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithUnit("s"),
	)
	requestCount, _ := meter.Int64Counter(
		"http.server.request.count",
	)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagationCarrier(r.Header))
		spanName := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		ctx, span := tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()

		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(sw, r.WithContext(ctx))
		elapsed := time.Since(start).Seconds()

		attrs := []attribute.KeyValue{
			semconv.HTTPRequestMethodKey.String(r.Method),
			semconv.HTTPRouteKey.String(r.URL.Path),
			semconv.HTTPResponseStatusCodeKey.Int(sw.status),
		}
		duration.Record(ctx, elapsed, metric.WithAttributes(attrs...))
		requestCount.Add(ctx, 1, metric.WithAttributes(attrs...))

		span.SetAttributes(attrs...)
	})
}

type propagationCarrier http.Header

func (c propagationCarrier) Get(key string) string {
	return http.Header(c).Get(key)
}

func (c propagationCarrier) Set(key, value string) {
	http.Header(c).Set(key, value)
}

func (c propagationCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}
