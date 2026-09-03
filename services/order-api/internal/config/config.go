package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Port           string
	ServiceName    string
	DatabaseURL    string
	KafkaBrokers   string
	CatalogAPIURL  string
	RedisURL       string
	CatalogTimeout time.Duration
	CBMaxRequests  uint32
	CBInterval     time.Duration
	CBTimeout      time.Duration
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

	cbMaxRequests := uint32(1)
	if v := os.Getenv("CB_MAX_REQUESTS"); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err != nil || n < 1 {
			return Config{}, fmt.Errorf("invalid CB_MAX_REQUESTS %q", v)
		}
		cbMaxRequests = uint32(n)
	}

	cbInterval := 30 * time.Second
	if v := os.Getenv("CB_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("invalid CB_INTERVAL %q: %w", v, err)
		}
		cbInterval = d
	}

	cbTimeout := 10 * time.Second
	if v := os.Getenv("CB_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("invalid CB_TIMEOUT %q: %w", v, err)
		}
		cbTimeout = d
	}

	return Config{
		Port:           port,
		ServiceName:    serviceName,
		DatabaseURL:    databaseURL,
		KafkaBrokers:   kafkaBrokers,
		CatalogAPIURL:  os.Getenv("CATALOG_API_URL"),
		RedisURL:       os.Getenv("REDIS_URL"),
		CatalogTimeout: catalogTimeout,
		CBMaxRequests:  cbMaxRequests,
		CBInterval:     cbInterval,
		CBTimeout:      cbTimeout,
	}, nil
}

func (c Config) Addr() string {
	return fmt.Sprintf(":%s", c.Port)
}
