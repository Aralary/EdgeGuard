package domain

type RateLimitPolicy struct {
	Enabled       bool
	Requests      int
	WindowSeconds int
}

func NewRateLimitPolicy(enabled bool, requests int, windowSeconds int) (RateLimitPolicy, error) {
	if !enabled {
		return RateLimitPolicy{}, nil
	}

	if requests <= 0 {
		return RateLimitPolicy{}, ErrInvalidRateLimitRequests
	}

	if windowSeconds <= 0 {
		return RateLimitPolicy{}, ErrInvalidRateLimitWindow
	}

	return RateLimitPolicy{
		Enabled:       true,
		Requests:      requests,
		WindowSeconds: windowSeconds,
	}, nil
}
