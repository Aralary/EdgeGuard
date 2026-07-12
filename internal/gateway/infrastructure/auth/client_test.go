package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aralary/edgeguard/internal/gateway/domain"
)

func TestClientValidateAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/internal/v1/api-keys/validate" {
			t.Fatalf("path = %q, want validation path", r.URL.Path)
		}
		if got := r.Header.Get(apiKeyHeader); got != "eg_live_test_secret" {
			t.Fatalf("X-API-Key = %q, want raw key", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"api_key_id":"key-1","project_id":"project-1"}`))
	}))
	defer server.Close()

	client, err := New(server.URL, server.Client())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	principal, err := client.ValidateAPIKey(context.Background(), "eg_live_test_secret")
	if err != nil {
		t.Fatalf("ValidateAPIKey() error = %v", err)
	}
	if principal.APIKeyID != "key-1" || principal.ProjectID != "project-1" {
		t.Fatalf("principal = %#v", principal)
	}
}

func TestClientValidateAPIKeyMapsUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "invalid api key", http.StatusUnauthorized)
	}))
	defer server.Close()

	client, err := New(server.URL, server.Client())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.ValidateAPIKey(context.Background(), "invalid")
	if !errors.Is(err, domain.ErrInvalidAPIKey) {
		t.Fatalf("error = %v, want ErrInvalidAPIKey", err)
	}
}

func TestClientValidateAPIKeyMapsServiceFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client, err := New(server.URL, server.Client())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.ValidateAPIKey(context.Background(), "key")
	if !errors.Is(err, domain.ErrAuthServiceUnavailable) {
		t.Fatalf("error = %v, want ErrAuthServiceUnavailable", err)
	}
}

func TestClientValidateAPIKeyRejectsInvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"api_key_id":"","project_id":"project-1"}`))
	}))
	defer server.Close()

	client, err := New(server.URL, server.Client())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.ValidateAPIKey(context.Background(), "key")
	if !errors.Is(err, domain.ErrAuthServiceUnavailable) {
		t.Fatalf("error = %v, want ErrAuthServiceUnavailable", err)
	}
}
