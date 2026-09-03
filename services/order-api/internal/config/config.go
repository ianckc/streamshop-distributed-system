package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Port          string
	ServiceName   string
	DatabaseURL   string
	KafkaBrokers  string
	CatalogAPIURL string
	RedisURL       string
	CatalogTimeout time.Duration
}

func Load() (Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3002"
	}

	serviceName := os.Getenv("SERVICE_NAME")
	if serviceName == "" {
		serviceName = "order-api"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		return Config{}, fmt.Errorf("KAFKA_BROKERS is required")
	}

	catalogTimeout := 2 * time.Second
	if v := os.Getenv("CATALOG_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("invalid CATALOG_TIMEOUT %q: %w", v, err)
		}
		catalogTimeout = d
	}

	return Config{
		Port:           port,
		ServiceName:    serviceName,
		DatabaseURL:    databaseURL,
		KafkaBrokers:   kafkaBrokers,
		CatalogAPIURL:  os.Getenv("CATALOG_API_URL"),
		RedisURL:       os.Getenv("REDIS_URL"),
		CatalogTimeout: catalogTimeout,
	}, nil
}

func (c Config) Addr() string {
	return fmt.Sprintf(":%s", c.Port)
}
