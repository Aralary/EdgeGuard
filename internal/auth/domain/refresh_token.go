package domain

import (
	"strings"
	"time"
)

type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

func NewRefreshToken(userID string, tokenHash string, expiresAt time.Time, now time.Time) (RefreshToken, error) {
	userID = strings.TrimSpace(userID)
	tokenHash = strings.TrimSpace(tokenHash)

	if userID == "" {
		return RefreshToken{}, ErrInvalidUserID
	}

	if tokenHash == "" {
		return RefreshToken{}, ErrInvalidTokenHash
	}

	if !expiresAt.After(now) {
		return RefreshToken{}, ErrInvalidExpiration
	}

	return RefreshToken{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}, nil
}

func (t RefreshToken) IsActive(now time.Time) bool {
	return t.RevokedAt == nil && now.Before(t.ExpiresAt)
}
