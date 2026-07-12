package usecase

import (
	"context"
	"strings"
)

type LogoutInput struct {
	RefreshToken string
}

func (u *Usecase) Logout(ctx context.Context, input LogoutInput) error {
	rawRefreshToken := strings.TrimSpace(input.RefreshToken)
	if rawRefreshToken == "" {
		return ErrInvalidRefreshToken
	}

	tokenHash := u.tokenService.HashRefreshToken(rawRefreshToken)
	if tokenHash == "" {
		return ErrInvalidRefreshToken
	}

	return u.refreshTokenRepository.RevokeRefreshTokenByHash(ctx, tokenHash, u.clock.Now())
}
