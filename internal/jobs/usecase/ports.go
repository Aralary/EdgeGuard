package usecase

import (
	"context"
	"time"

	platformjobs "github.com/aralary/edgeguard/internal/platform/jobs"
)

type Publisher interface {
	Publish(ctx context.Context, envelope platformjobs.Envelope) error
}

type IDGenerator interface {
	NewID() (string, error)
}

type Clock interface {
	Now() time.Time
}
