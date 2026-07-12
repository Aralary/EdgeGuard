package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "gateway.yaml")

	content := []byte(`http:
  addr: ":9090"

routes:
  - name: demo-api-v1
    path_prefix: /api/v1
    upstream_url: http://localhost:8081
    strip_prefix: true
    timeout_ms: 3000
`)

	if err := os.WriteFile(configPath, content, 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() unexpected error = %v", err)
	}

	if cfg.HTTP.Addr != ":9090" {
		t.Fatalf("HTTP.Addr = %q, want %q", cfg.HTTP.Addr, ":9090")
	}

	if len(cfg.Routes) != 1 {
		t.Fatalf("len(Routes) = %d, want 1", len(cfg.Routes))
	}

	route := cfg.Routes[0]
	if route.Name != "demo-api-v1" {
		t.Fatalf("route.Name = %q, want %q", route.Name, "demo-api-v1")
	}

	if !route.StripPrefix {
		t.Fatal("route.StripPrefix = false, want true")
	}
}

func TestLoadUsesDefaultHTTPAddr(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "gateway.yaml")

	content := []byte(`routes:
  - name: demo-api-v1
    path_prefix: /api/v1
    upstream_url: http://localhost:8081
`)

	if err := os.WriteFile(configPath, content, 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() unexpected error = %v", err)
	}

	if cfg.HTTP.Addr != ":8080" {
		t.Fatalf("HTTP.Addr = %q, want %q", cfg.HTTP.Addr, ":8080")
	}
}

func TestLoadReturnsErrorForMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}

func TestConfigDomainRoutes(t *testing.T) {
	cfg := Config{
		Routes: []RouteConfig{
			{
				ProjectID:              "project-1",
				Name:                   "with-timeout",
				PathPrefix:             "/api/v1",
				UpstreamURL:            "http://localhost:8081",
				StripPrefix:            true,
				TimeoutMS:              3000,
				AuthRequired:           true,
				RateLimitEnabled:       true,
				RateLimitRequests:      100,
				RateLimitWindowSeconds: 60,
			},
			{
				Name:        "default-timeout",
				PathPrefix:  "/internal",
				UpstreamURL: "http://localhost:8082",
				TimeoutMS:   0,
			},
		},
	}

	routes := cfg.DomainRoutes()
	if len(routes) != 2 {
		t.Fatalf("len(routes) = %d, want 2", len(routes))
	}

	if routes[0].Timeout != 3*time.Second {
		t.Fatalf("routes[0].Timeout = %s, want %s", routes[0].Timeout, 3*time.Second)
	}
	if routes[0].ProjectID != "project-1" || !routes[0].AuthRequired {
		t.Fatalf("routes[0] auth policy = %#v", routes[0])
	}
	if !routes[0].RateLimit.Enabled || routes[0].RateLimit.Limit != 100 || routes[0].RateLimit.Window != time.Minute {
		t.Fatalf("routes[0] rate limit = %#v", routes[0].RateLimit)
	}

	if routes[1].Timeout != 5*time.Second {
		t.Fatalf("routes[1].Timeout = %s, want %s", routes[1].Timeout, 5*time.Second)
	}
}
