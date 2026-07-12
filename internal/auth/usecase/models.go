package usecase

import (
	"time"

	"github.com/aralary/edgeguard/internal/auth/domain"
)

type UserInfo struct {
	ID        string
	Email     string
	Role      domain.Role
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type TokenPair struct {
	TokenType             string
	AccessToken           string
	AccessTokenExpiresAt  time.Time
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

type Session struct {
	User   UserInfo
	Tokens TokenPair
}

func userInfoFromDomain(user domain.User) UserInfo {
	return UserInfo{
		ID:        user.ID,
		Email:     user.Email,
		Role:      user.Role,
		Enabled:   user.Enabled,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}
