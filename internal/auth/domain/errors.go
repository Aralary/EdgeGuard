package domain

import "errors"

var (
	ErrInvalidEmail        = errors.New("invalid email")
	ErrInvalidPasswordHash = errors.New("invalid password hash")
	ErrInvalidRole         = errors.New("invalid role")
	ErrInvalidUserID       = errors.New("invalid user id")
	ErrInvalidProjectID    = errors.New("invalid project id")
	ErrInvalidName         = errors.New("invalid name")
	ErrInvalidTokenHash    = errors.New("invalid token hash")
	ErrInvalidKeyPrefix    = errors.New("invalid key prefix")
	ErrInvalidKeyHash      = errors.New("invalid key hash")
	ErrInvalidExpiration   = errors.New("invalid expiration")
)
