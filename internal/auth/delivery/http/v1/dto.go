package httpdelivery

import (
	"time"

	"github.com/aralary/edgeguard/internal/auth/usecase"
)

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type userResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type tokenPairResponse struct {
	TokenType             string    `json:"token_type"`
	AccessToken           string    `json:"access_token"`
	AccessTokenExpiresAt  time.Time `json:"access_token_expires_at"`
	RefreshToken          string    `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
}

type sessionResponse struct {
	User   userResponse      `json:"user"`
	Tokens tokenPairResponse `json:"tokens"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func newUserResponse(user usecase.UserInfo) userResponse {
	return userResponse{
		ID:        user.ID,
		Email:     user.Email,
		Role:      string(user.Role),
		Enabled:   user.Enabled,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func newTokenPairResponse(tokens usecase.TokenPair) tokenPairResponse {
	return tokenPairResponse{
		TokenType:             tokens.TokenType,
		AccessToken:           tokens.AccessToken,
		AccessTokenExpiresAt:  tokens.AccessTokenExpiresAt,
		RefreshToken:          tokens.RefreshToken,
		RefreshTokenExpiresAt: tokens.RefreshTokenExpiresAt,
	}
}

func newSessionResponse(session usecase.Session) sessionResponse {
	return sessionResponse{
		User:   newUserResponse(session.User),
		Tokens: newTokenPairResponse(session.Tokens),
	}
}
