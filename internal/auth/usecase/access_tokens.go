package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/aralary/edgeguard/internal/auth/domain"
)

type AccessTokenPrincipal struct {
	UserID    string
	Role      domain.Role
	ExpiresAt time.Time
}

func (u *Usecase) AuthenticateAccessToken(
	ctx context.Context,
	rawAccessToken string,
) (AccessTokenPrincipal, error) {
	if u.tokenService == nil || u.userRepository == nil {
		return AccessTokenPrincipal{}, ErrAccessTokenDependenciesMissing
	}

	rawAccessToken = strings.TrimSpace(rawAccessToken)
	if rawAccessToken == "" {
		return AccessTokenPrincipal{}, ErrInvalidAccessToken
	}

	principal, err := u.tokenService.ParseAccessToken(rawAccessToken, u.clock.Now())
	if err != nil {
		return AccessTokenPrincipal{}, ErrInvalidAccessToken
	}

	user, err := u.userRepository.GetUserByID(ctx, principal.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return AccessTokenPrincipal{}, ErrInvalidAccessToken
		}

		return AccessTokenPrincipal{}, err
	}

	if !user.Enabled {
		return AccessTokenPrincipal{}, ErrUserDisabled
	}
	if user.Role != principal.Role {
		return AccessTokenPrincipal{}, ErrInvalidAccessToken
	}

	return principal, nil
}
