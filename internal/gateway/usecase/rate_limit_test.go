package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aralary/edgeguard/internal/gateway/domain"
)

type fakeRateLimiter struct {
	request domain.RateLimitRequest
	result  domain.RateLimitResult
	err     error
	calls   int
}

func (l *fakeRateLimiter) Allow(
	_ context.Context,
	request domain.RateLimitRequest,
) (domain.RateLimitResult, error) {
	l.calls++
	l.request = request
	return l.result, l.err
}

func TestCheckRateLimitSkipsDisabledPolicy(t *testing.T) {
	limiter := &fakeRateLimiter{err: errors.New("must not be called")}
	uc := New(fakeRouteRepository{}, nil, nil, limiter)

	result, err := uc.CheckRateLimit(context.Background(), domain.Route{}, "ip:127.0.0.1")
	if err != nil {
		t.Fatalf("CheckRateLimit() error = %v", err)
	}
	if result.Enabled {
		t.Fatal("result.Enabled = true, want false")
	}
	if !result.Allowed {
		t.Fatal("result.Allowed = false, want true")
	}
	if limiter.calls != 0 {
		t.Fatalf("limiter calls = %d, want 0", limiter.calls)
	}
}

func TestCheckRateLimitUsesRouteAndClientIdentity(t *testing.T) {
	want := domain.RateLimitResult{
		Enabled:   true,
		Allowed:   true,
		Limit:     5,
		Remaining: 4,
	}
	limiter := &fakeRateLimiter{result: want}
	uc := New(fakeRouteRepository{}, nil, nil, limiter)

	route := domain.Route{
		ProjectID:  "project-1",
		Name:       "orders",
		PathPrefix: "/api/orders",
		RateLimit: domain.RateLimitPolicy{
			Enabled: true,
			Limit:   5,
			Window:  time.Minute,
		},
	}

	result, err := uc.CheckRateLimit(context.Background(), route, "api-key:key-1")
	if err != nil {
		t.Fatalf("CheckRateLimit() error = %v", err)
	}
	if result != want {
		t.Fatalf("result = %+v, want %+v", result, want)
	}
	if limiter.request.RouteKey != "project-1\x00orders\x00/api/orders" {
		t.Fatalf("route key = %q", limiter.request.RouteKey)
	}
	if limiter.request.ClientID != "api-key:key-1" {
		t.Fatalf("client id = %q", limiter.request.ClientID)
	}
	if limiter.request.Policy != route.RateLimit {
		t.Fatalf("policy = %+v, want %+v", limiter.request.Policy, route.RateLimit)
	}
}

func TestCheckRateLimitFailsWithoutLimiter(t *testing.T) {
	uc := New(fakeRouteRepository{}, nil, nil, nil)

	_, err := uc.CheckRateLimit(context.Background(), domain.Route{
		RateLimit: domain.RateLimitPolicy{Enabled: true, Limit: 5, Window: time.Minute},
	}, "ip:127.0.0.1")
	if !errors.Is(err, domain.ErrRateLimiterNotConfigured) {
		t.Fatalf("error = %v, want ErrRateLimiterNotConfigured", err)
	}
}

func TestCheckRateLimitValidatesPolicy(t *testing.T) {
	uc := New(fakeRouteRepository{}, nil, nil, &fakeRateLimiter{})

	_, err := uc.CheckRateLimit(context.Background(), domain.Route{
		RateLimit: domain.RateLimitPolicy{Enabled: true, Limit: 0, Window: time.Minute},
	}, "ip:127.0.0.1")
	if !errors.Is(err, domain.ErrInvalidRateLimitPolicy) {
		t.Fatalf("error = %v, want ErrInvalidRateLimitPolicy", err)
	}
}

func TestCheckRateLimitValidatesClientID(t *testing.T) {
	uc := New(fakeRouteRepository{}, nil, nil, &fakeRateLimiter{})

	_, err := uc.CheckRateLimit(context.Background(), domain.Route{
		RateLimit: domain.RateLimitPolicy{Enabled: true, Limit: 5, Window: time.Minute},
	}, " ")
	if !errors.Is(err, domain.ErrInvalidRateLimitClient) {
		t.Fatalf("error = %v, want ErrInvalidRateLimitClient", err)
	}
}

func TestCheckRateLimitPropagatesLimiterError(t *testing.T) {
	limiterErr := errors.New("redis unavailable")
	uc := New(fakeRouteRepository{}, nil, nil, &fakeRateLimiter{err: limiterErr})

	_, err := uc.CheckRateLimit(context.Background(), domain.Route{
		RateLimit: domain.RateLimitPolicy{Enabled: true, Limit: 5, Window: time.Minute},
	}, "ip:127.0.0.1")
	if !errors.Is(err, limiterErr) {
		t.Fatalf("error = %v, want %v", err, limiterErr)
	}
}
