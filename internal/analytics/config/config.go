package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultKafkaTopic         = "edgeguard.gateway.access.v1"
	defaultKafkaConsumerGroup = "edgeguard-analytics-v1"
	defaultKafkaClientID      = "edgeguard-analytics-worker"
	defaultRetryMin           = time.Second
	defaultRetryMax           = 30 * time.Second
	defaultMaxProcessAttempts = 5
)

type Config struct {
	KafkaBrokers       []string
	KafkaTopic         string
	KafkaConsumerGroup string
	KafkaClientID      string
	RetryMin           time.Duration
	RetryMax           time.Duration
	MaxProcessAttempts int
}

func Load() (Config, error) {
	brokers := splitAndNormalize(os.Getenv("KAFKA_BROKERS"))
	if len(brokers) == 0 {
		return Config{}, fmt.Errorf("KAFKA_BROKERS is required")
	}

	retryMin, err := durationFromEnv("ANALYTICS_RETRY_MIN", defaultRetryMin)
	if err != nil {
		return Config{}, err
	}

	retryMax, err := durationFromEnv("ANALYTICS_RETRY_MAX", defaultRetryMax)
	if err != nil {
		return Config{}, err
	}
	if retryMax < retryMin {
		return Config{}, fmt.Errorf("ANALYTICS_RETRY_MAX must be greater than or equal to ANALYTICS_RETRY_MIN")
	}

	maxProcessAttempts, err := intFromEnv("ANALYTICS_MAX_PROCESS_ATTEMPTS", defaultMaxProcessAttempts)
	if err != nil {
		return Config{}, err
	}
	if maxProcessAttempts <= 0 {
		return Config{}, fmt.Errorf("ANALYTICS_MAX_PROCESS_ATTEMPTS must be positive")
	}

	return Config{
		KafkaBrokers:       brokers,
		KafkaTopic:         stringFromEnv("KAFKA_ACCESS_TOPIC", defaultKafkaTopic),
		KafkaConsumerGroup: stringFromEnv("ANALYTICS_KAFKA_GROUP", defaultKafkaConsumerGroup),
		KafkaClientID:      stringFromEnv("ANALYTICS_KAFKA_CLIENT_ID", defaultKafkaClientID),
		RetryMin:           retryMin,
		RetryMax:           retryMax,
		MaxProcessAttempts: maxProcessAttempts,
	}, nil
}

func stringFromEnv(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

func durationFromEnv(key string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be positive", key)
	}

	return parsed, nil
}

func splitAndNormalize(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))

	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}

		seen[item] = struct{}{}
		result = append(result, item)
	}

	return result
}

func intFromEnv(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}

	return parsed, nil
}
