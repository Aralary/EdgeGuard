package usecase

import (
	"context"
	"errors"
	"strings"
)

type RefreshInput struct {
	RefreshToken string
}

func (u *Usecase) Refresh(ctx context.Context, input RefreshInput) (TokenPair, error) {
	rawRefreshToken := strings.TrimSpace(input.RefreshToken)
	if rawRefreshToken == "" {
		return TokenPair{}, ErrInvalidRefreshToken
	}

	tokenHash := u.tokenService.HashRefreshToken(rawRefreshToken)
	if tokenHash == "" {
		return TokenPair{}, ErrInvalidRefreshToken
	}

	currentToken, err := u.refreshTokenRepository.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, ErrRefreshTokenNotFound) {
			return TokenPair{}, ErrInvalidRefreshToken
		}

		return TokenPair{}, err
	}

	now := u.clock.Now()
	if !currentToken.IsActive(now) {
		return TokenPair{}, ErrInvalidRefreshToken
	}

	user, err := u.userRepository.GetUserByID(ctx, currentToken.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return TokenPair{}, ErrInvalidRefreshToken
		}

		return TokenPair{}, err
	}

	if !user.Enabled {
		return TokenPair{}, ErrUserDisabled
	}

	accessToken, replacementRawToken, replacementToken, err := u.buildTokens(user, now)
	if err != nil {
		return TokenPair{}, err
	}

	createdReplacement, err := u.refreshTokenRepository.RotateRefreshToken(
		ctx,
		currentToken.ID,
		replacementToken,
		now,
	)
	if err != nil {
		return TokenPair{}, err
	}

	return tokenPair(accessToken, replacementRawToken, createdReplacement.ExpiresAt), nil
}
