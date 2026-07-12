package httpdelivery

import (
	"context"

	"github.com/aralary/edgeguard/internal/auth/usecase"
)

type AuthUsecase interface {
	Register(ctx context.Context, input usecase.RegisterInput) (usecase.UserInfo, error)
	Login(ctx context.Context, input usecase.LoginInput) (usecase.Session, error)
	Refresh(ctx context.Context, input usecase.RefreshInput) (usecase.TokenPair, error)
	Logout(ctx context.Context, input usecase.LogoutInput) error

	AuthenticateAccessToken(
		ctx context.Context,
		rawAccessToken string,
	) (usecase.AccessTokenPrincipal, error)

	CreateAPIKey(
		ctx context.Context,
		input usecase.CreateAPIKeyInput,
	) (usecase.CreatedAPIKey, error)

	ListAPIKeys(
		ctx context.Context,
		projectID string,
	) ([]usecase.APIKeyInfo, error)

	RevokeAPIKey(
		ctx context.Context,
		projectID string,
		apiKeyID string,
	) error

	ValidateAPIKey(
		ctx context.Context,
		rawAPIKey string,
	) (usecase.APIKeyPrincipal, error)
}
