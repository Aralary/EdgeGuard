package controlplane

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientListRoutes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("request method = %q, want %q", r.Method, http.MethodGet)
		}

		if r.URL.Path != "/internal/v1/routes" {
			t.Fatalf("request path = %q, want %q", r.URL.Path, "/internal/v1/routes")
		}

		if r.Header.Get("Accept") != "application/json" {
			t.Fatalf("Accept header = %q, want %q", r.Header.Get("Accept"), "application/json")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{
				"name":"demo-api-v1",
				"path_prefix":"/api/v1/",
				"upstream_url":"http://demo-backend:8081",
				"strip_prefix":true,
				"timeout_ms":3000
			}
		]`))
	}))
	defer server.Close()

	client, err := New(server.URL, server.Client())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	routes, err := client.ListRoutes(context.Background())
	if err != nil {
		t.Fatalf("ListRoutes() error = %v", err)
	}

	if len(routes) != 1 {
		t.Fatalf("routes count = %d, want 1", len(routes))
	}

	route := routes[0]
	if route.Name != "demo-api-v1" {
		t.Fatalf("route name = %q, want %q", route.Name, "demo-api-v1")
	}
	if route.PathPrefix != "/api/v1" {
		t.Fatalf("route path prefix = %q, want %q", route.PathPrefix, "/api/v1")
	}
	if route.UpstreamURL != "http://demo-backend:8081" {
		t.Fatalf("route upstream url = %q, want %q", route.UpstreamURL, "http://demo-backend:8081")
	}
	if !route.StripPrefix {
		t.Fatal("route strip prefix = false, want true")
	}
	if route.Timeout != 3*time.Second {
		t.Fatalf("route timeout = %s, want %s", route.Timeout, 3*time.Second)
	}
}

func TestClientListRoutesReturnsEmptySlice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client, err := New(server.URL, server.Client())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	routes, err := client.ListRoutes(context.Background())
	if err != nil {
		t.Fatalf("ListRoutes() error = %v", err)
	}

	if routes == nil {
		t.Fatal("routes = nil, want empty non-nil slice")
	}
	if len(routes) != 0 {
		t.Fatalf("routes count = %d, want 0", len(routes))
	}
}

func TestClientListRoutesReturnsStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "control plane unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client, err := New(server.URL, server.Client())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.ListRoutes(context.Background())
	if err == nil {
		t.Fatal("ListRoutes() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "status 503") {
		t.Fatalf("ListRoutes() error = %q, want status 503", err)
	}
}

func TestClientListRoutesRejectsInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"name":`))
	}))
	defer server.Close()

	client, err := New(server.URL, server.Client())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.ListRoutes(context.Background())
	if err == nil {
		t.Fatal("ListRoutes() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "decode control plane routes response") {
		t.Fatalf("ListRoutes() error = %q, want decode error", err)
	}
}

func TestClientListRoutesRejectsInvalidRoutes(t *testing.T) {
	tests := []struct {
		name        string
		response    string
		wantErrPart string
	}{
		{
			name: "empty name",
			response: `[{
				"name":"",
				"path_prefix":"/api/v1",
				"upstream_url":"http://demo-backend:8081",
				"strip_prefix":true,
				"timeout_ms":3000
			}]`,
			wantErrPart: "name is required",
		},
		{
			name: "invalid path prefix",
			response: `[{
				"name":"demo-api-v1",
				"path_prefix":"api/v1",
				"upstream_url":"http://demo-backend:8081",
				"strip_prefix":true,
				"timeout_ms":3000
			}]`,
			wantErrPart: "path prefix must start with /",
		},
		{
			name: "invalid upstream url",
			response: `[{
				"name":"demo-api-v1",
				"path_prefix":"/api/v1",
				"upstream_url":"demo-backend:8081",
				"strip_prefix":true,
				"timeout_ms":3000
			}]`,
			wantErrPart: "upstream url must be an absolute http or https url",
		},
		{
			name: "invalid timeout",
			response: `[{
				"name":"demo-api-v1",
				"path_prefix":"/api/v1",
				"upstream_url":"http://demo-backend:8081",
				"strip_prefix":true,
				"timeout_ms":0
			}]`,
			wantErrPart: "timeout_ms must be greater than zero",
		},
		{
			name: "duplicate normalized path prefix",
			response: `[
				{
					"name":"demo-api-v1",
					"path_prefix":"/api/v1",
					"upstream_url":"http://demo-backend:8081",
					"strip_prefix":true,
					"timeout_ms":3000
				},
				{
					"name":"another-demo-api-v1",
					"path_prefix":"/api/v1/",
					"upstream_url":"http://demo-backend:8081",
					"strip_prefix":true,
					"timeout_ms":3000
				}
			]`,
			wantErrPart: `duplicate path prefix "/api/v1"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client, err := New(server.URL, server.Client())
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			_, err = client.ListRoutes(context.Background())
			if err == nil {
				t.Fatal("ListRoutes() error = nil, want error")
			}

			if !strings.Contains(err.Error(), tt.wantErrPart) {
				t.Fatalf("ListRoutes() error = %q, want substring %q", err, tt.wantErrPart)
			}
		})
	}
}

func TestClientListRoutesPropagatesContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	client, err := New(server.URL, server.Client())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = client.ListRoutes(ctx)
	if err == nil {
		t.Fatal("ListRoutes() error = nil, want error")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ListRoutes() error = %v, want context.Canceled", err)
	}
}

func TestNewRejectsInvalidBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
	}{
		{name: "empty", baseURL: ""},
		{name: "missing scheme", baseURL: "control-plane:8082"},
		{name: "unsupported scheme", baseURL: "ftp://control-plane:8082"},
		{name: "missing host", baseURL: "http:///internal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := New(tt.baseURL, nil); err == nil {
				t.Fatal("New() error = nil, want error")
			}
		})
	}
}
