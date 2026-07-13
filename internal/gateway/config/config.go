package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	defaultConfigPath            = "configs/gateway.yaml"
	defaultRoutesRefreshInterval = 10 * time.Second
	defaultKafkaAccessTopic      = "edgeguard.gateway.access.v1"
	defaultKafkaClientID         = "edgeguard-gateway"
)

type Config struct {
	ConfigPath            string
	ControlPlaneURL       string
	RoutesRefreshInterval time.Duration
	AuthServiceURL        string
	RedisURL              string
	RedisRateLimitPrefix  string
	KafkaBrokers          []string
	KafkaAccessTopic      string
	KafkaClientID         string
}

func Load() (Config, error) {
	cfg := Config{
		ConfigPath:            strings.TrimSpace(os.Getenv("GATEWAY_CONFIG_PATH")),
		ControlPlaneURL:       strings.TrimSpace(os.Getenv("CONTROL_PLANE_URL")),
		RoutesRefreshInterval: defaultRoutesRefreshInterval,
		AuthServiceURL:        strings.TrimSpace(os.Getenv("AUTH_SERVICE_URL")),
		RedisURL:              strings.TrimSpace(os.Getenv("REDIS_URL")),
		RedisRateLimitPrefix:  strings.TrimSpace(os.Getenv("REDIS_RATE_LIMIT_PREFIX")),
		KafkaBrokers:          splitCSV(os.Getenv("KAFKA_BROKERS")),
		KafkaAccessTopic:      strings.TrimSpace(os.Getenv("KAFKA_ACCESS_TOPIC")),
		KafkaClientID:         strings.TrimSpace(os.Getenv("KAFKA_CLIENT_ID")),
	}

	if cfg.ConfigPath == "" {
		cfg.ConfigPath = defaultConfigPath
	}
	if cfg.KafkaAccessTopic == "" {
		cfg.KafkaAccessTopic = defaultKafkaAccessTopic
	}
	if cfg.KafkaClientID == "" {
		cfg.KafkaClientID = defaultKafkaClientID
	}

	if rawInterval := strings.TrimSpace(os.Getenv("ROUTES_REFRESH_INTERVAL")); rawInterval != "" {
		interval, err := time.ParseDuration(rawInterval)
		if err != nil {
			return Config{}, fmt.Errorf("parse ROUTES_REFRESH_INTERVAL: %w", err)
		}
		if interval <= 0 {
			return Config{}, errors.New("ROUTES_REFRESH_INTERVAL must be positive")
		}

		cfg.RoutesRefreshInterval = interval
	}

	return cfg, nil
}

func splitCSV(value string) []string {
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
