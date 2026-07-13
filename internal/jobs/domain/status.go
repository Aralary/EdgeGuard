package domain

import (
	"strings"
	"time"

	platformjobs "github.com/aralary/edgeguard/internal/platform/jobs"
)

type Status string

const (
	StatusPublishing    Status = "publishing"
	StatusPublishFailed Status = "publish_failed"
	StatusQueued        Status = "queued"
	StatusProcessing    Status = "processing"
	StatusRetrying      Status = "retrying"
	StatusSucceeded     Status = "succeeded"
	StatusFailed        Status = "failed"
)

type Job struct {
	ID             string
	Type           platformjobs.Type
	Status         Status
	CurrentAttempt int
	MaxAttempts    int
	CreatedAt      time.Time
	QueuedAt       *time.Time
	StartedAt      *time.Time
	CompletedAt    *time.Time
	UpdatedAt      time.Time
	LastError      string
	ResultMessage  string
	OutputPath     string
	AffectedRows   int64
}

type ExecutionResult struct {
	Message      string
	OutputPath   string
	AffectedRows int64
}

func NewPublishingJob(envelope platformjobs.Envelope, now time.Time) Job {
	return Job{
		ID:          envelope.ID,
		Type:        envelope.Type,
		Status:      StatusPublishing,
		MaxAttempts: envelope.MaxAttempts,
		CreatedAt:   envelope.CreatedAt.UTC(),
		UpdatedAt:   now.UTC(),
	}
}

func (s Status) Final() bool {
	return s == StatusSucceeded || s == StatusFailed
}

func NormalizeFailureMessage(value string) string {
	return strings.TrimSpace(value)
}
