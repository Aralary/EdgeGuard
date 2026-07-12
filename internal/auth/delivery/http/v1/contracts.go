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
}
