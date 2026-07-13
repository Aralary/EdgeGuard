package config

import (
	"errors"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("JOBS_WORKER_PREFETCH", "")
	t.Setenv("JOBS_RETRY_MIN", "")
	t.Setenv("JOBS_RETRY_MAX", "")

	config, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if config.Prefetch != defaultPrefetch {
		t.Fatalf("Prefetch = %d, want %d", config.Prefetch, defaultPrefetch)
	}
	if config.RetryMin != 5*time.Second || config.RetryMax != time.Minute {
		t.Fatalf("retry range = %s..%s", config.RetryMin, config.RetryMax)
	}
}

func TestLoadRejectsRetryMaxBelowMin(t *testing.T) {
	t.Setenv("JOBS_RETRY_MIN", "10s")
	t.Setenv("JOBS_RETRY_MAX", "5s")

	_, err := Load()
	if !errors.Is(err, ErrInvalidRetryMax) {
		t.Fatalf("Load() error = %v, want %v", err, ErrInvalidRetryMax)
	}
}

func TestLoadAllowedHosts(t *testing.T) {
	t.Setenv("JOBS_WEBHOOK_ALLOWED_HOSTS", " demo-backend, EXAMPLE.COM ")

	config, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(config.WebhookAllowedHosts) != 2 || config.WebhookAllowedHosts[0] != "demo-backend" || config.WebhookAllowedHosts[1] != "example.com" {
		t.Fatalf("WebhookAllowedHosts = %#v", config.WebhookAllowedHosts)
	}
}
