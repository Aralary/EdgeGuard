package domain

import (
	"errors"
	"testing"
)

func TestNewRouteKeepsPolicies(t *testing.T) {
	route, err := NewRoute(
		"service-1",
		"protected-orders",
		"/api/v1",
		true,
		0,
		true,
		true,
		RateLimitPolicy{
			Enabled:       true,
			Requests:      100,
			WindowSeconds: 60,
		},
	)
	if err != nil {
		t.Fatalf("NewRoute() error = %v", err)
	}

	if !route.AuthRequired {
		t.Fatal("AuthRequired = false, want true")
	}
	if route.TimeoutMS != DefaultRouteTimeoutMS {
		t.Fatalf("TimeoutMS = %d, want %d", route.TimeoutMS, DefaultRouteTimeoutMS)
	}
	if !route.RateLimit.Enabled || route.RateLimit.Requests != 100 || route.RateLimit.WindowSeconds != 60 {
		t.Fatalf("RateLimit = %#v, want enabled 100 requests per 60 seconds", route.RateLimit)
	}
}

func TestNewRouteNormalizesDisabledRateLimit(t *testing.T) {
	route, err := NewRoute(
		"service-1",
		"public-orders",
		"/api/v1",
		true,
		3000,
		true,
		false,
		RateLimitPolicy{
			Enabled:       false,
			Requests:      100,
			WindowSeconds: 60,
		},
	)
	if err != nil {
		t.Fatalf("NewRoute() error = %v", err)
	}

	if route.RateLimit != (RateLimitPolicy{}) {
		t.Fatalf("RateLimit = %#v, want disabled zero policy", route.RateLimit)
	}
}

func TestNewRouteRejectsInvalidRateLimit(t *testing.T) {
	tests := []struct {
		name    string
		policy  RateLimitPolicy
		wantErr error
	}{
		{
			name: "zero requests",
			policy: RateLimitPolicy{
				Enabled:       true,
				Requests:      0,
				WindowSeconds: 60,
			},
			wantErr: ErrInvalidRateLimitRequests,
		},
		{
			name: "zero window",
			policy: RateLimitPolicy{
				Enabled:       true,
				Requests:      100,
				WindowSeconds: 0,
			},
			wantErr: ErrInvalidRateLimitWindow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewRoute(
				"service-1",
				"orders",
				"/api/v1",
				true,
				3000,
				true,
				false,
				tt.policy,
			)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewRoute() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
