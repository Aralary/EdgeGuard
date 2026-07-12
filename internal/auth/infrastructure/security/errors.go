package security

import "errors"

var (
	ErrInvalidBcryptCost       = errors.New("invalid bcrypt cost")
	ErrJWTSecretTooShort       = errors.New("jwt secret must contain at least 32 bytes")
	ErrInvalidAccessTokenTTL   = errors.New("invalid access token ttl")
	ErrInvalidRefreshTokenSize = errors.New("invalid refresh token size")
	ErrInvalidTokenUser        = errors.New("invalid token user")
	ErrInvalidAccessToken      = errors.New("invalid access token")
	ErrInvalidAPIKey           = errors.New("invalid api key")
)
