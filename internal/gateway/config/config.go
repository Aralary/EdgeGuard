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
)

type Config struct {
	ConfigPath            string
	ControlPlaneURL       string
	RoutesRefreshInterval time.Duration
	AuthServiceURL        string
	RedisURL              string
	RedisRateLimitPrefix  string
}

func Load() (Config, error) {
	cfg := Config{
		ConfigPath:            strings.TrimSpace(os.Getenv("GATEWAY_CONFIG_PATH")),
		ControlPlaneURL:       strings.TrimSpace(os.Getenv("CONTROL_PLANE_URL")),
		RoutesRefreshInterval: defaultRoutesRefreshInterval,
		AuthServiceURL:        strings.TrimSpace(os.Getenv("AUTH_SERVICE_URL")),
		RedisURL:              strings.TrimSpace(os.Getenv("REDIS_URL")),
		RedisRateLimitPrefix:  strings.TrimSpace(os.Getenv("REDIS_RATE_LIMIT_PREFIX")),
	}

	if cfg.ConfigPath == "" {
		cfg.ConfigPath = defaultConfigPath
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
