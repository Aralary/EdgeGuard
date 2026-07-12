package domain

import (
	"errors"
	"testing"
)

func TestNewUser(t *testing.T) {
	user, err := NewUser(" User@Example.COM ", "password-hash", RoleMember)
	if err != nil {
		t.Fatalf("NewUser() error = %v", err)
	}

	if user.Email != "user@example.com" {
		t.Fatalf("Email = %q, want %q", user.Email, "user@example.com")
	}

	if user.Role != RoleMember {
		t.Fatalf("Role = %q, want %q", user.Role, RoleMember)
	}

	if !user.Enabled {
		t.Fatal("Enabled = false, want true")
	}
}

func TestNewUserValidation(t *testing.T) {
	tests := []struct {
		name         string
		email        string
		passwordHash string
		role         Role
		wantErr      error
	}{
		{
			name:         "invalid email",
			email:        "not-an-email",
			passwordHash: "hash",
			role:         RoleMember,
			wantErr:      ErrInvalidEmail,
		},
		{
			name:         "empty password hash",
			email:        "user@example.com",
			passwordHash: " ",
			role:         RoleMember,
			wantErr:      ErrInvalidPasswordHash,
		},
		{
			name:         "invalid role",
			email:        "user@example.com",
			passwordHash: "hash",
			role:         Role("owner"),
			wantErr:      ErrInvalidRole,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewUser(tt.email, tt.passwordHash, tt.role)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewUser() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
