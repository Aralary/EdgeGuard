package ratelimit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aralary/edgeguard/internal/gateway/domain"
	"github.com/redis/go-redis/v9"
)

const defaultKeyPrefix = "edgeguard:rate-limit"

const fixedWindowScript = `
local current = redis.call("INCR", KEYS[1])
if current == 1 then
    redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
local ttl = redis.call("PTTL", KEYS[1])
return {current, ttl}
`

type scriptEvaluator interface {
	Eval(
		ctx context.Context,
		script string,
		keys []string,
		args ...any,
	) (any, error)
}

type redisEvaluator struct {
	client *redis.Client
	script *redis.Script
}

func (e redisEvaluator) Eval(
	ctx context.Context,
	script string,
	keys []string,
	args ...any,
) (any, error) {
	return e.script.Run(ctx, e.client, keys, args...).Result()
}

var _ interface {
	Allow(context.Context, domain.RateLimitRequest) (domain.RateLimitResult, error)
} = (*Limiter)(nil)

type Limiter struct {
	client    *redis.Client
	evaluator scriptEvaluator
	keyPrefix string
	now       func() time.Time
}

func New(redisURL string, keyPrefix string) (*Limiter, error) {
	redisURL = strings.TrimSpace(redisURL)
	if redisURL == "" {
		return nil, errors.New("redis url is required")
	}

	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	options.MaxRetries = -1
	options.DialTimeout = 500 * time.Millisecond
	options.ReadTimeout = 500 * time.Millisecond
	options.WriteTimeout = 500 * time.Millisecond

	client := redis.NewClient(options)

	keyPrefix = strings.TrimSpace(keyPrefix)
	if keyPrefix == "" {
		keyPrefix = defaultKeyPrefix
	}

	return &Limiter{
		client: client,
		evaluator: redisEvaluator{
			client: client,
			script: redis.NewScript(fixedWindowScript),
		},
		keyPrefix: keyPrefix,
		now:       time.Now,
	}, nil
}

func (l *Limiter) Ping(ctx context.Context) error {
	if l == nil || l.client == nil {
		return errors.New("redis limiter is not configured")
	}

	if err := l.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}

	return nil
}

func (l *Limiter) Close() error {
	if l == nil || l.client == nil {
		return nil
	}

	return l.client.Close()
}

func (l *Limiter) Allow(
	ctx context.Context,
	request domain.RateLimitRequest,
) (domain.RateLimitResult, error) {
	if request.Policy.Limit <= 0 || request.Policy.Window <= 0 {
		return domain.RateLimitResult{}, domain.ErrInvalidRateLimitPolicy
	}
	if strings.TrimSpace(request.RouteKey) == "" || strings.TrimSpace(request.ClientID) == "" {
		return domain.RateLimitResult{}, domain.ErrInvalidRateLimitClient
	}

	now := l.now().UTC()
	windowMilliseconds := request.Policy.Window.Milliseconds()
	if windowMilliseconds <= 0 {
		return domain.RateLimitResult{}, domain.ErrInvalidRateLimitPolicy
	}

	currentWindow := now.UnixMilli() / windowMilliseconds
	resetAt := time.UnixMilli((currentWindow + 1) * windowMilliseconds).UTC()
	ttl := resetAt.Sub(now)
	if ttl < time.Millisecond {
		ttl = time.Millisecond
	}

	key := l.redisKey(request.RouteKey, request.ClientID, currentWindow)
	value, err := l.evaluator.Eval(
		ctx,
		fixedWindowScript,
		[]string{key},
		ttl.Milliseconds(),
	)
	if err != nil {
		return domain.RateLimitResult{}, fmt.Errorf("execute redis rate limit script: %w", err)
	}

	current, redisTTL, err := parseScriptResult(value)
	if err != nil {
		return domain.RateLimitResult{}, err
	}

	if redisTTL > 0 {
		resetAt = now.Add(time.Duration(redisTTL) * time.Millisecond).UTC()
	}

	remaining := 0
	if current < int64(request.Policy.Limit) {
		remaining = request.Policy.Limit - int(current)
	}

	return domain.RateLimitResult{
		Enabled:    true,
		Allowed:    current <= int64(request.Policy.Limit),
		Limit:      request.Policy.Limit,
		Remaining:  remaining,
		ResetAt:    resetAt,
		RetryAfter: maxDuration(resetAt.Sub(now), time.Millisecond),
	}, nil
}

func (l *Limiter) redisKey(routeKey string, clientID string, window int64) string {
	return fmt.Sprintf(
		"%s:%s:%s:%d",
		l.keyPrefix,
		hashKeyPart(routeKey),
		hashKeyPart(clientID),
		window,
	)
}

func hashKeyPart(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:16])
}

func parseScriptResult(value any) (int64, int64, error) {
	items, ok := value.([]any)
	if !ok || len(items) != 2 {
		return 0, 0, fmt.Errorf("unexpected redis rate limit response: %T", value)
	}

	current, err := integerValue(items[0])
	if err != nil {
		return 0, 0, fmt.Errorf("decode redis rate limit counter: %w", err)
	}

	ttl, err := integerValue(items[1])
	if err != nil {
		return 0, 0, fmt.Errorf("decode redis rate limit ttl: %w", err)
	}

	return current, ttl, nil
}

func integerValue(value any) (int64, error) {
	switch typed := value.(type) {
	case int64:
		return typed, nil
	case int:
		return int64(typed), nil
	default:
		return 0, fmt.Errorf("expected integer, got %T", value)
	}
}

func maxDuration(left time.Duration, right time.Duration) time.Duration {
	if left > right {
		return left
	}

	return right
}
