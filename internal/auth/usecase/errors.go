package usecase

import "errors"

var (
	ErrInvalidPassword           = errors.New("invalid password")
	ErrInvalidCredentials        = errors.New("invalid credentials")
	ErrUserDisabled              = errors.New("user disabled")
	ErrEmailAlreadyExists        = errors.New("email already exists")
	ErrUserNotFound              = errors.New("user not found")
	ErrInvalidRefreshToken       = errors.New("invalid refresh token")
	ErrRefreshTokenNotFound      = errors.New("refresh token not found")
	ErrInvalidAPIKey             = errors.New("invalid api key")
	ErrInvalidAPIKeyID           = errors.New("invalid api key id")
	ErrAPIKeyNotFound            = errors.New("api key not found")
	ErrAPIKeyNameAlreadyExists   = errors.New("api key name already exists")
	ErrProjectNotFound           = errors.New("project not found")
	ErrAPIKeyDependenciesMissing = errors.New("api key dependencies missing")
)
