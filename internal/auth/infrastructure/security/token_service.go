package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aralary/edgeguard/internal/auth/domain"
	"github.com/aralary/edgeguard/internal/auth/usecase"
	"github.com/golang-jwt/jwt/v5"
)

const (
	DefaultJWTIssuer        = "edgeguard-auth"
	DefaultJWTAudience      = "edgeguard"
	DefaultAccessTokenTTL   = 15 * time.Minute
	DefaultRefreshTokenSize = 32

	minimumJWTSecretSize    = 32
	minimumRefreshTokenSize = 16
	maximumRefreshTokenSize = 128
)

type TokenServiceConfig struct {
	JWTSecret        string
	JWTIssuer        string
	JWTAudience      string
	AccessTokenTTL   time.Duration
	RefreshTokenSize int
}

type AccessTokenClaims struct {
	Role domain.Role `json:"role"`
	jwt.RegisteredClaims
}

type TokenService struct {
	secret           []byte
	issuer           string
	audience         string
	accessTokenTTL   time.Duration
	refreshTokenSize int
	random           io.Reader
}

func NewTokenService(config TokenServiceConfig) (*TokenService, error) {
	if len(config.JWTSecret) < minimumJWTSecretSize {
		return nil, ErrJWTSecretTooShort
	}

	issuer := strings.TrimSpace(config.JWTIssuer)
	if issuer == "" {
		issuer = DefaultJWTIssuer
	}

	audience := strings.TrimSpace(config.JWTAudience)
	if audience == "" {
		audience = DefaultJWTAudience
	}

	accessTokenTTL := config.AccessTokenTTL
	if accessTokenTTL == 0 {
		accessTokenTTL = DefaultAccessTokenTTL
	}
	if accessTokenTTL < 0 {
		return nil, ErrInvalidAccessTokenTTL
	}

	refreshTokenSize := config.RefreshTokenSize
	if refreshTokenSize == 0 {
		refreshTokenSize = DefaultRefreshTokenSize
	}
	if refreshTokenSize < minimumRefreshTokenSize || refreshTokenSize > maximumRefreshTokenSize {
		return nil, ErrInvalidRefreshTokenSize
	}

	return &TokenService{
		secret:           []byte(config.JWTSecret),
		issuer:           issuer,
		audience:         audience,
		accessTokenTTL:   accessTokenTTL,
		refreshTokenSize: refreshTokenSize,
		random:           rand.Reader,
	}, nil
}

func (s *TokenService) IssueAccessToken(
	user domain.User,
	now time.Time,
) (usecase.IssuedAccessToken, error) {
	userID := strings.TrimSpace(user.ID)
	if userID == "" || !user.Role.IsValid() || !user.Enabled || now.IsZero() {
		return usecase.IssuedAccessToken{}, ErrInvalidTokenUser
	}

	expiresAt := now.Add(s.accessTokenTTL)
	claims := AccessTokenClaims{
		Role: user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID,
			Audience:  jwt.ClaimStrings{s.audience},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	rawToken, err := token.SignedString(s.secret)
	if err != nil {
		return usecase.IssuedAccessToken{}, fmt.Errorf("sign access token: %w", err)
	}

	return usecase.IssuedAccessToken{
		Value:     rawToken,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *TokenService) ParseAccessToken(
	rawToken string,
	now time.Time,
) (usecase.AccessTokenPrincipal, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" || now.IsZero() {
		return usecase.AccessTokenPrincipal{}, ErrInvalidAccessToken
	}

	claims := &AccessTokenClaims{}
	token, err := jwt.ParseWithClaims(
		rawToken,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, ErrInvalidAccessToken
			}

			return s.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(s.issuer),
		jwt.WithAudience(s.audience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithNotBeforeRequired(),
		jwt.WithTimeFunc(func() time.Time { return now }),
		jwt.WithStrictDecoding(),
	)
	if err != nil {
		return usecase.AccessTokenPrincipal{}, fmt.Errorf("%w: %v", ErrInvalidAccessToken, err)
	}
	if !token.Valid {
		return usecase.AccessTokenPrincipal{}, ErrInvalidAccessToken
	}

	userID := strings.TrimSpace(claims.Subject)
	if userID == "" || !claims.Role.IsValid() || claims.ExpiresAt == nil {
		return usecase.AccessTokenPrincipal{}, ErrInvalidAccessToken
	}

	return usecase.AccessTokenPrincipal{
		UserID:    userID,
		Role:      claims.Role,
		ExpiresAt: claims.ExpiresAt.Time,
	}, nil
}

func (s *TokenService) GenerateRefreshToken() (
	rawToken string,
	tokenHash string,
	err error,
) {
	randomBytes := make([]byte, s.refreshTokenSize)
	if _, err := io.ReadFull(s.random, randomBytes); err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}

	rawToken = base64.RawURLEncoding.EncodeToString(randomBytes)
	return rawToken, s.HashRefreshToken(rawToken), nil
}

func (s *TokenService) HashRefreshToken(rawToken string) string {
	hash := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(hash[:])
}
