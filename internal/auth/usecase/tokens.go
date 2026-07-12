package usecase

import (
	"time"

	"github.com/aralary/edgeguard/internal/auth/domain"
)

func (u *Usecase) buildTokens(
	user domain.User,
	now time.Time,
) (IssuedAccessToken, string, domain.RefreshToken, error) {
	accessToken, err := u.tokenService.IssueAccessToken(user, now)
	if err != nil {
		return IssuedAccessToken{}, "", domain.RefreshToken{}, err
	}

	rawRefreshToken, refreshTokenHash, err := u.tokenService.GenerateRefreshToken()
	if err != nil {
		return IssuedAccessToken{}, "", domain.RefreshToken{}, err
	}

	refreshToken, err := domain.NewRefreshToken(
		user.ID,
		refreshTokenHash,
		now.Add(u.refreshTokenTTL),
		now,
	)
	if err != nil {
		return IssuedAccessToken{}, "", domain.RefreshToken{}, err
	}

	return accessToken, rawRefreshToken, refreshToken, nil
}

func tokenPair(
	accessToken IssuedAccessToken,
	rawRefreshToken string,
	refreshTokenExpiresAt time.Time,
) TokenPair {
	return TokenPair{
		TokenType:             "Bearer",
		AccessToken:           accessToken.Value,
		AccessTokenExpiresAt:  accessToken.ExpiresAt,
		RefreshToken:          rawRefreshToken,
		RefreshTokenExpiresAt: refreshTokenExpiresAt,
	}
}
