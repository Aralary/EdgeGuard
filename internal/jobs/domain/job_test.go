package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewWebhookPayloadDefaultsMethod(t *testing.T) {
	payload, err := NewWebhookPayload("https://example.com/hook", "", nil, nil)
	if err != nil {
		t.Fatalf("NewWebhookPayload() error = %v", err)
	}
	if payload.Method != "POST" {
		t.Fatalf("Method = %q, want POST", payload.Method)
	}
}

func TestNormalizeMaxAttempts(t *testing.T) {
	value, err := NormalizeMaxAttempts(0)
	if err != nil {
		t.Fatalf("NormalizeMaxAttempts() error = %v", err)
	}
	if value != DefaultMaxAttempts {
		t.Fatalf("value = %d, want %d", value, DefaultMaxAttempts)
	}

	_, err = NormalizeMaxAttempts(MaxMaxAttempts + 1)
	if !errors.Is(err, ErrInvalidMaxAttempts) {
		t.Fatalf("NormalizeMaxAttempts() error = %v, want %v", err, ErrInvalidMaxAttempts)
	}
}

func TestNewReportPayloadRejectsInvalidRange(t *testing.T) {
	now := time.Now()
	_, err := NewReportPayload("project", now, now, "json")
	if !errors.Is(err, ErrInvalidReportRange) {
		t.Fatalf("NewReportPayload() error = %v, want %v", err, ErrInvalidReportRange)
	}
}
