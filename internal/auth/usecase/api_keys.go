package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aralary/edgeguard/internal/auth/domain"
)

type CreateAPIKeyInput struct {
	ProjectID string
	Name      string
	ExpiresAt *time.Time
}

type CreatedAPIKey struct {
	APIKey APIKeyInfo
	Value  string
}

type APIKeyInfo struct {
	ID         string
	ProjectID  string
	Name       string
	KeyPrefix  string
	Enabled    bool
	ExpiresAt  *time.Time
	LastUsedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type APIKeyPrincipal struct {
	APIKeyID  string
	ProjectID string
}

func (u *Usecase) CreateAPIKey(
	ctx context.Context,
	input CreateAPIKeyInput,
) (CreatedAPIKey, error) {
	if u.apiKeyRepository == nil || u.apiKeyService == nil {
		return CreatedAPIKey{}, ErrAPIKeyDependenciesMissing
	}

	now := u.clock.Now()
	projectID := strings.TrimSpace(input.ProjectID)
	if projectID == "" {
		return CreatedAPIKey{}, domain.ErrInvalidProjectID
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return CreatedAPIKey{}, domain.ErrInvalidName
	}

	if input.ExpiresAt != nil && !input.ExpiresAt.After(now) {
		return CreatedAPIKey{}, domain.ErrInvalidExpiration
	}

	generated, err := u.apiKeyService.GenerateAPIKey()
	if err != nil {
		return CreatedAPIKey{}, fmt.Errorf("generate api key: %w", err)
	}

	key, err := domain.NewAPIKey(
		projectID,
		name,
		generated.Prefix,
		generated.Hash,
		input.ExpiresAt,
		now,
	)
	if err != nil {
		return CreatedAPIKey{}, err
	}

	created, err := u.apiKeyRepository.CreateAPIKey(ctx, key)
	if err != nil {
		return CreatedAPIKey{}, err
	}

	return CreatedAPIKey{
		APIKey: apiKeyInfoFromDomain(created),
		Value:  generated.Value,
	}, nil
}

func (u *Usecase) ListAPIKeys(
	ctx context.Context,
	projectID string,
) ([]APIKeyInfo, error) {
	if u.apiKeyRepository == nil {
		return nil, ErrAPIKeyDependenciesMissing
	}

	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, domain.ErrInvalidProjectID
	}

	keys, err := u.apiKeyRepository.ListAPIKeysByProjectID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	result := make([]APIKeyInfo, 0, len(keys))
	for _, key := range keys {
		result = append(result, apiKeyInfoFromDomain(key))
	}

	return result, nil
}

func (u *Usecase) RevokeAPIKey(
	ctx context.Context,
	projectID string,
	apiKeyID string,
) error {
	if u.apiKeyRepository == nil {
		return ErrAPIKeyDependenciesMissing
	}

	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return domain.ErrInvalidProjectID
	}

	apiKeyID = strings.TrimSpace(apiKeyID)
	if apiKeyID == "" {
		return ErrInvalidAPIKeyID
	}

	return u.apiKeyRepository.DisableAPIKey(ctx, projectID, apiKeyID, u.clock.Now())
}

func (u *Usecase) ValidateAPIKey(
	ctx context.Context,
	rawAPIKey string,
) (APIKeyPrincipal, error) {
	if u.apiKeyRepository == nil || u.apiKeyService == nil {
		return APIKeyPrincipal{}, ErrAPIKeyDependenciesMissing
	}

	parsed, err := u.apiKeyService.ParseAPIKey(rawAPIKey)
	if err != nil {
		return APIKeyPrincipal{}, ErrInvalidAPIKey
	}

	key, err := u.apiKeyRepository.GetAPIKeyByPrefix(ctx, parsed.Prefix)
	if err != nil {
		if errors.Is(err, ErrAPIKeyNotFound) {
			return APIKeyPrincipal{}, ErrInvalidAPIKey
		}

		return APIKeyPrincipal{}, err
	}

	now := u.clock.Now()
	if !u.apiKeyService.MatchAPIKeyHash(parsed.Hash, key.KeyHash) || !key.IsActive(now) {
		return APIKeyPrincipal{}, ErrInvalidAPIKey
	}

	if err := u.apiKeyRepository.UpdateAPIKeyLastUsedAt(ctx, key.ID, now); err != nil {
		if errors.Is(err, ErrAPIKeyNotFound) {
			return APIKeyPrincipal{}, ErrInvalidAPIKey
		}

		return APIKeyPrincipal{}, err
	}

	return APIKeyPrincipal{
		APIKeyID:  key.ID,
		ProjectID: key.ProjectID,
	}, nil
}

func apiKeyInfoFromDomain(key domain.APIKey) APIKeyInfo {
	return APIKeyInfo{
		ID:         key.ID,
		ProjectID:  key.ProjectID,
		Name:       key.Name,
		KeyPrefix:  key.KeyPrefix,
		Enabled:    key.Enabled,
		ExpiresAt:  key.ExpiresAt,
		LastUsedAt: key.LastUsedAt,
		CreatedAt:  key.CreatedAt,
		UpdatedAt:  key.UpdatedAt,
	}
}
