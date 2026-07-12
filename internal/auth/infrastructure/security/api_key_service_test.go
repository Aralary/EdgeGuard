package security

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestAPIKeyServiceGenerateAndParse(t *testing.T) {
	service := NewAPIKeyService()
	service.random = bytes.NewReader(bytes.Repeat([]byte{0xAB}, apiKeyPrefixSize+apiKeySecretSize))

	generated, err := service.GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error = %v", err)
	}

	if !strings.HasPrefix(generated.Value, "eg_live_") {
		t.Fatalf("generated value = %q", generated.Value)
	}
	if generated.Prefix != strings.Repeat("ab", apiKeyPrefixSize) {
		t.Fatalf("generated prefix = %q", generated.Prefix)
	}
	if len(generated.Hash) != 64 {
		t.Fatalf("generated hash length = %d, want 64", len(generated.Hash))
	}

	parsed, err := service.ParseAPIKey(generated.Value)
	if err != nil {
		t.Fatalf("ParseAPIKey() error = %v", err)
	}
	if parsed.Prefix != generated.Prefix || parsed.Hash != generated.Hash {
		t.Fatalf("parsed key = %+v, generated = %+v", parsed, generated)
	}
	if !service.MatchAPIKeyHash(parsed.Hash, generated.Hash) {
		t.Fatal("MatchAPIKeyHash() = false, want true")
	}
}

func TestAPIKeyServiceGeneratesUniqueKeys(t *testing.T) {
	service := NewAPIKeyService()

	first, err := service.GenerateAPIKey()
	if err != nil {
		t.Fatalf("first GenerateAPIKey() error = %v", err)
	}
	second, err := service.GenerateAPIKey()
	if err != nil {
		t.Fatalf("second GenerateAPIKey() error = %v", err)
	}

	if first.Value == second.Value || first.Prefix == second.Prefix || first.Hash == second.Hash {
		t.Fatalf("generated keys are not unique: first=%+v second=%+v", first, second)
	}
}

func TestAPIKeyServiceRejectsMalformedKeys(t *testing.T) {
	service := NewAPIKeyService()

	tests := []string{
		"",
		"invalid",
		"eg_test_0011223344556677_secret",
		"eg_live_short_secret",
		"eg_live_0011223344556677_not-base64!",
	}

	for _, rawAPIKey := range tests {
		t.Run(rawAPIKey, func(t *testing.T) {
			_, err := service.ParseAPIKey(rawAPIKey)
			if !errors.Is(err, ErrInvalidAPIKey) {
				t.Fatalf("ParseAPIKey(%q) error = %v, want %v", rawAPIKey, err, ErrInvalidAPIKey)
			}
		})
	}
}

func TestAPIKeyServiceRejectsDifferentOrMalformedHashes(t *testing.T) {
	service := NewAPIKeyService()
	validHash := hashAPIKey("api-key")

	if service.MatchAPIKeyHash(validHash, hashAPIKey("different-key")) {
		t.Fatal("MatchAPIKeyHash() = true for different hashes")
	}
	if service.MatchAPIKeyHash("not-hex", validHash) {
		t.Fatal("MatchAPIKeyHash() = true for malformed actual hash")
	}
	if service.MatchAPIKeyHash(validHash, "not-hex") {
		t.Fatal("MatchAPIKeyHash() = true for malformed expected hash")
	}
}
