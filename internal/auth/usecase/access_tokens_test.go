package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aralary/edgeguard/internal/auth/domain"
)

func TestAuthenticateAccessToken(t *testing.T) {
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	users := &fakeUserRepository{userByID: domain.User{
		ID:      "user-1",
		Role:    domain.RoleMember,
		Enabled: true,
	}}
	tokens := &fakeTokenService{parsedAccessToken: AccessTokenPrincipal{
		UserID:    "user-1",
		Role:      domain.RoleMember,
		ExpiresAt: now.Add(time.Minute),
	}}

	uc := newTestUsecase(
		users,
		&fakeRefreshTokenRepository{},
		&fakePasswordHasher{},
		tokens,
		fixedClock{now: now},
		Config{},
	)

	principal, err := uc.AuthenticateAccessToken(context.Background(), "access-token")
	if err != nil {
		t.Fatalf("authenticate access token: %v", err)
	}
	if principal.UserID != "user-1" || principal.Role != domain.RoleMember {
		t.Fatalf("unexpected principal: %+v", principal)
	}
}

func TestAuthenticateAccessTokenRejectsUnknownUser(t *testing.T) {
	tokens := &fakeTokenService{parsedAccessToken: AccessTokenPrincipal{
		UserID: "unknown",
		Role:   domain.RoleMember,
	}}
	users := &fakeUserRepository{idErr: ErrUserNotFound}
	uc := newTestUsecase(
		users,
		&fakeRefreshTokenRepository{},
		&fakePasswordHasher{},
		tokens,
		fixedClock{now: time.Now().UTC()},
		Config{},
	)

	_, err := uc.AuthenticateAccessToken(context.Background(), "access-token")
	if !errors.Is(err, ErrInvalidAccessToken) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthenticateAccessTokenRejectsDisabledUser(t *testing.T) {
	users := &fakeUserRepository{userByID: domain.User{
		ID:      "user-1",
		Role:    domain.RoleMember,
		Enabled: false,
	}}
	tokens := &fakeTokenService{parsedAccessToken: AccessTokenPrincipal{
		UserID: "user-1",
		Role:   domain.RoleMember,
	}}
	uc := newTestUsecase(
		users,
		&fakeRefreshTokenRepository{},
		&fakePasswordHasher{},
		tokens,
		fixedClock{now: time.Now().UTC()},
		Config{},
	)

	_, err := uc.AuthenticateAccessToken(context.Background(), "access-token")
	if !errors.Is(err, ErrUserDisabled) {
		t.Fatalf("unexpected error: %v", err)
	}
}
