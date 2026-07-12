package security

import (
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestPasswordHasherHashAndMatches(t *testing.T) {
	hasher, err := NewPasswordHasher(0)
	if err != nil {
		t.Fatalf("new password hasher: %v", err)
	}

	hash, err := hasher.Hash("correct horse battery staple")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	if hash == "correct horse battery staple" {
		t.Fatal("password was not hashed")
	}

	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		t.Fatalf("read bcrypt cost: %v", err)
	}
	if cost != DefaultBcryptCost {
		t.Fatalf("unexpected bcrypt cost: got %d want %d", cost, DefaultBcryptCost)
	}

	if !hasher.Matches(hash, "correct horse battery staple") {
		t.Fatal("expected matching password")
	}
	if hasher.Matches(hash, "wrong password") {
		t.Fatal("unexpected password match")
	}
	if hasher.Matches("not-a-bcrypt-hash", "password") {
		t.Fatal("malformed hash must not match")
	}
}

func TestNewPasswordHasherRejectsInvalidCost(t *testing.T) {
	_, err := NewPasswordHasher(bcrypt.MaxCost + 1)
	if !errors.Is(err, ErrInvalidBcryptCost) {
		t.Fatalf("unexpected error: %v", err)
	}
}
