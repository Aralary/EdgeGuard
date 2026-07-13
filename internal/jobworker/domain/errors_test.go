package domain

import (
	"errors"
	"testing"
)

func TestPermanentError(t *testing.T) {
	baseError := errors.New("invalid payload")
	err := Permanent(baseError)

	if !IsPermanent(err) {
		t.Fatal("IsPermanent() = false, want true")
	}
	if !errors.Is(err, baseError) {
		t.Fatalf("errors.Is() = false for wrapped error")
	}
}
