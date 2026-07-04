package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aralary/edgeguard/internal/gateway/domain"
)

func TestHTTPUtilProxyServeHTTP(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/orders/ord_1" {
			t.Fatalf("upstream path = %q, want %q", r.URL.Path, "/orders/ord_1")
		}

		if r.URL.RawQuery != "expand=true" {
			t.Fatalf("upstream raw query = %q, want %q", r.URL.RawQuery, "expand=true")
		}

		if r.Header.Get("X-Forwarded-Host") != "api.example.com" {
			t.Fatalf("X-Forwarded-Host = %q, want %q", r.Header.Get("X-Forwarded-Host"), "api.example.com")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	req := httptest.NewRequest(http.MethodGet, "http://api.example.com/api/v1/orders/ord_1?expand=true", nil)
	req.Host = "api.example.com"
	res := httptest.NewRecorder()

	route := domain.Route{
		Name:        "demo-api-v1",
		PathPrefix:  "/api/v1",
		UpstreamURL: upstream.URL,
		StripPrefix: true,
		Timeout:     time.Second,
	}

	NewHTTPUtilProxy().ServeHTTP(res, req, route)

	if res.Code != http.StatusAccepted {
		t.Fatalf("response status = %d, want %d", res.Code, http.StatusAccepted)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	if string(body) != `{"ok":true}` {
		t.Fatalf("response body = %q, want %q", string(body), `{"ok":true}`)
	}
}

func TestHTTPUtilProxyServeHTTPReturnsBadGatewayForInvalidUpstreamURL(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://api.example.com/api/v1/orders", nil)
	res := httptest.NewRecorder()

	route := domain.Route{
		Name:        "bad-upstream",
		PathPrefix:  "/api/v1",
		UpstreamURL: "://bad-url",
		StripPrefix: true,
		Timeout:     time.Second,
	}

	NewHTTPUtilProxy().ServeHTTP(res, req, route)

	if res.Code != http.StatusBadGateway {
		t.Fatalf("response status = %d, want %d", res.Code, http.StatusBadGateway)
	}
}

func TestJoinPaths(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want string
	}{
		{name: "empty base path", a: "", b: "/orders", want: "/orders"},
		{name: "empty request path", a: "/api", b: "", want: "/api"},
		{name: "both have slash", a: "/api/", b: "/orders", want: "/api/orders"},
		{name: "none has slash", a: "/api", b: "orders", want: "/api/orders"},
		{name: "single slash between paths", a: "/api", b: "/orders", want: "/api/orders"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinPaths(tt.a, tt.b)
			if got != tt.want {
				t.Fatalf("joinPaths() = %q, want %q", got, tt.want)
			}
		})
	}
}
