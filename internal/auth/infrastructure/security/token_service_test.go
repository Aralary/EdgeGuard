package security

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aralary/edgeguard/internal/auth/domain"
	"github.com/golang-jwt/jwt/v5"
)

const testJWTSecret = "0123456789abcdef0123456789abcdef"

func TestTokenServiceIssueAndParseAccessToken(t *testing.T) {
	service := newTestTokenService(t, TokenServiceConfig{
		JWTSecret:      testJWTSecret,
		JWTIssuer:      "edgeguard-test-auth",
		JWTAudience:    "edgeguard-test",
		AccessTokenTTL: 10 * time.Minute,
	})

	now := time.Date(2026, 7, 12, 10, 0, 0, 0, time.UTC)
	issued, err := service.IssueAccessToken(domain.User{
		ID:      "user-1",
		Role:    domain.RoleMember,
		Enabled: true,
	}, now)
	if err != nil {
		t.Fatalf("issue access token: %v", err)
	}

	wantExpiresAt := now.Add(10 * time.Minute)
	if !issued.ExpiresAt.Equal(wantExpiresAt) {
		t.Fatalf("unexpected expiration: got %s want %s", issued.ExpiresAt, wantExpiresAt)
	}

	parsed, err := service.ParseAccessToken(issued.Value, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("parse access token: %v", err)
	}
	if parsed.UserID != "user-1" {
		t.Fatalf("unexpected user id: %q", parsed.UserID)
	}
	if parsed.Role != domain.RoleMember {
		t.Fatalf("unexpected role: %q", parsed.Role)
	}
	if !parsed.ExpiresAt.Equal(wantExpiresAt) {
		t.Fatalf("unexpected parsed expiration: %s", parsed.ExpiresAt)
	}
}

func TestTokenServiceRejectsExpiredAccessToken(t *testing.T) {
	service := newTestTokenService(t, TokenServiceConfig{
		JWTSecret:      testJWTSecret,
		AccessTokenTTL: time.Minute,
	})

	now := time.Date(2026, 7, 12, 10, 0, 0, 0, time.UTC)
	issued, err := service.IssueAccessToken(domain.User{
		ID:      "user-1",
		Role:    domain.RoleMember,
		Enabled: true,
	}, now)
	if err != nil {
		t.Fatalf("issue access token: %v", err)
	}

	_, err = service.ParseAccessToken(issued.Value, now.Add(2*time.Minute))
	if !errors.Is(err, ErrInvalidAccessToken) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTokenServiceRejectsUnexpectedSigningMethod(t *testing.T) {
	service := newTestTokenService(t, TokenServiceConfig{JWTSecret: testJWTSecret})
	now := time.Date(2026, 7, 12, 10, 0, 0, 0, time.UTC)

	claims := AccessTokenClaims{
		Role: domain.RoleMember,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    DefaultJWTIssuer,
			Subject:   "user-1",
			Audience:  jwt.ClaimStrings{DefaultJWTAudience},
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS384, claims)
	rawToken, err := token.SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	_, err = service.ParseAccessToken(rawToken, now)
	if !errors.Is(err, ErrInvalidAccessToken) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTokenServiceRejectsTokenSignedWithAnotherSecret(t *testing.T) {
	issuer := newTestTokenService(t, TokenServiceConfig{JWTSecret: testJWTSecret})
	verifier := newTestTokenService(t, TokenServiceConfig{
		JWTSecret: "abcdef0123456789abcdef0123456789",
	})
	now := time.Date(2026, 7, 12, 10, 0, 0, 0, time.UTC)

	issued, err := issuer.IssueAccessToken(domain.User{
		ID:      "user-1",
		Role:    domain.RoleMember,
		Enabled: true,
	}, now)
	if err != nil {
		t.Fatalf("issue access token: %v", err)
	}

	_, err = verifier.ParseAccessToken(issued.Value, now)
	if !errors.Is(err, ErrInvalidAccessToken) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTokenServiceGenerateRefreshToken(t *testing.T) {
	service := newTestTokenService(t, TokenServiceConfig{
		JWTSecret:        testJWTSecret,
		RefreshTokenSize: 48,
	})

	rawToken, tokenHash, err := service.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("generate refresh token: %v", err)
	}

	decoded, err := base64.RawURLEncoding.DecodeString(rawToken)
	if err != nil {
		t.Fatalf("decode refresh token: %v", err)
	}
	if len(decoded) != 48 {
		t.Fatalf("unexpected refresh token size: %d", len(decoded))
	}

	hash := sha256.Sum256([]byte(rawToken))
	wantHash := hex.EncodeToString(hash[:])
	if tokenHash != wantHash {
		t.Fatalf("unexpected token hash: got %q want %q", tokenHash, wantHash)
	}
	if service.HashRefreshToken(rawToken) != tokenHash {
		t.Fatal("HashRefreshToken returned another hash")
	}

	secondRawToken, _, err := service.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("generate second refresh token: %v", err)
	}
	if secondRawToken == rawToken {
		t.Fatal("generated duplicate refresh tokens")
	}
}

func TestNewTokenServiceValidation(t *testing.T) {
	tests := []struct {
		name   string
		config TokenServiceConfig
		want   error
	}{
		{
			name:   "weak secret",
			config: TokenServiceConfig{JWTSecret: "short"},
			want:   ErrJWTSecretTooShort,
		},
		{
			name: "negative ttl",
			config: TokenServiceConfig{
				JWTSecret:      testJWTSecret,
				AccessTokenTTL: -time.Second,
			},
			want: ErrInvalidAccessTokenTTL,
		},
		{
			name: "small refresh token",
			config: TokenServiceConfig{
				JWTSecret:        testJWTSecret,
				RefreshTokenSize: 8,
			},
			want: ErrInvalidRefreshTokenSize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTokenService(tt.config)
			if !errors.Is(err, tt.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestTokenServiceRejectsInvalidUser(t *testing.T) {
	service := newTestTokenService(t, TokenServiceConfig{JWTSecret: testJWTSecret})
	now := time.Date(2026, 7, 12, 10, 0, 0, 0, time.UTC)

	users := []domain.User{
		{Role: domain.RoleMember, Enabled: true},
		{ID: "user-1", Role: domain.Role("unknown"), Enabled: true},
		{ID: "user-1", Role: domain.RoleMember, Enabled: false},
	}

	for _, user := range users {
		_, err := service.IssueAccessToken(user, now)
		if !errors.Is(err, ErrInvalidTokenUser) {
			t.Fatalf("unexpected error for user %+v: %v", user, err)
		}
	}
}

func newTestTokenService(t *testing.T, config TokenServiceConfig) *TokenService {
	t.Helper()

	service, err := NewTokenService(config)
	if err != nil {
		t.Fatalf("new token service: %v", err)
	}

	return service
}

func TestTokenServiceDefaults(t *testing.T) {
	service := newTestTokenService(t, TokenServiceConfig{JWTSecret: testJWTSecret})

	if service.issuer != DefaultJWTIssuer {
		t.Fatalf("unexpected issuer: %q", service.issuer)
	}
	if service.audience != DefaultJWTAudience {
		t.Fatalf("unexpected audience: %q", service.audience)
	}
	if service.accessTokenTTL != DefaultAccessTokenTTL {
		t.Fatalf("unexpected ttl: %s", service.accessTokenTTL)
	}
	if service.refreshTokenSize != DefaultRefreshTokenSize {
		t.Fatalf("unexpected refresh token size: %d", service.refreshTokenSize)
	}
}

func TestParseAccessTokenRejectsEmptyToken(t *testing.T) {
	service := newTestTokenService(t, TokenServiceConfig{JWTSecret: testJWTSecret})

	_, err := service.ParseAccessToken(strings.Repeat(" ", 3), time.Now())
	if !errors.Is(err, ErrInvalidAccessToken) {
		t.Fatalf("unexpected error: %v", err)
	}
}
