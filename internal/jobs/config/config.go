package config

import (
	"errors"
	"os"
	"strings"
	"time"
)

const (
	defaultRabbitMQURL    = "amqp://guest:guest@localhost:5672/"
	defaultExchange       = "edgeguard.jobs.v1"
	defaultQueue          = "edgeguard.jobs.main.v1"
	defaultPublishTimeout = 5 * time.Second
)

var ErrInvalidPublishTimeout = errors.New("invalid jobs publish timeout")

type Config struct {
	RabbitMQURL    string
	Exchange       string
	Queue          string
	PublishTimeout time.Duration
}

func Load() (Config, error) {
	publishTimeout := defaultPublishTimeout
	if value := strings.TrimSpace(os.Getenv("JOBS_PUBLISH_TIMEOUT")); value != "" {
		parsed, err := time.ParseDuration(value)
		if err != nil || parsed <= 0 {
			return Config{}, ErrInvalidPublishTimeout
		}
		publishTimeout = parsed
	}

	return Config{
		RabbitMQURL:    envOrDefault("RABBITMQ_URL", defaultRabbitMQURL),
		Exchange:       envOrDefault("JOBS_EXCHANGE", defaultExchange),
		Queue:          envOrDefault("JOBS_QUEUE", defaultQueue),
		PublishTimeout: publishTimeout,
	}, nil
}

func envOrDefault(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}

	return value
}
