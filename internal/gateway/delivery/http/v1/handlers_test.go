package httpdelivery

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aralary/edgeguard/internal/gateway/domain"
	"github.com/labstack/echo/v5"
)

type gatewayUsecaseStub struct {
	route              domain.Route
	resolveErr         error
	principal          domain.APIKeyPrincipal
	authorizeErr       error
	rateLimitResult    domain.RateLimitResult
	rateLimitErr       error
	rateLimitClientID  string
	rateLimitCallCount int
}

func (s *gatewayUsecaseStub) ResolveRoute(context.Context, string) (domain.Route, error) {
	return s.route, s.resolveErr
}

func (s *gatewayUsecaseStub) AuthorizeRoute(
	context.Context,
	domain.Route,
	string,
) (domain.APIKeyPrincipal, error) {
	return s.principal, s.authorizeErr
}

func (s *gatewayUsecaseStub) CheckRateLimit(
	_ context.Context,
	_ domain.Route,
	clientID string,
) (domain.RateLimitResult, error) {
	s.rateLimitClientID = clientID
	s.rateLimitCallCount++
	return s.rateLimitResult, s.rateLimitErr
}

type proxyStub struct {
	called     bool
	apiKeySeen string
}

func (p *proxyStub) ServeHTTP(w http.ResponseWriter, r *http.Request, route domain.Route) {
	p.called = true
	p.apiKeySeen = r.Header.Get(apiKeyHeader)
	w.WriteHeader(http.StatusOK)
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

func TestProtectedRouteRequiresAPIKey(t *testing.T) {
	proxy := &proxyStub{}
	gateway := &gatewayUsecaseStub{
		route:        domain.Route{AuthRequired: true},
		authorizeErr: domain.ErrAPIKeyRequired,
	}

	recorder := performGatewayRequest(t, gateway, proxy, "")

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
	if proxy.called {
		t.Fatal("proxy was called for unauthorized request")
	}
	if gateway.rateLimitCallCount != 0 {
		t.Fatal("rate limiter was called for unauthorized request")
	}
}

func TestProtectedRouteRejectsDifferentProject(t *testing.T) {
	proxy := &proxyStub{}
	gateway := &gatewayUsecaseStub{
		route:        domain.Route{AuthRequired: true},
		authorizeErr: domain.ErrAPIKeyProjectMismatch,
	}

	recorder := performGatewayRequest(t, gateway, proxy, "eg_live_other_project")

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}
	if proxy.called {
		t.Fatal("proxy was called for forbidden request")
	}
}

func TestProtectedRouteUsesAPIKeyIDForRateLimit(t *testing.T) {
	proxy := &proxyStub{}
	gateway := &gatewayUsecaseStub{
		route: domain.Route{
			AuthRequired: true,
			RateLimit: domain.RateLimitPolicy{
				Enabled: true,
				Limit:   10,
				Window:  time.Minute,
			},
		},
		principal: domain.APIKeyPrincipal{APIKeyID: "key-123", ProjectID: "project-1"},
		rateLimitResult: domain.RateLimitResult{
			Enabled:   true,
			Allowed:   true,
			Limit:     10,
			Remaining: 9,
			ResetAt:   time.Unix(1_800_000_000, 0),
		},
	}

	recorder := performGatewayRequest(t, gateway, proxy, "eg_live_valid")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if gateway.rateLimitClientID != "api-key:key-123" {
		t.Fatalf("rate limit client id = %q", gateway.rateLimitClientID)
	}
	if recorder.Header().Get(rateLimitLimitHeader) != "10" {
		t.Fatalf("limit header = %q", recorder.Header().Get(rateLimitLimitHeader))
	}
	if recorder.Header().Get(rateLimitRemainingHeader) != "9" {
		t.Fatalf("remaining header = %q", recorder.Header().Get(rateLimitRemainingHeader))
	}
}

func TestPublicRouteUsesDirectClientIPForRateLimit(t *testing.T) {
	proxy := &proxyStub{}
	gateway := &gatewayUsecaseStub{
		route: domain.Route{
			RateLimit: domain.RateLimitPolicy{
				Enabled: true,
				Limit:   2,
				Window:  time.Minute,
			},
		},
		rateLimitResult: domain.RateLimitResult{Enabled: true, Allowed: true},
	}

	recorder := performGatewayRequest(t, gateway, proxy, "")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if gateway.rateLimitClientID != "ip:192.0.2.1" {
		t.Fatalf("rate limit client id = %q", gateway.rateLimitClientID)
	}
}

func TestRateLimitExceededReturnsTooManyRequests(t *testing.T) {
	proxy := &proxyStub{}
	gateway := &gatewayUsecaseStub{
		route: domain.Route{
			RateLimit: domain.RateLimitPolicy{Enabled: true, Limit: 3, Window: 10 * time.Second},
		},
		rateLimitResult: domain.RateLimitResult{
			Enabled:    true,
			Allowed:    false,
			Limit:      3,
			Remaining:  0,
			ResetAt:    time.Unix(1_800_000_000, 0),
			RetryAfter: 4 * time.Second,
		},
	}

	recorder := performGatewayRequest(t, gateway, proxy, "")

	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", recorder.Code)
	}
	if proxy.called {
		t.Fatal("proxy was called after rate limit exceeded")
	}
	if recorder.Header().Get("Retry-After") != "4" {
		t.Fatalf("Retry-After = %q", recorder.Header().Get("Retry-After"))
	}
}

func TestRateLimiterFailureIsFailOpen(t *testing.T) {
	proxy := &proxyStub{}
	gateway := &gatewayUsecaseStub{
		route: domain.Route{
			RateLimit: domain.RateLimitPolicy{Enabled: true, Limit: 3, Window: 10 * time.Second},
		},
		rateLimitErr: errors.New("redis unavailable"),
	}

	recorder := performGatewayRequest(t, gateway, proxy, "")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !proxy.called {
		t.Fatal("proxy was not called when limiter failed")
	}
}

func TestProtectedRouteStripsAPIKeyBeforeProxy(t *testing.T) {
	proxy := &proxyStub{}
	gateway := &gatewayUsecaseStub{
		route:     domain.Route{AuthRequired: true},
		principal: domain.APIKeyPrincipal{APIKeyID: "key-1", ProjectID: "project-1"},
	}

	recorder := performGatewayRequest(t, gateway, proxy, "eg_live_valid")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !proxy.called {
		t.Fatal("proxy was not called")
	}
	if proxy.apiKeySeen != "" {
		t.Fatalf("upstream received X-API-Key = %q, want empty", proxy.apiKeySeen)
	}
}

func TestPublicRouteAlsoStripsEdgeGuardAPIKey(t *testing.T) {
	proxy := &proxyStub{}
	gateway := &gatewayUsecaseStub{route: domain.Route{AuthRequired: false}}

	recorder := performGatewayRequest(t, gateway, proxy, "eg_live_not_forwarded")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if proxy.apiKeySeen != "" {
		t.Fatalf("upstream received X-API-Key = %q, want empty", proxy.apiKeySeen)
	}
}

func TestAuthorizationServiceFailureReturnsServiceUnavailable(t *testing.T) {
	proxy := &proxyStub{}
	gateway := &gatewayUsecaseStub{
		route:        domain.Route{AuthRequired: true},
		authorizeErr: domain.ErrAuthServiceUnavailable,
	}

	recorder := performGatewayRequest(t, gateway, proxy, "eg_live_valid")

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", recorder.Code)
	}
	if proxy.called {
		t.Fatal("proxy was called while auth service was unavailable")
	}
}

func performGatewayRequest(
	t *testing.T,
	gatewayUsecase GatewayUsecase,
	proxy UpstreamProxy,
	apiKey string,
) *httptest.ResponseRecorder {
	t.Helper()

	e := echo.New()
	handler := NewHandler(gatewayUsecase, proxy, testLogger{})
	handler.RegisterRoutes(e)

	req := httptest.NewRequest(http.MethodGet, "/protected/orders", nil)
	if apiKey != "" {
		req.Header.Set(apiKeyHeader, apiKey)
	}

	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, req)

	return recorder
}
