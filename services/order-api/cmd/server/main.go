package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ianckc/distributed-systems/services/order-api/internal/catalog"
	"github.com/ianckc/distributed-systems/services/order-api/internal/config"
	"github.com/ianckc/distributed-systems/services/order-api/internal/events"
	"github.com/ianckc/distributed-systems/services/order-api/internal/handler"
	"github.com/ianckc/distributed-systems/services/order-api/internal/idempotency"
	"github.com/ianckc/distributed-systems/services/order-api/internal/outbox"
	"github.com/ianckc/distributed-systems/services/order-api/internal/redisclient"
	"github.com/ianckc/distributed-systems/services/order-api/internal/store/postgres"
	"github.com/ianckc/distributed-systems/services/order-api/internal/telemetry"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	otelCtx, otelCancel := context.WithTimeout(context.Background(), 5*time.Second)
	otelShutdown, err := telemetry.Init(otelCtx, cfg.ServiceName)
	otelCancel()
	if err != nil {
		slog.Warn("otel init failed, continuing without telemetry", "error", err)
	}
	if otelShutdown != nil {
		defer func() {
			shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutCancel()
			if err := otelShutdown(shutCtx); err != nil {
				slog.Error("otel shutdown failed", "error", err)
			}
		}()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	pool, err := postgres.Connect(ctx, cfg.DatabaseURL)
	cancel()
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	orderStore := postgres.NewOrderStore(pool)
	publisher := events.NewKafkaPublisher(cfg.KafkaBrokers)
	defer func() {
		if err := publisher.Close(); err != nil {
			slog.Error("failed to close kafka publisher", "error", err)
		}
	}()

	outboxStore := outbox.NewPostgresStore(pool)
	pollerCtx, pollerCancel := context.WithCancel(context.Background())
	var pollerWG sync.WaitGroup
	pollerWG.Add(1)
	go func() {
		defer pollerWG.Done()
		(&outbox.Poller{
			Store:    outboxStore,
			Publish:  publisher.Publish,
			Interval: outbox.DefaultInterval,
		}).Run(pollerCtx)
	}()
	slog.Info("outbox poller started", "interval", outbox.DefaultInterval)

	orderHandler := handler.OrderHandler{Store: orderStore}

	if cfg.RedisURL != "" {
		redisCtx, redisCancel := context.WithTimeout(context.Background(), 5*time.Second)
		redisClient, err := redisclient.Connect(redisCtx, cfg.RedisURL)
		redisCancel()
		if err != nil {
			slog.Error("failed to connect to redis", "error", err)
			os.Exit(1)
		}
		defer func() {
			if err := redisClient.Close(); err != nil {
				slog.Error("failed to close redis", "error", err)
			}
		}()
		orderHandler.Idempotency = idempotency.NewStore(idempotency.NewRedisBackend(redisClient))
		slog.Info("idempotency keys enabled")
	}

	if cfg.CatalogAPIURL != "" {
		orderHandler.Catalog = catalog.NewClient(cfg.CatalogAPIURL, cfg.CatalogTimeout)
		slog.Info("catalog validation enabled", "url", cfg.CatalogAPIURL, "timeout", cfg.CatalogTimeout)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /health", handler.HealthHandler{ServiceName: cfg.ServiceName})
	mux.Handle("GET /ready", handler.ReadyHandler{ServiceName: cfg.ServiceName, Store: orderStore})
	mux.HandleFunc("POST /api/orders", orderHandler.Create)

	server := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      telemetry.HTTPMiddleware(mux),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	slog.Info("shutting down")
	pollerCancel()
	pollerWG.Wait()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown failed", "error", err)
		os.Exit(1)
	}
}
