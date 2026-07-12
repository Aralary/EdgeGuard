package usecase

import (
	"context"
	"errors"
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

	err := u.refreshTokenRepository.RevokeRefreshTokenByHash(ctx, tokenHash, u.clock.Now())
	if errors.Is(err, ErrRefreshTokenNotFound) {
		return ErrInvalidRefreshToken
	}

	return err
}
