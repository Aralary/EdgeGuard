package ratelimit

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aralary/edgeguard/internal/gateway/domain"
)

type fakeEvaluator struct {
	value  any
	err    error
	script string
	keys   []string
	args   []any
}

func (e *fakeEvaluator) Eval(
	_ context.Context,
	script string,
	keys []string,
	args ...any,
) (any, error) {
	e.script = script
	e.keys = append([]string(nil), keys...)
	e.args = append([]any(nil), args...)
	return e.value, e.err
}

func TestLimiterAllowsRequestAndBuildsPrivateRedisKey(t *testing.T) {
	fixedNow := time.Date(2026, 7, 13, 12, 0, 5, 0, time.UTC)
	evaluator := &fakeEvaluator{value: []any{int64(1), int64(55_000)}}
	limiter := &Limiter{
		evaluator: evaluator,
		keyPrefix: "edgeguard:test",
		now:       func() time.Time { return fixedNow },
	}

	result, err := limiter.Allow(context.Background(), domain.RateLimitRequest{
		RouteKey: "project-1\x00orders\x00/api/orders",
		ClientID: "api-key:key-1",
		Policy: domain.RateLimitPolicy{
			Enabled: true,
			Limit:   3,
			Window:  time.Minute,
		},
	})
	if err != nil {
		t.Fatalf("Allow() error = %v", err)
	}
	if !result.Allowed || result.Remaining != 2 || result.Limit != 3 {
		t.Fatalf("result = %+v", result)
	}
	if result.RetryAfter != 55*time.Second {
		t.Fatalf("RetryAfter = %s, want 55s", result.RetryAfter)
	}
	if len(evaluator.keys) != 1 {
		t.Fatalf("keys = %#v", evaluator.keys)
	}
	key := evaluator.keys[0]
	if !strings.HasPrefix(key, "edgeguard:test:") {
		t.Fatalf("key = %q", key)
	}
	if strings.Contains(key, "project-1") || strings.Contains(key, "key-1") {
		t.Fatalf("redis key leaks route/client identifiers: %q", key)
	}
	if len(evaluator.args) != 1 || evaluator.args[0] != int64(55_000) {
		t.Fatalf("script args = %#v, want [55000]", evaluator.args)
	}
}

func TestLimiterRejectsRequestAboveLimit(t *testing.T) {
	fixedNow := time.Date(2026, 7, 13, 12, 0, 5, 0, time.UTC)
	limiter := &Limiter{
		evaluator: &fakeEvaluator{value: []any{int64(4), int64(10_000)}},
		keyPrefix: "edgeguard:test",
		now:       func() time.Time { return fixedNow },
	}

	result, err := limiter.Allow(context.Background(), domain.RateLimitRequest{
		RouteKey: "route",
		ClientID: "client",
		Policy:   domain.RateLimitPolicy{Enabled: true, Limit: 3, Window: time.Minute},
	})
	if err != nil {
		t.Fatalf("Allow() error = %v", err)
	}
	if result.Allowed {
		t.Fatal("Allowed = true, want false")
	}
	if result.Remaining != 0 {
		t.Fatalf("Remaining = %d, want 0", result.Remaining)
	}
}

func TestLimiterPropagatesRedisError(t *testing.T) {
	redisErr := errors.New("redis unavailable")
	limiter := &Limiter{
		evaluator: &fakeEvaluator{err: redisErr},
		keyPrefix: "edgeguard:test",
		now:       time.Now,
	}

	_, err := limiter.Allow(context.Background(), domain.RateLimitRequest{
		RouteKey: "route",
		ClientID: "client",
		Policy:   domain.RateLimitPolicy{Enabled: true, Limit: 3, Window: time.Minute},
	})
	if !errors.Is(err, redisErr) {
		t.Fatalf("error = %v, want %v", err, redisErr)
	}
}

func TestLimiterValidatesInput(t *testing.T) {
	limiter := &Limiter{
		evaluator: &fakeEvaluator{},
		keyPrefix: "edgeguard:test",
		now:       time.Now,
	}

	_, err := limiter.Allow(context.Background(), domain.RateLimitRequest{
		RouteKey: "route",
		ClientID: "client",
		Policy:   domain.RateLimitPolicy{Enabled: true, Limit: 0, Window: time.Minute},
	})
	if !errors.Is(err, domain.ErrInvalidRateLimitPolicy) {
		t.Fatalf("error = %v, want ErrInvalidRateLimitPolicy", err)
	}

	_, err = limiter.Allow(context.Background(), domain.RateLimitRequest{
		RouteKey: "",
		ClientID: "client",
		Policy:   domain.RateLimitPolicy{Enabled: true, Limit: 3, Window: time.Minute},
	})
	if !errors.Is(err, domain.ErrInvalidRateLimitClient) {
		t.Fatalf("error = %v, want ErrInvalidRateLimitClient", err)
	}
}

func TestParseScriptResultRejectsUnexpectedResponse(t *testing.T) {
	_, _, err := parseScriptResult("unexpected")
	if err == nil {
		t.Fatal("parseScriptResult() error = nil")
	}
}

func TestRedisKeyChangesByWindow(t *testing.T) {
	limiter := &Limiter{keyPrefix: "edgeguard:test"}

	first := limiter.redisKey("route", "client", 1)
	second := limiter.redisKey("route", "client", 2)
	if first == second {
		t.Fatalf("redis keys are equal: %q", first)
	}
}
