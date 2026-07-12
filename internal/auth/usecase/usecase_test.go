package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aralary/edgeguard/internal/auth/domain"
)

type fakeUserRepository struct {
	createdUser domain.User
	userByEmail domain.User
	userByID    domain.User
	createErr   error
	emailErr    error
	idErr       error
}

func (r *fakeUserRepository) CreateUser(_ context.Context, user domain.User) (domain.User, error) {
	if r.createErr != nil {
		return domain.User{}, r.createErr
	}

	user.ID = "user-1"
	r.createdUser = user
	return user, nil
}

func (r *fakeUserRepository) GetUserByEmail(_ context.Context, _ string) (domain.User, error) {
	if r.emailErr != nil {
		return domain.User{}, r.emailErr
	}

	return r.userByEmail, nil
}

func (r *fakeUserRepository) GetUserByID(_ context.Context, _ string) (domain.User, error) {
	if r.idErr != nil {
		return domain.User{}, r.idErr
	}

	return r.userByID, nil
}

type fakeRefreshTokenRepository struct {
	currentToken domain.RefreshToken
	createdToken domain.RefreshToken
	rotatedToken domain.RefreshToken
	rotatedID    string
	revokedHash  string
	revokedAt    time.Time
	createErr    error
	getErr       error
	rotateErr    error
	revokeErr    error
}

func (r *fakeRefreshTokenRepository) CreateRefreshToken(
	_ context.Context,
	token domain.RefreshToken,
) (domain.RefreshToken, error) {
	if r.createErr != nil {
		return domain.RefreshToken{}, r.createErr
	}

	token.ID = "refresh-1"
	r.createdToken = token
	return token, nil
}

func (r *fakeRefreshTokenRepository) GetRefreshTokenByHash(
	_ context.Context,
	_ string,
) (domain.RefreshToken, error) {
	if r.getErr != nil {
		return domain.RefreshToken{}, r.getErr
	}

	return r.currentToken, nil
}

func (r *fakeRefreshTokenRepository) RotateRefreshToken(
	_ context.Context,
	currentTokenID string,
	replacement domain.RefreshToken,
	revokedAt time.Time,
) (domain.RefreshToken, error) {
	if r.rotateErr != nil {
		return domain.RefreshToken{}, r.rotateErr
	}

	replacement.ID = "refresh-2"
	r.rotatedID = currentTokenID
	r.rotatedToken = replacement
	r.revokedAt = revokedAt
	return replacement, nil
}

func (r *fakeRefreshTokenRepository) RevokeRefreshTokenByHash(
	_ context.Context,
	tokenHash string,
	revokedAt time.Time,
) error {
	if r.revokeErr != nil {
		return r.revokeErr
	}

	r.revokedHash = tokenHash
	r.revokedAt = revokedAt
	return nil
}

type fakePasswordHasher struct {
	hash      string
	hashErr   error
	matches   bool
	password  string
	compared  string
	plainText string
}

func (h *fakePasswordHasher) Hash(password string) (string, error) {
	h.password = password
	if h.hashErr != nil {
		return "", h.hashErr
	}

	return h.hash, nil
}

func (h *fakePasswordHasher) Matches(passwordHash string, password string) bool {
	h.compared = passwordHash
	h.plainText = password
	return h.matches
}

type fakeTokenService struct {
	accessToken IssuedAccessToken
	accessErr   error
	rawToken    string
	tokenHash   string
	generateErr error
}

func (s *fakeTokenService) IssueAccessToken(
	_ domain.User,
	_ time.Time,
) (IssuedAccessToken, error) {
	if s.accessErr != nil {
		return IssuedAccessToken{}, s.accessErr
	}

	return s.accessToken, nil
}

func (s *fakeTokenService) GenerateRefreshToken() (string, string, error) {
	if s.generateErr != nil {
		return "", "", s.generateErr
	}

	return s.rawToken, s.tokenHash, nil
}

func (s *fakeTokenService) HashRefreshToken(rawToken string) string {
	return "hash:" + rawToken
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

func TestRegisterCreatesMemberWithHashedPassword(t *testing.T) {
	users := &fakeUserRepository{}
	hasher := &fakePasswordHasher{hash: "password-hash"}
	uc := New(
		users,
		&fakeRefreshTokenRepository{},
		hasher,
		&fakeTokenService{},
		fixedClock{},
		Config{},
	)

	got, err := uc.Register(context.Background(), RegisterInput{
		Email:    " User@Example.COM ",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if hasher.password != "password123" {
		t.Fatalf("hashed password = %q, want %q", hasher.password, "password123")
	}

	if users.createdUser.Email != "user@example.com" {
		t.Fatalf("created email = %q, want %q", users.createdUser.Email, "user@example.com")
	}

	if users.createdUser.PasswordHash != "password-hash" {
		t.Fatalf("created password hash = %q, want %q", users.createdUser.PasswordHash, "password-hash")
	}

	if users.createdUser.Role != domain.RoleMember {
		t.Fatalf("created role = %q, want %q", users.createdUser.Role, domain.RoleMember)
	}

	if got.ID != "user-1" || got.Email != "user@example.com" {
		t.Fatalf("Register() result = %+v", got)
	}
}

func TestRegisterRejectsInvalidPasswordBeforeHashing(t *testing.T) {
	hasher := &fakePasswordHasher{hash: "password-hash"}
	uc := New(
		&fakeUserRepository{},
		&fakeRefreshTokenRepository{},
		hasher,
		&fakeTokenService{},
		fixedClock{},
		Config{},
	)

	_, err := uc.Register(context.Background(), RegisterInput{
		Email:    "user@example.com",
		Password: "short",
	})
	if !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("Register() error = %v, want %v", err, ErrInvalidPassword)
	}

	if hasher.password != "" {
		t.Fatalf("Hash() called with %q for invalid password", hasher.password)
	}
}

func TestLoginCreatesSessionAndPersistsRefreshToken(t *testing.T) {
	now := time.Date(2026, time.July, 12, 12, 0, 0, 0, time.UTC)
	accessExpiresAt := now.Add(15 * time.Minute)
	user := domain.User{
		ID:           "user-1",
		Email:        "user@example.com",
		PasswordHash: "password-hash",
		Role:         domain.RoleMember,
		Enabled:      true,
	}

	users := &fakeUserRepository{userByEmail: user}
	refreshTokens := &fakeRefreshTokenRepository{}
	hasher := &fakePasswordHasher{matches: true}
	tokens := &fakeTokenService{
		accessToken: IssuedAccessToken{Value: "access-token", ExpiresAt: accessExpiresAt},
		rawToken:    "refresh-token",
		tokenHash:   "refresh-hash",
	}

	uc := New(users, refreshTokens, hasher, tokens, fixedClock{now: now}, Config{})

	got, err := uc.Login(context.Background(), LoginInput{
		Email:    " User@Example.COM ",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if hasher.compared != "password-hash" || hasher.plainText != "password123" {
		t.Fatalf("Matches() arguments = (%q, %q)", hasher.compared, hasher.plainText)
	}

	if refreshTokens.createdToken.UserID != user.ID {
		t.Fatalf("refresh token user id = %q, want %q", refreshTokens.createdToken.UserID, user.ID)
	}

	if refreshTokens.createdToken.TokenHash != "refresh-hash" {
		t.Fatalf("refresh token hash = %q, want %q", refreshTokens.createdToken.TokenHash, "refresh-hash")
	}

	wantRefreshExpiry := now.Add(DefaultRefreshTokenTTL)
	if !refreshTokens.createdToken.ExpiresAt.Equal(wantRefreshExpiry) {
		t.Fatalf("refresh token expiry = %v, want %v", refreshTokens.createdToken.ExpiresAt, wantRefreshExpiry)
	}

	if got.Tokens.AccessToken != "access-token" || got.Tokens.RefreshToken != "refresh-token" {
		t.Fatalf("Login() tokens = %+v", got.Tokens)
	}

	if got.Tokens.TokenType != "Bearer" {
		t.Fatalf("token type = %q, want Bearer", got.Tokens.TokenType)
	}
}

func TestLoginRejectsInvalidCredentials(t *testing.T) {
	tests := []struct {
		name    string
		users   *fakeUserRepository
		hasher  *fakePasswordHasher
		wantErr error
	}{
		{
			name:    "user not found",
			users:   &fakeUserRepository{emailErr: ErrUserNotFound},
			hasher:  &fakePasswordHasher{},
			wantErr: ErrInvalidCredentials,
		},
		{
			name: "password mismatch",
			users: &fakeUserRepository{userByEmail: domain.User{
				ID:           "user-1",
				PasswordHash: "password-hash",
				Enabled:      true,
			}},
			hasher:  &fakePasswordHasher{matches: false},
			wantErr: ErrInvalidCredentials,
		},
		{
			name: "disabled user",
			users: &fakeUserRepository{userByEmail: domain.User{
				ID:           "user-1",
				PasswordHash: "password-hash",
				Enabled:      false,
			}},
			hasher:  &fakePasswordHasher{matches: true},
			wantErr: ErrUserDisabled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := New(
				tt.users,
				&fakeRefreshTokenRepository{},
				tt.hasher,
				&fakeTokenService{},
				fixedClock{},
				Config{},
			)

			_, err := uc.Login(context.Background(), LoginInput{
				Email:    "user@example.com",
				Password: "password123",
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Login() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestRefreshRotatesTokenAtomically(t *testing.T) {
	now := time.Date(2026, time.July, 12, 12, 0, 0, 0, time.UTC)
	user := domain.User{
		ID:      "user-1",
		Email:   "user@example.com",
		Role:    domain.RoleMember,
		Enabled: true,
	}

	users := &fakeUserRepository{userByID: user}
	refreshTokens := &fakeRefreshTokenRepository{
		currentToken: domain.RefreshToken{
			ID:        "refresh-1",
			UserID:    user.ID,
			TokenHash: "hash:current-token",
			ExpiresAt: now.Add(time.Hour),
		},
	}
	tokens := &fakeTokenService{
		accessToken: IssuedAccessToken{Value: "new-access-token", ExpiresAt: now.Add(15 * time.Minute)},
		rawToken:    "new-refresh-token",
		tokenHash:   "new-refresh-hash",
	}

	uc := New(
		users,
		refreshTokens,
		&fakePasswordHasher{},
		tokens,
		fixedClock{now: now},
		Config{RefreshTokenTTL: 7 * 24 * time.Hour},
	)

	got, err := uc.Refresh(context.Background(), RefreshInput{RefreshToken: "current-token"})
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	if refreshTokens.rotatedID != "refresh-1" {
		t.Fatalf("rotated token id = %q, want %q", refreshTokens.rotatedID, "refresh-1")
	}

	if refreshTokens.rotatedToken.TokenHash != "new-refresh-hash" {
		t.Fatalf("replacement hash = %q, want %q", refreshTokens.rotatedToken.TokenHash, "new-refresh-hash")
	}

	wantExpiry := now.Add(7 * 24 * time.Hour)
	if !refreshTokens.rotatedToken.ExpiresAt.Equal(wantExpiry) {
		t.Fatalf("replacement expiry = %v, want %v", refreshTokens.rotatedToken.ExpiresAt, wantExpiry)
	}

	if !refreshTokens.revokedAt.Equal(now) {
		t.Fatalf("revoked at = %v, want %v", refreshTokens.revokedAt, now)
	}

	if got.AccessToken != "new-access-token" || got.RefreshToken != "new-refresh-token" {
		t.Fatalf("Refresh() tokens = %+v", got)
	}
}

func TestRefreshRejectsExpiredOrUnknownToken(t *testing.T) {
	now := time.Date(2026, time.July, 12, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		refreshTokens *fakeRefreshTokenRepository
	}{
		{
			name:          "unknown token",
			refreshTokens: &fakeRefreshTokenRepository{getErr: ErrRefreshTokenNotFound},
		},
		{
			name: "expired token",
			refreshTokens: &fakeRefreshTokenRepository{currentToken: domain.RefreshToken{
				ID:        "refresh-1",
				UserID:    "user-1",
				ExpiresAt: now,
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := New(
				&fakeUserRepository{},
				tt.refreshTokens,
				&fakePasswordHasher{},
				&fakeTokenService{},
				fixedClock{now: now},
				Config{},
			)

			_, err := uc.Refresh(context.Background(), RefreshInput{RefreshToken: "current-token"})
			if !errors.Is(err, ErrInvalidRefreshToken) {
				t.Fatalf("Refresh() error = %v, want %v", err, ErrInvalidRefreshToken)
			}

			if tt.refreshTokens.rotatedID != "" {
				t.Fatalf("RotateRefreshToken() called for invalid token")
			}
		})
	}
}

func TestLogoutRevokesRefreshToken(t *testing.T) {
	now := time.Date(2026, time.July, 12, 12, 0, 0, 0, time.UTC)
	refreshTokens := &fakeRefreshTokenRepository{}
	uc := New(
		&fakeUserRepository{},
		refreshTokens,
		&fakePasswordHasher{},
		&fakeTokenService{},
		fixedClock{now: now},
		Config{},
	)

	if err := uc.Logout(context.Background(), LogoutInput{RefreshToken: "refresh-token"}); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}

	if refreshTokens.revokedHash != "hash:refresh-token" {
		t.Fatalf("revoked hash = %q, want %q", refreshTokens.revokedHash, "hash:refresh-token")
	}

	if !refreshTokens.revokedAt.Equal(now) {
		t.Fatalf("revoked at = %v, want %v", refreshTokens.revokedAt, now)
	}
}

func TestLogoutHidesUnknownRefreshToken(t *testing.T) {
	uc := New(
		&fakeUserRepository{},
		&fakeRefreshTokenRepository{revokeErr: ErrRefreshTokenNotFound},
		&fakePasswordHasher{},
		&fakeTokenService{},
		fixedClock{},
		Config{},
	)

	err := uc.Logout(context.Background(), LogoutInput{RefreshToken: "unknown-token"})
	if !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("Logout() error = %v, want %v", err, ErrInvalidRefreshToken)
	}
}
