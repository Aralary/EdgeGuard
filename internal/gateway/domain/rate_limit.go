package domain

import "time"

type RateLimitPolicy struct {
	Enabled bool
	Limit   int
	Window  time.Duration
}

type RateLimitRequest struct {
	RouteKey string
	ClientID string
	Policy   RateLimitPolicy
}

type RateLimitResult struct {
	Enabled    bool
	Allowed    bool
	Limit      int
	Remaining  int
	ResetAt    time.Time
	RetryAfter time.Duration
}
