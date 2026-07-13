package domain

import "errors"

var (
	ErrInvalidWebhookURL   = errors.New("invalid webhook url")
	ErrInvalidHTTPMethod   = errors.New("invalid webhook method")
	ErrInvalidWebhookBody  = errors.New("invalid webhook body")
	ErrInvalidProjectID    = errors.New("invalid project id")
	ErrInvalidReportRange  = errors.New("invalid report range")
	ErrInvalidReportFormat = errors.New("invalid report format")
	ErrInvalidCleanupTime  = errors.New("invalid cleanup time")
	ErrInvalidBatchSize    = errors.New("invalid cleanup batch size")
	ErrInvalidMaxAttempts  = errors.New("invalid max attempts")
	ErrJobNotFound         = errors.New("job not found")
)
