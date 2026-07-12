package usecase

import "errors"

var (
	ErrInvalidPassword      = errors.New("invalid password")
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrUserDisabled         = errors.New("user disabled")
	ErrEmailAlreadyExists   = errors.New("email already exists")
	ErrUserNotFound         = errors.New("user not found")
	ErrInvalidRefreshToken  = errors.New("invalid refresh token")
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
)
