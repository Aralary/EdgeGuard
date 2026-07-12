package httpdelivery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aralary/edgeguard/internal/gateway/domain"
	"github.com/labstack/echo/v5"
)

type gatewayUsecaseStub struct {
	route        domain.Route
	resolveErr   error
	authorizeErr error
}

func (s gatewayUsecaseStub) ResolveRoute(context.Context, string) (domain.Route, error) {
	return s.route, s.resolveErr
}

func (s gatewayUsecaseStub) AuthorizeRoute(context.Context, domain.Route, string) error {
	return s.authorizeErr
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
	recorder := performGatewayRequest(
		t,
		gatewayUsecaseStub{
			route:        domain.Route{AuthRequired: true},
			authorizeErr: domain.ErrAPIKeyRequired,
		},
		proxy,
		"",
	)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
	if proxy.called {
		t.Fatal("proxy was called for unauthorized request")
	}
}

func TestProtectedRouteRejectsDifferentProject(t *testing.T) {
	proxy := &proxyStub{}
	recorder := performGatewayRequest(
		t,
		gatewayUsecaseStub{
			route:        domain.Route{AuthRequired: true},
			authorizeErr: domain.ErrAPIKeyProjectMismatch,
		},
		proxy,
		"eg_live_other_project",
	)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}
	if proxy.called {
		t.Fatal("proxy was called for forbidden request")
	}
}

func TestProtectedRouteStripsAPIKeyBeforeProxy(t *testing.T) {
	proxy := &proxyStub{}
	recorder := performGatewayRequest(
		t,
		gatewayUsecaseStub{route: domain.Route{AuthRequired: true}},
		proxy,
		"eg_live_valid",
	)

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
	recorder := performGatewayRequest(
		t,
		gatewayUsecaseStub{route: domain.Route{AuthRequired: false}},
		proxy,
		"eg_live_not_forwarded",
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if proxy.apiKeySeen != "" {
		t.Fatalf("upstream received X-API-Key = %q, want empty", proxy.apiKeySeen)
	}
}

func TestAuthorizationServiceFailureReturnsServiceUnavailable(t *testing.T) {
	proxy := &proxyStub{}
	recorder := performGatewayRequest(
		t,
		gatewayUsecaseStub{
			route:        domain.Route{AuthRequired: true},
			authorizeErr: domain.ErrAuthServiceUnavailable,
		},
		proxy,
		"eg_live_valid",
	)

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
