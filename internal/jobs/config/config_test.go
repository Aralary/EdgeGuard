package config

import (
	"errors"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("RABBITMQ_URL", "")
	t.Setenv("JOBS_EXCHANGE", "")
	t.Setenv("JOBS_QUEUE", "")
	t.Setenv("JOBS_PUBLISH_TIMEOUT", "")

	config, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if config.PublishTimeout != 5*time.Second {
		t.Fatalf("PublishTimeout = %s, want 5s", config.PublishTimeout)
	}
}

func TestLoadRejectsInvalidTimeout(t *testing.T) {
	t.Setenv("JOBS_PUBLISH_TIMEOUT", "0s")

	_, err := Load()
	if !errors.Is(err, ErrInvalidPublishTimeout) {
		t.Fatalf("Load() error = %v, want %v", err, ErrInvalidPublishTimeout)
	}
}
