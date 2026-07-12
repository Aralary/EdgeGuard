package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("GATEWAY_CONFIG_PATH", "")
	t.Setenv("CONTROL_PLANE_URL", "")
	t.Setenv("ROUTES_REFRESH_INTERVAL", "")
	t.Setenv("AUTH_SERVICE_URL", "")
	t.Setenv("REDIS_URL", "")
	t.Setenv("REDIS_RATE_LIMIT_PREFIX", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.ConfigPath != defaultConfigPath {
		t.Fatalf("ConfigPath = %q, want %q", cfg.ConfigPath, defaultConfigPath)
	}
	if cfg.RoutesRefreshInterval != defaultRoutesRefreshInterval {
		t.Fatalf("RoutesRefreshInterval = %s, want %s", cfg.RoutesRefreshInterval, defaultRoutesRefreshInterval)
	}
}

func TestLoadEnvironment(t *testing.T) {
	t.Setenv("GATEWAY_CONFIG_PATH", "custom.yaml")
	t.Setenv("CONTROL_PLANE_URL", " http://control-plane:8082 ")
	t.Setenv("ROUTES_REFRESH_INTERVAL", "30s")
	t.Setenv("AUTH_SERVICE_URL", " http://auth:8083 ")
	t.Setenv("REDIS_URL", " redis://redis:6379/0 ")
	t.Setenv("REDIS_RATE_LIMIT_PREFIX", " custom-prefix ")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.ConfigPath != "custom.yaml" {
		t.Fatalf("ConfigPath = %q", cfg.ConfigPath)
	}
	if cfg.ControlPlaneURL != "http://control-plane:8082" {
		t.Fatalf("ControlPlaneURL = %q", cfg.ControlPlaneURL)
	}
	if cfg.RoutesRefreshInterval != 30*time.Second {
		t.Fatalf("RoutesRefreshInterval = %s", cfg.RoutesRefreshInterval)
	}
	if cfg.AuthServiceURL != "http://auth:8083" {
		t.Fatalf("AuthServiceURL = %q", cfg.AuthServiceURL)
	}
	if cfg.RedisURL != "redis://redis:6379/0" {
		t.Fatalf("RedisURL = %q", cfg.RedisURL)
	}
	if cfg.RedisRateLimitPrefix != "custom-prefix" {
		t.Fatalf("RedisRateLimitPrefix = %q", cfg.RedisRateLimitPrefix)
	}
}

func TestLoadRejectsInvalidRefreshInterval(t *testing.T) {
	t.Setenv("ROUTES_REFRESH_INTERVAL", "0s")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}
