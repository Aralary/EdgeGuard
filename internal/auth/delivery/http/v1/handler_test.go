package httpdelivery

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aralary/edgeguard/internal/auth/domain"
	"github.com/aralary/edgeguard/internal/auth/usecase"
	"github.com/labstack/echo/v5"
)

type authUsecaseStub struct {
	registerFn func(context.Context, usecase.RegisterInput) (usecase.UserInfo, error)
	loginFn    func(context.Context, usecase.LoginInput) (usecase.Session, error)
	refreshFn  func(context.Context, usecase.RefreshInput) (usecase.TokenPair, error)
	logoutFn   func(context.Context, usecase.LogoutInput) error
}

func (s authUsecaseStub) Register(ctx context.Context, input usecase.RegisterInput) (usecase.UserInfo, error) {
	return s.registerFn(ctx, input)
}

func (s authUsecaseStub) Login(ctx context.Context, input usecase.LoginInput) (usecase.Session, error) {
	return s.loginFn(ctx, input)
}

func (s authUsecaseStub) Refresh(ctx context.Context, input usecase.RefreshInput) (usecase.TokenPair, error) {
	return s.refreshFn(ctx, input)
}

func (s authUsecaseStub) Logout(ctx context.Context, input usecase.LogoutInput) error {
	return s.logoutFn(ctx, input)
}

type testLogger struct{}

func (testLogger) Debug(args ...interface{})                 {}
func (testLogger) Debugf(format string, args ...interface{}) {}
func (testLogger) Info(args ...interface{})                  {}
func (testLogger) Infof(format string, args ...interface{})  {}
func (testLogger) Warn(args ...interface{})                  {}
func (testLogger) Warnf(format string, args ...interface{})  {}
func (testLogger) Error(args ...interface{})                 {}
func (testLogger) Errorf(format string, args ...interface{}) {}

func TestHealth(t *testing.T) {
	recorder := performRequest(t, authUsecaseStub{}, http.MethodGet, "/health", "")

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	assertJSONField(t, recorder.Body.String(), "service", "auth")
}

func TestRegister(t *testing.T) {
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	stub := authUsecaseStub{
		registerFn: func(_ context.Context, input usecase.RegisterInput) (usecase.UserInfo, error) {
			if input.Email != "user@example.com" || input.Password != "password123" {
				t.Fatalf("unexpected register input: %+v", input)
			}

			return usecase.UserInfo{
				ID:        "user-id",
				Email:     input.Email,
				Role:      domain.RoleMember,
				Enabled:   true,
				CreatedAt: now,
				UpdatedAt: now,
			}, nil
		},
	}

	recorder := performRequest(
		t,
		stub,
		http.MethodPost,
		"/api/v1/auth/register",
		`{"email":"user@example.com","password":"password123"}`,
	)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}

	assertJSONField(t, recorder.Body.String(), "id", "user-id")
	assertJSONField(t, recorder.Body.String(), "role", "member")
}

func TestRegisterRejectsUnknownFields(t *testing.T) {
	stub := authUsecaseStub{
		registerFn: func(context.Context, usecase.RegisterInput) (usecase.UserInfo, error) {
			t.Fatal("register must not be called")
			return usecase.UserInfo{}, nil
		},
	}

	recorder := performRequest(
		t,
		stub,
		http.MethodPost,
		"/api/v1/auth/register",
		`{"email":"user@example.com","password":"password123","role":"admin"}`,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	assertJSONField(t, recorder.Body.String(), "error", "invalid request body")
}

func TestRegisterConflict(t *testing.T) {
	stub := authUsecaseStub{
		registerFn: func(context.Context, usecase.RegisterInput) (usecase.UserInfo, error) {
			return usecase.UserInfo{}, usecase.ErrEmailAlreadyExists
		},
	}

	recorder := performRequest(
		t,
		stub,
		http.MethodPost,
		"/api/v1/auth/register",
		`{"email":"user@example.com","password":"password123"}`,
	)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, recorder.Code)
	}
}

func TestLogin(t *testing.T) {
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	stub := authUsecaseStub{
		loginFn: func(_ context.Context, input usecase.LoginInput) (usecase.Session, error) {
			if input.Email != "user@example.com" || input.Password != "password123" {
				t.Fatalf("unexpected login input: %+v", input)
			}

			return usecase.Session{
				User: usecase.UserInfo{
					ID:        "user-id",
					Email:     input.Email,
					Role:      domain.RoleMember,
					Enabled:   true,
					CreatedAt: now,
					UpdatedAt: now,
				},
				Tokens: tokenPairFixture(now),
			}, nil
		},
	}

	recorder := performRequest(
		t,
		stub,
		http.MethodPost,
		"/api/v1/auth/login",
		`{"email":"user@example.com","password":"password123"}`,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var response sessionResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Tokens.AccessToken != "access-token" || response.Tokens.RefreshToken != "refresh-token" {
		t.Fatalf("unexpected tokens: %+v", response.Tokens)
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	stub := authUsecaseStub{
		loginFn: func(context.Context, usecase.LoginInput) (usecase.Session, error) {
			return usecase.Session{}, usecase.ErrInvalidCredentials
		},
	}

	recorder := performRequest(
		t,
		stub,
		http.MethodPost,
		"/api/v1/auth/login",
		`{"email":"user@example.com","password":"wrong-password"}`,
	)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestRefresh(t *testing.T) {
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	stub := authUsecaseStub{
		refreshFn: func(_ context.Context, input usecase.RefreshInput) (usecase.TokenPair, error) {
			if input.RefreshToken != "old-refresh-token" {
				t.Fatalf("unexpected refresh token: %q", input.RefreshToken)
			}
			return tokenPairFixture(now), nil
		},
	}

	recorder := performRequest(
		t,
		stub,
		http.MethodPost,
		"/api/v1/auth/refresh",
		`{"refresh_token":"old-refresh-token"}`,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	assertJSONField(t, recorder.Body.String(), "refresh_token", "refresh-token")
}

func TestRefreshInvalidToken(t *testing.T) {
	stub := authUsecaseStub{
		refreshFn: func(context.Context, usecase.RefreshInput) (usecase.TokenPair, error) {
			return usecase.TokenPair{}, usecase.ErrInvalidRefreshToken
		},
	}

	recorder := performRequest(
		t,
		stub,
		http.MethodPost,
		"/api/v1/auth/refresh",
		`{"refresh_token":"invalid"}`,
	)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestLogout(t *testing.T) {
	stub := authUsecaseStub{
		logoutFn: func(_ context.Context, input usecase.LogoutInput) error {
			if input.RefreshToken != "refresh-token" {
				t.Fatalf("unexpected refresh token: %q", input.RefreshToken)
			}
			return nil
		},
	}

	recorder := performRequest(
		t,
		stub,
		http.MethodPost,
		"/api/v1/auth/logout",
		`{"refresh_token":"refresh-token"}`,
	)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("expected empty body, got %q", recorder.Body.String())
	}
}

func TestInternalError(t *testing.T) {
	stub := authUsecaseStub{
		loginFn: func(context.Context, usecase.LoginInput) (usecase.Session, error) {
			return usecase.Session{}, errors.New("database unavailable")
		},
	}

	recorder := performRequest(
		t,
		stub,
		http.MethodPost,
		"/api/v1/auth/login",
		`{"email":"user@example.com","password":"password123"}`,
	)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	assertJSONField(t, recorder.Body.String(), "error", "internal error")
}

func performRequest(
	t *testing.T,
	stub authUsecaseStub,
	method string,
	path string,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()

	e := echo.New()
	NewHandler(stub, testLogger{}).RegisterRoutes(e)

	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}

	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, request)
	return recorder
}

func tokenPairFixture(now time.Time) usecase.TokenPair {
	return usecase.TokenPair{
		TokenType:             "Bearer",
		AccessToken:           "access-token",
		AccessTokenExpiresAt:  now.Add(15 * time.Minute),
		RefreshToken:          "refresh-token",
		RefreshTokenExpiresAt: now.Add(30 * 24 * time.Hour),
	}
}

func assertJSONField(t *testing.T, body string, field string, expected string) {
	t.Helper()

	var response map[string]any
	if err := json.Unmarshal([]byte(body), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	actual, ok := response[field].(string)
	if !ok {
		t.Fatalf("field %q is not a string: %#v", field, response[field])
	}
	if actual != expected {
		t.Fatalf("expected %s=%q, got %q", field, expected, actual)
	}
}
