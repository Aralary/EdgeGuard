package domain

import (
	"net/mail"
	"strings"
	"time"
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Role         Role
	Enabled      bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewUser(email string, passwordHash string, role Role) (User, error) {
	email = normalizeEmail(email)
	passwordHash = strings.TrimSpace(passwordHash)

	if !isValidEmail(email) {
		return User{}, ErrInvalidEmail
	}

	if passwordHash == "" {
		return User{}, ErrInvalidPasswordHash
	}

	if !role.IsValid() {
		return User{}, ErrInvalidRole
	}

	return User{
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		Enabled:      true,
	}, nil
}

func normalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isValidEmail(value string) bool {
	address, err := mail.ParseAddress(value)
	if err != nil {
		return false
	}

	return address.Address == value
}
