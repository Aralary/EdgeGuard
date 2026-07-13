package httpdelivery

import (
	"context"

	"github.com/aralary/edgeguard/internal/jobs/domain"
	"github.com/aralary/edgeguard/internal/jobs/usecase"
)

type JobsUsecase interface {
	SubmitWebhook(ctx context.Context, input usecase.SubmitWebhookInput) (domain.Receipt, error)
	SubmitReport(ctx context.Context, input usecase.SubmitReportInput) (domain.Receipt, error)
	SubmitCleanup(ctx context.Context, input usecase.SubmitCleanupInput) (domain.Receipt, error)
}
