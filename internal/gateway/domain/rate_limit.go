package domain

import "time"

type RateLimitPolicy struct {
	Enabled bool
	Limit   int
	Window  time.Duration
}
