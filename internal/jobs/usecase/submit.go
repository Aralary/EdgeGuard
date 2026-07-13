package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aralary/edgeguard/internal/jobs/domain"
	platformjobs "github.com/aralary/edgeguard/internal/platform/jobs"
)

type SubmitWebhookInput struct {
	URL         string
	Method      string
	Headers     map[string]string
	Body        json.RawMessage
	MaxAttempts int
}

type SubmitReportInput struct {
	ProjectID   string
	From        time.Time
	To          time.Time
	Format      string
	MaxAttempts int
}

type SubmitCleanupInput struct {
	Before      time.Time
	BatchSize   int
	MaxAttempts int
}

func (u *Usecase) SubmitWebhook(ctx context.Context, input SubmitWebhookInput) (domain.Receipt, error) {
	payload, err := domain.NewWebhookPayload(input.URL, input.Method, input.Headers, input.Body)
	if err != nil {
		return domain.Receipt{}, err
	}

	return u.submit(ctx, platformjobs.TypeWebhookDeliver, payload, input.MaxAttempts)
}

func (u *Usecase) SubmitReport(ctx context.Context, input SubmitReportInput) (domain.Receipt, error) {
	payload, err := domain.NewReportPayload(input.ProjectID, input.From, input.To, input.Format)
	if err != nil {
		return domain.Receipt{}, err
	}

	return u.submit(ctx, platformjobs.TypeReportGenerate, payload, input.MaxAttempts)
}

func (u *Usecase) SubmitCleanup(ctx context.Context, input SubmitCleanupInput) (domain.Receipt, error) {
	payload, err := domain.NewCleanupPayload(input.Before, input.BatchSize)
	if err != nil {
		return domain.Receipt{}, err
	}

	return u.submit(ctx, platformjobs.TypeCleanupExpiredTokens, payload, input.MaxAttempts)
}

func (u *Usecase) submit(ctx context.Context, jobType platformjobs.Type, payload any, maxAttempts int) (domain.Receipt, error) {
	maxAttempts, err := domain.NormalizeMaxAttempts(maxAttempts)
	if err != nil {
		return domain.Receipt{}, err
	}

	jobID, err := u.idGenerator.NewID()
	if err != nil {
		return domain.Receipt{}, fmt.Errorf("generate job id: %w", err)
	}

	createdAt := u.clock.Now().UTC()
	envelope, err := platformjobs.NewEnvelope(jobID, jobType, createdAt, maxAttempts, payload)
	if err != nil {
		return domain.Receipt{}, fmt.Errorf("build job envelope: %w", err)
	}

	job := domain.NewPublishingJob(envelope, createdAt)
	if err := u.repository.CreateJob(ctx, job); err != nil {
		return domain.Receipt{}, fmt.Errorf("persist job before publishing: %w", err)
	}

	if err := u.publisher.Publish(ctx, envelope); err != nil {
		failedAt := u.clock.Now().UTC()
		_ = u.repository.MarkSubmissionFailed(ctx, envelope.ID, err.Error(), failedAt)
		return domain.Receipt{}, fmt.Errorf("publish job: %w", err)
	}

	// RabbitMQ has already confirmed the message. A status update failure must not
	// turn the successful submission into a client-visible error because retrying
	// the HTTP request could enqueue a duplicate job. The worker can move a job
	// directly from publishing to processing if it consumes the message first.
	_ = u.repository.MarkQueued(ctx, envelope.ID, u.clock.Now().UTC())

	return domain.Receipt{
		ID:        envelope.ID,
		Type:      envelope.Type,
		Status:    "queued",
		CreatedAt: envelope.CreatedAt,
	}, nil
}
