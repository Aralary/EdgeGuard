package usecase

import (
	"context"
	"time"

	"github.com/aralary/edgeguard/internal/auth/domain"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user domain.User) (domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (domain.User, error)
	GetUserByID(ctx context.Context, id string) (domain.User, error)
}

type RefreshTokenRepository interface {
	CreateRefreshToken(ctx context.Context, token domain.RefreshToken) (domain.RefreshToken, error)
	GetRefreshTokenByHash(ctx context.Context, tokenHash string) (domain.RefreshToken, error)
	RotateRefreshToken(
		ctx context.Context,
		currentTokenID string,
		replacement domain.RefreshToken,
		revokedAt time.Time,
	) (domain.RefreshToken, error)
	RevokeRefreshTokenByHash(ctx context.Context, tokenHash string, revokedAt time.Time) error
}

type APIKeyRepository interface {
	CreateAPIKey(ctx context.Context, key domain.APIKey) (domain.APIKey, error)
	ListAPIKeysByProjectID(ctx context.Context, projectID string) ([]domain.APIKey, error)
	GetAPIKeyByPrefix(ctx context.Context, keyPrefix string) (domain.APIKey, error)
	DisableAPIKey(ctx context.Context, projectID string, apiKeyID string, disabledAt time.Time) error
	UpdateAPIKeyLastUsedAt(ctx context.Context, apiKeyID string, usedAt time.Time) error
}

type GeneratedAPIKey struct {
	Value  string
	Prefix string
	Hash   string
}

type ParsedAPIKey struct {
	Prefix string
	Hash   string
}

type APIKeyService interface {
	GenerateAPIKey() (GeneratedAPIKey, error)
	ParseAPIKey(rawAPIKey string) (ParsedAPIKey, error)
	MatchAPIKeyHash(actualHash string, expectedHash string) bool
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Matches(passwordHash string, password string) bool
}

type IssuedAccessToken struct {
	Value     string
	ExpiresAt time.Time
}

type TokenService interface {
	IssueAccessToken(user domain.User, now time.Time) (IssuedAccessToken, error)
	ParseAccessToken(rawToken string, now time.Time) (AccessTokenPrincipal, error)
	GenerateRefreshToken() (rawToken string, tokenHash string, err error)
	HashRefreshToken(rawToken string) string
}

type Clock interface {
	Now() time.Time
}
