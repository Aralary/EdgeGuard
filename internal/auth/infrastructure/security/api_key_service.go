package security

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/aralary/edgeguard/internal/auth/usecase"
)

const (
	apiKeyEnvironment = "live"
	apiKeyPrefixSize  = 8
	apiKeySecretSize  = 32
)

type APIKeyService struct {
	random io.Reader
}

var _ usecase.APIKeyService = (*APIKeyService)(nil)

func NewAPIKeyService() *APIKeyService {
	return &APIKeyService{random: rand.Reader}
}

func (s *APIKeyService) GenerateAPIKey() (usecase.GeneratedAPIKey, error) {
	prefixBytes := make([]byte, apiKeyPrefixSize)
	if _, err := io.ReadFull(s.random, prefixBytes); err != nil {
		return usecase.GeneratedAPIKey{}, fmt.Errorf("generate api key prefix: %w", err)
	}

	secretBytes := make([]byte, apiKeySecretSize)
	if _, err := io.ReadFull(s.random, secretBytes); err != nil {
		return usecase.GeneratedAPIKey{}, fmt.Errorf("generate api key secret: %w", err)
	}

	prefix := hex.EncodeToString(prefixBytes)
	secret := base64.RawURLEncoding.EncodeToString(secretBytes)
	value := fmt.Sprintf("eg_%s_%s_%s", apiKeyEnvironment, prefix, secret)

	return usecase.GeneratedAPIKey{
		Value:  value,
		Prefix: prefix,
		Hash:   hashAPIKey(value),
	}, nil
}

func (s *APIKeyService) ParseAPIKey(rawAPIKey string) (usecase.ParsedAPIKey, error) {
	rawAPIKey = strings.TrimSpace(rawAPIKey)
	parts := strings.SplitN(rawAPIKey, "_", 4)
	if len(parts) != 4 || parts[0] != "eg" || parts[1] != apiKeyEnvironment {
		return usecase.ParsedAPIKey{}, ErrInvalidAPIKey
	}

	prefix := parts[2]
	prefixBytes, err := hex.DecodeString(prefix)
	if err != nil || len(prefixBytes) != apiKeyPrefixSize {
		return usecase.ParsedAPIKey{}, ErrInvalidAPIKey
	}

	secretBytes, err := base64.RawURLEncoding.DecodeString(parts[3])
	if err != nil || len(secretBytes) != apiKeySecretSize {
		return usecase.ParsedAPIKey{}, ErrInvalidAPIKey
	}

	return usecase.ParsedAPIKey{
		Prefix: prefix,
		Hash:   hashAPIKey(rawAPIKey),
	}, nil
}

func (s *APIKeyService) MatchAPIKeyHash(actualHash string, expectedHash string) bool {
	actual, err := hex.DecodeString(actualHash)
	if err != nil {
		return false
	}

	expected, err := hex.DecodeString(expectedHash)
	if err != nil || len(actual) != len(expected) {
		return false
	}

	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func hashAPIKey(rawAPIKey string) string {
	hash := sha256.Sum256([]byte(rawAPIKey))
	return hex.EncodeToString(hash[:])
}
