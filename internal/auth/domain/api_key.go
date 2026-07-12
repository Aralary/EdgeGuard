package domain

import (
	"strings"
	"time"
)

type APIKey struct {
	ID         string
	ProjectID  string
	Name       string
	KeyPrefix  string
	KeyHash    string
	Enabled    bool
	ExpiresAt  *time.Time
	LastUsedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func NewAPIKey(
	projectID string,
	name string,
	keyPrefix string,
	keyHash string,
	expiresAt *time.Time,
	now time.Time,
) (APIKey, error) {
	projectID = strings.TrimSpace(projectID)
	name = strings.TrimSpace(name)
	keyPrefix = strings.TrimSpace(keyPrefix)
	keyHash = strings.TrimSpace(keyHash)

	if projectID == "" {
		return APIKey{}, ErrInvalidProjectID
	}

	if name == "" {
		return APIKey{}, ErrInvalidName
	}

	if keyPrefix == "" {
		return APIKey{}, ErrInvalidKeyPrefix
	}

	if keyHash == "" {
		return APIKey{}, ErrInvalidKeyHash
	}

	if expiresAt != nil && !expiresAt.After(now) {
		return APIKey{}, ErrInvalidExpiration
	}

	return APIKey{
		ProjectID: projectID,
		Name:      name,
		KeyPrefix: keyPrefix,
		KeyHash:   keyHash,
		Enabled:   true,
		ExpiresAt: expiresAt,
	}, nil
}

func (k APIKey) IsActive(now time.Time) bool {
	if !k.Enabled {
		return false
	}

	return k.ExpiresAt == nil || now.Before(*k.ExpiresAt)
}
