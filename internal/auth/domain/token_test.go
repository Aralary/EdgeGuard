package domain

import (
	"errors"
	"testing"
	"time"
)

func TestRefreshTokenIsActive(t *testing.T) {
	now := time.Date(2026, time.July, 12, 12, 0, 0, 0, time.UTC)
	token, err := NewRefreshToken("user-id", "token-hash", now.Add(time.Hour), now)
	if err != nil {
		t.Fatalf("NewRefreshToken() error = %v", err)
	}

	if !token.IsActive(now) {
		t.Fatal("IsActive() = false, want true")
	}

	revokedAt := now
	token.RevokedAt = &revokedAt
	if token.IsActive(now) {
		t.Fatal("IsActive() = true for revoked token")
	}
}

func TestNewRefreshTokenRejectsExpiredToken(t *testing.T) {
	now := time.Date(2026, time.July, 12, 12, 0, 0, 0, time.UTC)

	_, err := NewRefreshToken("user-id", "token-hash", now, now)
	if !errors.Is(err, ErrInvalidExpiration) {
		t.Fatalf("NewRefreshToken() error = %v, want %v", err, ErrInvalidExpiration)
	}
}
