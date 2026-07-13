package usecase

import "errors"

var (
	ErrInvalidGatewayAccessEvent = errors.New("invalid gateway access event")
	ErrInvalidProjectID          = errors.New("invalid project id")
	ErrInvalidTimeRange          = errors.New("invalid analytics time range")
	ErrAnalyticsRangeTooLarge    = errors.New("analytics time range exceeds 90 days")
	ErrInvalidLimit              = errors.New("invalid analytics limit")
)
