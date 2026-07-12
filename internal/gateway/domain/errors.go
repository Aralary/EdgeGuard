package domain

import "errors"

var (
	ErrRouteNotFound                = errors.New("route not found")
	ErrInvalidRoute                 = errors.New("invalid route")
	ErrAPIKeyRequired               = errors.New("api key is required")
	ErrInvalidAPIKey                = errors.New("invalid api key")
	ErrAPIKeyProjectMismatch        = errors.New("api key is not allowed for this route")
	ErrAuthServiceUnavailable       = errors.New("auth service unavailable")
	ErrAPIKeyValidatorNotConfigured = errors.New("api key validator is not configured")
)
