package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port          string
	ServiceName   string
	DatabaseURL   string
	KafkaBrokers  string
	CatalogAPIURL string
	RedisURL      string
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

	return Config{
		Port:          port,
		ServiceName:   serviceName,
		DatabaseURL:   databaseURL,
		KafkaBrokers:  kafkaBrokers,
		CatalogAPIURL: os.Getenv("CATALOG_API_URL"),
		RedisURL:      os.Getenv("REDIS_URL"),
	}, nil
}

func (c Config) Addr() string {
	return fmt.Sprintf(":%s", c.Port)
}
