package domain

import "errors"

var (
	ErrInvalidName              = errors.New("invalid name")
	ErrInvalidProjectID         = errors.New("invalid project id")
	ErrInvalidServiceID         = errors.New("invalid service id")
	ErrInvalidUpstreamURL       = errors.New("invalid upstream url")
	ErrInvalidPathPrefix        = errors.New("invalid path prefix")
	ErrInvalidRateLimitRequests = errors.New("invalid rate limit requests")
	ErrInvalidRateLimitWindow   = errors.New("invalid rate limit window")
)
