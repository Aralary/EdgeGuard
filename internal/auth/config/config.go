package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aralary/edgeguard/internal/auth/infrastructure/security"
	"github.com/aralary/edgeguard/internal/auth/usecase"
)

const DefaultHTTPAddr = ":8083"

type Config struct {
	HTTPAddr         string
	JWTSecret        string
	JWTIssuer        string
	JWTAudience      string
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
	RefreshTokenSize int
	BcryptCost       int
}

func Load() (Config, error) {
	jwtSecret := strings.TrimSpace(os.Getenv("AUTH_JWT_SECRET"))
	if jwtSecret == "" {
		return Config{}, fmt.Errorf("AUTH_JWT_SECRET is required")
	}

	accessTokenTTL, err := durationFromEnv(
		"AUTH_ACCESS_TOKEN_TTL",
		security.DefaultAccessTokenTTL,
	)
	if err != nil {
		return Config{}, err
	}

	refreshTokenTTL, err := durationFromEnv(
		"AUTH_REFRESH_TOKEN_TTL",
		usecase.DefaultRefreshTokenTTL,
	)
	if err != nil {
		return Config{}, err
	}

	refreshTokenSize, err := intFromEnv(
		"AUTH_REFRESH_TOKEN_SIZE",
		security.DefaultRefreshTokenSize,
	)
	if err != nil {
		return Config{}, err
	}

	bcryptCost, err := intFromEnv(
		"AUTH_BCRYPT_COST",
		security.DefaultBcryptCost,
	)
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTPAddr:         stringFromEnv("AUTH_HTTP_ADDR", DefaultHTTPAddr),
		JWTSecret:        jwtSecret,
		JWTIssuer:        stringFromEnv("AUTH_JWT_ISSUER", security.DefaultJWTIssuer),
		JWTAudience:      stringFromEnv("AUTH_JWT_AUDIENCE", security.DefaultJWTAudience),
		AccessTokenTTL:   accessTokenTTL,
		RefreshTokenTTL:  refreshTokenTTL,
		RefreshTokenSize: refreshTokenSize,
		BcryptCost:       bcryptCost,
	}, nil
}

func stringFromEnv(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

func durationFromEnv(key string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be positive", key)
	}

	return parsed, nil
}

func intFromEnv(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}

	return parsed, nil
}
