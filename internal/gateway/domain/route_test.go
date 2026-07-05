package domain

import (
	"testing"
	"time"
)

func TestRouteMatches(t *testing.T) {
	tests := []struct {
		name      string
		route     Route
		path      string
		wantMatch bool
	}{
		{
			name:      "root route matches any path",
			route:     Route{PathPrefix: "/"},
			path:      "/api/v1/orders",
			wantMatch: true,
		},
		{
			name:      "exact prefix match",
			route:     Route{PathPrefix: "/api/v1"},
			path:      "/api/v1",
			wantMatch: true,
		},
		{
			name:      "nested path match",
			route:     Route{PathPrefix: "/api/v1"},
			path:      "/api/v1/orders",
			wantMatch: true,
		},
		{
			name:      "prefix boundary is respected",
			route:     Route{PathPrefix: "/api/v1"},
			path:      "/api/v10/orders",
			wantMatch: false,
		},
		{
			name:      "trailing slash in route prefix is ignored for exact match",
			route:     Route{PathPrefix: "/api/v1/"},
			path:      "/api/v1",
			wantMatch: true,
		},
		{
			name:      "trailing slash in route prefix is ignored for nested path",
			route:     Route{PathPrefix: "/api/v1/"},
			path:      "/api/v1/orders",
			wantMatch: true,
		},
		{
			name:      "different prefix does not match",
			route:     Route{PathPrefix: "/api/v1"},
			path:      "/internal/orders",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.route.Matches(tt.path)
			if got != tt.wantMatch {
				t.Fatalf("Matches() = %v, want %v", got, tt.wantMatch)
			}
		})
	}
}

func TestRouteUpstreamPath(t *testing.T) {
	tests := []struct {
		name         string
		route        Route
		originalPath string
		wantPath     string
	}{
		{
			name:         "strip disabled keeps original path",
			route:        Route{PathPrefix: "/api/v1", StripPrefix: false},
			originalPath: "/api/v1/orders",
			wantPath:     "/api/v1/orders",
		},
		{
			name:         "strip exact prefix returns root",
			route:        Route{PathPrefix: "/api/v1", StripPrefix: true},
			originalPath: "/api/v1",
			wantPath:     "/",
		},
		{
			name:         "strip prefix from nested path",
			route:        Route{PathPrefix: "/api/v1", StripPrefix: true},
			originalPath: "/api/v1/orders/ord_1",
			wantPath:     "/orders/ord_1",
		},
		{
			name:         "strip prefix with trailing slash in config",
			route:        Route{PathPrefix: "/api/v1/", StripPrefix: true},
			originalPath: "/api/v1/orders",
			wantPath:     "/orders",
		},
		{
			name:         "root prefix keeps original path",
			route:        Route{PathPrefix: "/", StripPrefix: true},
			originalPath: "/orders",
			wantPath:     "/orders",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.route.UpstreamPath(tt.originalPath)
			if got != tt.wantPath {
				t.Fatalf("UpstreamPath() = %q, want %q", got, tt.wantPath)
			}
		})
	}
}

func TestRouteKeepsTimeoutAsDomainValue(t *testing.T) {
	route := Route{Timeout: 3 * time.Second}

	if route.Timeout != 3*time.Second {
		t.Fatalf("Timeout = %s, want %s", route.Timeout, 3*time.Second)
	}
}
