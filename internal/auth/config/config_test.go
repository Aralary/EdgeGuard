package config

import (
	"testing"
	"time"

	"github.com/aralary/edgeguard/internal/auth/infrastructure/security"
	"github.com/aralary/edgeguard/internal/auth/usecase"
)

func TestLoadConfigDefaults(t *testing.T) {
	clearOptionalAuthEnv(t)
	t.Setenv("AUTH_JWT_SECRET", "0123456789abcdef0123456789abcdef")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.HTTPAddr != DefaultHTTPAddr {
		t.Fatalf("unexpected http addr: %s", cfg.HTTPAddr)
	}
	if cfg.JWTIssuer != security.DefaultJWTIssuer {
		t.Fatalf("unexpected issuer: %s", cfg.JWTIssuer)
	}
	if cfg.JWTAudience != security.DefaultJWTAudience {
		t.Fatalf("unexpected audience: %s", cfg.JWTAudience)
	}
	if cfg.AccessTokenTTL != security.DefaultAccessTokenTTL {
		t.Fatalf("unexpected access token ttl: %s", cfg.AccessTokenTTL)
	}
	if cfg.RefreshTokenTTL != usecase.DefaultRefreshTokenTTL {
		t.Fatalf("unexpected refresh token ttl: %s", cfg.RefreshTokenTTL)
	}
	if cfg.RefreshTokenSize != security.DefaultRefreshTokenSize {
		t.Fatalf("unexpected refresh token size: %d", cfg.RefreshTokenSize)
	}
	if cfg.BcryptCost != security.DefaultBcryptCost {
		t.Fatalf("unexpected bcrypt cost: %d", cfg.BcryptCost)
	}
}

func TestLoadConfigOverrides(t *testing.T) {
	t.Setenv("AUTH_HTTP_ADDR", ":9083")
	t.Setenv("AUTH_JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("AUTH_JWT_ISSUER", "custom-issuer")
	t.Setenv("AUTH_JWT_AUDIENCE", "custom-audience")
	t.Setenv("AUTH_ACCESS_TOKEN_TTL", "5m")
	t.Setenv("AUTH_REFRESH_TOKEN_TTL", "48h")
	t.Setenv("AUTH_REFRESH_TOKEN_SIZE", "48")
	t.Setenv("AUTH_BCRYPT_COST", "12")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.HTTPAddr != ":9083" ||
		cfg.JWTIssuer != "custom-issuer" ||
		cfg.JWTAudience != "custom-audience" ||
		cfg.AccessTokenTTL != 5*time.Minute ||
		cfg.RefreshTokenTTL != 48*time.Hour ||
		cfg.RefreshTokenSize != 48 ||
		cfg.BcryptCost != 12 {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadConfigRequiresJWTSecret(t *testing.T) {
	clearOptionalAuthEnv(t)
	t.Setenv("AUTH_JWT_SECRET", "")

	if _, err := Load(); err == nil {
		t.Fatal("expected missing jwt secret error")
	}
}

func TestLoadConfigRejectsInvalidDuration(t *testing.T) {
	clearOptionalAuthEnv(t)
	t.Setenv("AUTH_JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("AUTH_ACCESS_TOKEN_TTL", "invalid")

	if _, err := Load(); err == nil {
		t.Fatal("expected invalid duration error")
	}
}

func TestLoadConfigRejectsNonPositiveDuration(t *testing.T) {
	clearOptionalAuthEnv(t)
	t.Setenv("AUTH_JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("AUTH_REFRESH_TOKEN_TTL", "0s")

	if _, err := Load(); err == nil {
		t.Fatal("expected non-positive duration error")
	}
}

func TestLoadConfigRejectsInvalidInteger(t *testing.T) {
	clearOptionalAuthEnv(t)
	t.Setenv("AUTH_JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("AUTH_BCRYPT_COST", "invalid")

	if _, err := Load(); err == nil {
		t.Fatal("expected invalid integer error")
	}
}

func clearOptionalAuthEnv(t *testing.T) {
	t.Helper()

	for _, key := range []string{
		"AUTH_HTTP_ADDR",
		"AUTH_JWT_ISSUER",
		"AUTH_JWT_AUDIENCE",
		"AUTH_ACCESS_TOKEN_TTL",
		"AUTH_REFRESH_TOKEN_TTL",
		"AUTH_REFRESH_TOKEN_SIZE",
		"AUTH_BCRYPT_COST",
	} {
		t.Setenv(key, "")
	}
}
