package httpdelivery

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/aralary/edgeguard/internal/auth/domain"
	"github.com/aralary/edgeguard/internal/auth/usecase"
)

func TestCreateAPIKeyRequiresAccessToken(t *testing.T) {
	recorder := performRequest(
		t,
		authUsecaseStub{},
		http.MethodPost,
		"/api/v1/projects/project-1/api-keys",
		`{"name":"gateway"}`,
	)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	}
	assertJSONField(t, recorder.Body.String(), "error", "invalid access token")
}

func TestCreateAPIKey(t *testing.T) {
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	stub := authUsecaseStub{
		authenticateAccessTokenFn: func(_ context.Context, rawToken string) (usecase.AccessTokenPrincipal, error) {
			if rawToken != "access-token" {
				t.Fatalf("unexpected access token: %q", rawToken)
			}
			return usecase.AccessTokenPrincipal{UserID: "user-1", Role: domain.RoleMember}, nil
		},
		createAPIKeyFn: func(_ context.Context, input usecase.CreateAPIKeyInput) (usecase.CreatedAPIKey, error) {
			if input.ProjectID != "project-1" || input.Name != "gateway" {
				t.Fatalf("unexpected input: %+v", input)
			}
			return usecase.CreatedAPIKey{
				APIKey: usecase.APIKeyInfo{
					ID:        "key-1",
					ProjectID: "project-1",
					Name:      "gateway",
					KeyPrefix: "0011223344556677",
					Enabled:   true,
					CreatedAt: now,
					UpdatedAt: now,
				},
				Value: "eg_live_0011223344556677_secret",
			}, nil
		},
	}

	recorder := performRequestWithHeaders(
		t,
		stub,
		http.MethodPost,
		"/api/v1/projects/project-1/api-keys",
		`{"name":"gateway"}`,
		map[string]string{"Authorization": "Bearer access-token"},
	)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}

	var response createdAPIKeyResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.APIKey.ID != "key-1" || response.Value != "eg_live_0011223344556677_secret" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestListAPIKeys(t *testing.T) {
	stub := authUsecaseStub{
		authenticateAccessTokenFn: func(context.Context, string) (usecase.AccessTokenPrincipal, error) {
			return usecase.AccessTokenPrincipal{UserID: "user-1", Role: domain.RoleMember}, nil
		},
		listAPIKeysFn: func(_ context.Context, projectID string) ([]usecase.APIKeyInfo, error) {
			if projectID != "project-1" {
				t.Fatalf("unexpected project id: %q", projectID)
			}
			return []usecase.APIKeyInfo{{
				ID:        "key-1",
				ProjectID: projectID,
				Name:      "gateway",
				KeyPrefix: "0011223344556677",
				Enabled:   true,
			}}, nil
		},
	}

	recorder := performRequestWithHeaders(
		t,
		stub,
		http.MethodGet,
		"/api/v1/projects/project-1/api-keys",
		"",
		map[string]string{"Authorization": "Bearer access-token"},
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var response []apiKeyResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response) != 1 || response[0].ID != "key-1" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestRevokeAPIKey(t *testing.T) {
	stub := authUsecaseStub{
		authenticateAccessTokenFn: func(context.Context, string) (usecase.AccessTokenPrincipal, error) {
			return usecase.AccessTokenPrincipal{UserID: "user-1", Role: domain.RoleMember}, nil
		},
		revokeAPIKeyFn: func(_ context.Context, projectID string, apiKeyID string) error {
			if projectID != "project-1" || apiKeyID != "key-1" {
				t.Fatalf("unexpected ids: project=%q key=%q", projectID, apiKeyID)
			}
			return nil
		},
	}

	recorder := performRequestWithHeaders(
		t,
		stub,
		http.MethodDelete,
		"/api/v1/projects/project-1/api-keys/key-1",
		"",
		map[string]string{"Authorization": "Bearer access-token"},
	)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNoContent, recorder.Code, recorder.Body.String())
	}
}

func TestValidateAPIKey(t *testing.T) {
	stub := authUsecaseStub{
		validateAPIKeyFn: func(_ context.Context, rawAPIKey string) (usecase.APIKeyPrincipal, error) {
			if rawAPIKey != "eg_live_prefix_secret" {
				t.Fatalf("unexpected api key: %q", rawAPIKey)
			}
			return usecase.APIKeyPrincipal{APIKeyID: "key-1", ProjectID: "project-1"}, nil
		},
	}

	recorder := performRequestWithHeaders(
		t,
		stub,
		http.MethodPost,
		"/internal/v1/api-keys/validate",
		"",
		map[string]string{apiKeyHeader: "eg_live_prefix_secret"},
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	assertJSONField(t, recorder.Body.String(), "api_key_id", "key-1")
	assertJSONField(t, recorder.Body.String(), "project_id", "project-1")
}

func TestValidateAPIKeyRejectsInvalidKey(t *testing.T) {
	stub := authUsecaseStub{
		validateAPIKeyFn: func(context.Context, string) (usecase.APIKeyPrincipal, error) {
			return usecase.APIKeyPrincipal{}, usecase.ErrInvalidAPIKey
		},
	}

	recorder := performRequest(
		t,
		stub,
		http.MethodPost,
		"/internal/v1/api-keys/validate",
		"",
	)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	}
}
