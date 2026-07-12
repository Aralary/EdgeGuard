package usecase

import (
	"context"
	"errors"
	"strings"
)

type LoginInput struct {
	Email    string
	Password string
}

func (u *Usecase) Login(ctx context.Context, input LoginInput) (Session, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))

	user, err := u.userRepository.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return Session{}, ErrInvalidCredentials
		}

		return Session{}, err
	}

	if !user.Enabled {
		return Session{}, ErrUserDisabled
	}

	if !u.passwordHasher.Matches(user.PasswordHash, input.Password) {
		return Session{}, ErrInvalidCredentials
	}

	now := u.clock.Now()
	accessToken, rawRefreshToken, refreshToken, err := u.buildTokens(user, now)
	if err != nil {
		return Session{}, err
	}

	createdRefreshToken, err := u.refreshTokenRepository.CreateRefreshToken(ctx, refreshToken)
	if err != nil {
		return Session{}, err
	}

	return Session{
		User: userInfoFromDomain(user),
		Tokens: tokenPair(
			accessToken,
			rawRefreshToken,
			createdRefreshToken.ExpiresAt,
		),
	}, nil
}
