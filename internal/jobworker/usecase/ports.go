package usecase

import (
	"context"

	platformjobs "github.com/aralary/edgeguard/internal/platform/jobs"
)

type WebhookSender interface {
	Deliver(ctx context.Context, payload platformjobs.WebhookPayload) error
}

type ReportGenerator interface {
	Generate(ctx context.Context, jobID string, payload platformjobs.ReportPayload) (outputPath string, rowCount int64, err error)
}

type TokenCleaner interface {
	CleanupExpiredTokens(ctx context.Context, payload platformjobs.CleanupPayload) (int64, error)
}
