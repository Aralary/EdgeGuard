package usecase

import (
	"context"
	"time"

	"github.com/aralary/edgeguard/internal/jobs/domain"
	platformjobs "github.com/aralary/edgeguard/internal/platform/jobs"
)

type Publisher interface {
	Publish(ctx context.Context, envelope platformjobs.Envelope) error
}

type JobRepository interface {
	CreateJob(ctx context.Context, job domain.Job) error
	MarkQueued(ctx context.Context, jobID string, at time.Time) error
	MarkSubmissionFailed(ctx context.Context, jobID string, failure string, at time.Time) error
	GetJob(ctx context.Context, jobID string) (domain.Job, error)
}

type IDGenerator interface {
	NewID() (string, error)
}

type Clock interface {
	Now() time.Time
}
