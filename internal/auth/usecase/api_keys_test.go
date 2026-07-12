package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aralary/edgeguard/internal/auth/domain"
)

type fakeAPIKeyRepository struct {
	createdKey     domain.APIKey
	keys           []domain.APIKey
	keyByPrefix    domain.APIKey
	createErr      error
	listErr        error
	getErr         error
	disableErr     error
	updateUsedErr  error
	disabledID     string
	disabledProjID string
	disabledAt     time.Time
	usedID         string
	usedAt         time.Time
}

func (r *fakeAPIKeyRepository) CreateAPIKey(
	_ context.Context,
	key domain.APIKey,
) (domain.APIKey, error) {
	if r.createErr != nil {
		return domain.APIKey{}, r.createErr
	}

	key.ID = "key-1"
	key.CreatedAt = time.Date(2026, time.July, 12, 12, 0, 0, 0, time.UTC)
	key.UpdatedAt = key.CreatedAt
	r.createdKey = key
	return key, nil
}

func (r *fakeAPIKeyRepository) ListAPIKeysByProjectID(
	_ context.Context,
	_ string,
) ([]domain.APIKey, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}

	return r.keys, nil
}

func (r *fakeAPIKeyRepository) GetAPIKeyByPrefix(
	_ context.Context,
	_ string,
) (domain.APIKey, error) {
	if r.getErr != nil {
		return domain.APIKey{}, r.getErr
	}

	return r.keyByPrefix, nil
}

func (r *fakeAPIKeyRepository) DisableAPIKey(
	_ context.Context,
	projectID string,
	apiKeyID string,
	disabledAt time.Time,
) error {
	if r.disableErr != nil {
		return r.disableErr
	}

	r.disabledProjID = projectID
	r.disabledID = apiKeyID
	r.disabledAt = disabledAt
	return nil
}

func (r *fakeAPIKeyRepository) UpdateAPIKeyLastUsedAt(
	_ context.Context,
	apiKeyID string,
	usedAt time.Time,
) error {
	if r.updateUsedErr != nil {
		return r.updateUsedErr
	}

	r.usedID = apiKeyID
	r.usedAt = usedAt
	return nil
}

type fakeAPIKeyService struct {
	generated   GeneratedAPIKey
	parsed      ParsedAPIKey
	generateErr error
	parseErr    error
	matches     bool
}

func (s *fakeAPIKeyService) GenerateAPIKey() (GeneratedAPIKey, error) {
	if s.generateErr != nil {
		return GeneratedAPIKey{}, s.generateErr
	}

	return s.generated, nil
}

func (s *fakeAPIKeyService) ParseAPIKey(_ string) (ParsedAPIKey, error) {
	if s.parseErr != nil {
		return ParsedAPIKey{}, s.parseErr
	}

	return s.parsed, nil
}

func (s *fakeAPIKeyService) MatchAPIKeyHash(_ string, _ string) bool {
	return s.matches
}

func newAPIKeyUsecase(
	repository APIKeyRepository,
	service APIKeyService,
	clock Clock,
) *Usecase {
	return New(Dependencies{
		APIKeyRepository: repository,
		APIKeyService:    service,
		Clock:            clock,
	}, Config{})
}

func TestCreateAPIKeyReturnsSecretOnlyOnce(t *testing.T) {
	now := time.Date(2026, time.July, 12, 12, 0, 0, 0, time.UTC)
	repository := &fakeAPIKeyRepository{}
	service := &fakeAPIKeyService{generated: GeneratedAPIKey{
		Value:  "eg_live_prefix_secret",
		Prefix: "prefix",
		Hash:   "key-hash",
	}}
	uc := newAPIKeyUsecase(repository, service, fixedClock{now: now})

	created, err := uc.CreateAPIKey(context.Background(), CreateAPIKeyInput{
		ProjectID: "project-1",
		Name:      "production",
	})
	if err != nil {
		t.Fatalf("CreateAPIKey() error = %v", err)
	}

	if created.Value != "eg_live_prefix_secret" {
		t.Fatalf("created value = %q", created.Value)
	}
	if repository.createdKey.KeyHash != "key-hash" {
		t.Fatalf("stored hash = %q", repository.createdKey.KeyHash)
	}
	if created.APIKey.ID != "key-1" || created.APIKey.KeyPrefix != "prefix" {
		t.Fatalf("created metadata = %+v", created.APIKey)
	}
}

func TestListAPIKeysDoesNotExposeHashes(t *testing.T) {
	repository := &fakeAPIKeyRepository{keys: []domain.APIKey{{
		ID:        "key-1",
		ProjectID: "project-1",
		Name:      "production",
		KeyPrefix: "prefix",
		KeyHash:   "secret-hash",
		Enabled:   true,
	}}}
	uc := newAPIKeyUsecase(repository, &fakeAPIKeyService{}, fixedClock{})

	keys, err := uc.ListAPIKeys(context.Background(), "project-1")
	if err != nil {
		t.Fatalf("ListAPIKeys() error = %v", err)
	}
	if len(keys) != 1 || keys[0].ID != "key-1" || keys[0].KeyPrefix != "prefix" {
		t.Fatalf("ListAPIKeys() = %+v", keys)
	}
}

func TestValidateAPIKeyReturnsPrincipalAndUpdatesLastUsedAt(t *testing.T) {
	now := time.Date(2026, time.July, 12, 12, 0, 0, 0, time.UTC)
	repository := &fakeAPIKeyRepository{keyByPrefix: domain.APIKey{
		ID:        "key-1",
		ProjectID: "project-1",
		KeyPrefix: "prefix",
		KeyHash:   "stored-hash",
		Enabled:   true,
	}}
	service := &fakeAPIKeyService{
		parsed:  ParsedAPIKey{Prefix: "prefix", Hash: "actual-hash"},
		matches: true,
	}
	uc := newAPIKeyUsecase(repository, service, fixedClock{now: now})

	principal, err := uc.ValidateAPIKey(context.Background(), "raw-key")
	if err != nil {
		t.Fatalf("ValidateAPIKey() error = %v", err)
	}
	if principal.APIKeyID != "key-1" || principal.ProjectID != "project-1" {
		t.Fatalf("principal = %+v", principal)
	}
	if repository.usedID != "key-1" || !repository.usedAt.Equal(now) {
		t.Fatalf("last used update = (%q, %v)", repository.usedID, repository.usedAt)
	}
}

func TestValidateAPIKeyHidesInvalidKeyReason(t *testing.T) {
	now := time.Date(2026, time.July, 12, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		repository *fakeAPIKeyRepository
		service    *fakeAPIKeyService
	}{
		{
			name:       "malformed key",
			repository: &fakeAPIKeyRepository{},
			service:    &fakeAPIKeyService{parseErr: errors.New("malformed")},
		},
		{
			name:       "unknown prefix",
			repository: &fakeAPIKeyRepository{getErr: ErrAPIKeyNotFound},
			service:    &fakeAPIKeyService{parsed: ParsedAPIKey{Prefix: "missing"}},
		},
		{
			name: "hash mismatch",
			repository: &fakeAPIKeyRepository{keyByPrefix: domain.APIKey{
				ID:      "key-1",
				Enabled: true,
			}},
			service: &fakeAPIKeyService{matches: false},
		},
		{
			name: "disabled key",
			repository: &fakeAPIKeyRepository{keyByPrefix: domain.APIKey{
				ID:      "key-1",
				Enabled: false,
			}},
			service: &fakeAPIKeyService{matches: true},
		},
		{
			name: "expired key",
			repository: &fakeAPIKeyRepository{keyByPrefix: domain.APIKey{
				ID:        "key-1",
				Enabled:   true,
				ExpiresAt: &now,
			}},
			service: &fakeAPIKeyService{matches: true},
		},
		{
			name: "revoked during validation",
			repository: &fakeAPIKeyRepository{
				keyByPrefix:   domain.APIKey{ID: "key-1", Enabled: true},
				updateUsedErr: ErrAPIKeyNotFound,
			},
			service: &fakeAPIKeyService{matches: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := newAPIKeyUsecase(tt.repository, tt.service, fixedClock{now: now})

			_, err := uc.ValidateAPIKey(context.Background(), "raw-key")
			if !errors.Is(err, ErrInvalidAPIKey) {
				t.Fatalf("ValidateAPIKey() error = %v, want %v", err, ErrInvalidAPIKey)
			}
		})
	}
}

func TestRevokeAPIKeyDisablesProjectKey(t *testing.T) {
	now := time.Date(2026, time.July, 12, 12, 0, 0, 0, time.UTC)
	repository := &fakeAPIKeyRepository{}
	uc := newAPIKeyUsecase(repository, &fakeAPIKeyService{}, fixedClock{now: now})

	if err := uc.RevokeAPIKey(context.Background(), "project-1", "key-1"); err != nil {
		t.Fatalf("RevokeAPIKey() error = %v", err)
	}
	if repository.disabledProjID != "project-1" || repository.disabledID != "key-1" {
		t.Fatalf("disabled key = project %q key %q", repository.disabledProjID, repository.disabledID)
	}
	if !repository.disabledAt.Equal(now) {
		t.Fatalf("disabled at = %v, want %v", repository.disabledAt, now)
	}
}
