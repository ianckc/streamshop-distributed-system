package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port        string
	ServiceName string
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

	return Config{
		Port:        port,
		ServiceName: serviceName,
	}, nil
}

func (c Config) Addr() string {
	return fmt.Sprintf(":%s", c.Port)
}
