package domain

import (
	"errors"
	"testing"
	"time"
)

func TestAPIKeyIsActive(t *testing.T) {
	now := time.Date(2026, time.July, 12, 12, 0, 0, 0, time.UTC)
	expiresAt := now.Add(time.Hour)

	key, err := NewAPIKey("project-id", "CI key", "eg_live_abcd", "key-hash", &expiresAt, now)
	if err != nil {
		t.Fatalf("NewAPIKey() error = %v", err)
	}

	if !key.IsActive(now) {
		t.Fatal("IsActive() = false, want true")
	}

	key.Enabled = false
	if key.IsActive(now) {
		t.Fatal("IsActive() = true for disabled key")
	}
}

func TestNewAPIKeyValidation(t *testing.T) {
	now := time.Date(2026, time.July, 12, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Minute)

	tests := []struct {
		name      string
		projectID string
		keyName   string
		keyPrefix string
		keyHash   string
		expiresAt *time.Time
		wantErr   error
	}{
		{
			name:      "missing project id",
			projectID: " ",
			keyName:   "CI key",
			keyPrefix: "prefix",
			keyHash:   "hash",
			wantErr:   ErrInvalidProjectID,
		},
		{
			name:      "missing name",
			projectID: "project-id",
			keyName:   " ",
			keyPrefix: "prefix",
			keyHash:   "hash",
			wantErr:   ErrInvalidName,
		},
		{
			name:      "missing prefix",
			projectID: "project-id",
			keyName:   "CI key",
			keyPrefix: " ",
			keyHash:   "hash",
			wantErr:   ErrInvalidKeyPrefix,
		},
		{
			name:      "missing hash",
			projectID: "project-id",
			keyName:   "CI key",
			keyPrefix: "prefix",
			keyHash:   " ",
			wantErr:   ErrInvalidKeyHash,
		},
		{
			name:      "expired key",
			projectID: "project-id",
			keyName:   "CI key",
			keyPrefix: "prefix",
			keyHash:   "hash",
			expiresAt: &past,
			wantErr:   ErrInvalidExpiration,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAPIKey(tt.projectID, tt.keyName, tt.keyPrefix, tt.keyHash, tt.expiresAt, now)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewAPIKey() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
