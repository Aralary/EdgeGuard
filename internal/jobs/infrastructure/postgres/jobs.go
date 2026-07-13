package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	jobsdomain "github.com/aralary/edgeguard/internal/jobs/domain"
	platformjobs "github.com/aralary/edgeguard/internal/platform/jobs"
	"github.com/jackc/pgx/v5"
)

const createJobQuery = `
INSERT INTO background_jobs (
    id,
    type,
    status,
    current_attempt,
    max_attempts,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id;
`

func (r *Repository) CreateJob(ctx context.Context, job jobsdomain.Job) error {
	var id string
	if err := r.db.QueryRow(
		ctx,
		createJobQuery,
		job.ID,
		string(job.Type),
		string(job.Status),
		job.CurrentAttempt,
		job.MaxAttempts,
		job.CreatedAt,
		job.UpdatedAt,
	).Scan(&id); err != nil {
		return fmt.Errorf("insert background job: %w", err)
	}

	return nil
}

const markQueuedQuery = `
UPDATE background_jobs
SET status = 'queued',
    queued_at = COALESCE(queued_at, $2),
    updated_at = $2,
    last_error = ''
WHERE id = $1
  AND status = 'publishing'
RETURNING id;
`

func (r *Repository) MarkQueued(ctx context.Context, jobID string, at time.Time) error {
	return scanOptionalID(r.db.QueryRow(ctx, markQueuedQuery, jobID, at.UTC()), "mark background job queued")
}

const markSubmissionFailedQuery = `
UPDATE background_jobs
SET status = 'publish_failed',
    completed_at = $2,
    updated_at = $2,
    last_error = $3
WHERE id = $1
  AND status = 'publishing'
RETURNING id;
`

func (r *Repository) MarkSubmissionFailed(ctx context.Context, jobID string, failure string, at time.Time) error {
	return scanOptionalID(
		r.db.QueryRow(ctx, markSubmissionFailedQuery, jobID, at.UTC(), jobsdomain.NormalizeFailureMessage(failure)),
		"mark background job submission failed",
	)
}

const getJobQuery = `
SELECT
    id,
    type,
    status,
    current_attempt,
    max_attempts,
    created_at,
    queued_at,
    started_at,
    completed_at,
    updated_at,
    last_error,
    result_message,
    output_path,
    affected_rows
FROM background_jobs
WHERE id = $1;
`

func (r *Repository) GetJob(ctx context.Context, jobID string) (jobsdomain.Job, error) {
	var job jobsdomain.Job
	var jobType string
	var status string

	err := r.db.QueryRow(ctx, getJobQuery, jobID).Scan(
		&job.ID,
		&jobType,
		&status,
		&job.CurrentAttempt,
		&job.MaxAttempts,
		&job.CreatedAt,
		&job.QueuedAt,
		&job.StartedAt,
		&job.CompletedAt,
		&job.UpdatedAt,
		&job.LastError,
		&job.ResultMessage,
		&job.OutputPath,
		&job.AffectedRows,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return jobsdomain.Job{}, jobsdomain.ErrJobNotFound
	}
	if err != nil {
		return jobsdomain.Job{}, fmt.Errorf("select background job: %w", err)
	}

	job.Type = platformjobs.Type(jobType)
	job.Status = jobsdomain.Status(status)
	return job, nil
}

const prepareAttemptQuery = `
WITH prepared AS (
    INSERT INTO background_jobs (
        id,
        type,
        status,
        current_attempt,
        max_attempts,
        created_at,
        started_at,
        updated_at
    )
    VALUES ($1, $2, 'processing', LEAST($6::integer, $3::integer), $3::integer, $4, $5, $5)
    ON CONFLICT (id) DO UPDATE
    SET status = 'processing',
        current_attempt = LEAST($6::integer, background_jobs.max_attempts),
        started_at = COALESCE(background_jobs.started_at, $5),
        updated_at = $5,
        last_error = ''
    WHERE background_jobs.status NOT IN ('succeeded', 'failed')
    RETURNING status
)
SELECT status FROM prepared
UNION ALL
SELECT status
FROM background_jobs
WHERE id = $1
  AND NOT EXISTS (SELECT 1 FROM prepared)
LIMIT 1;
`

func (r *Repository) PrepareAttempt(
	ctx context.Context,
	envelope platformjobs.Envelope,
	attempt int,
	at time.Time,
) (jobsdomain.Status, error) {
	var status string
	if err := r.db.QueryRow(
		ctx,
		prepareAttemptQuery,
		envelope.ID,
		string(envelope.Type),
		envelope.MaxAttempts,
		envelope.CreatedAt.UTC(),
		at.UTC(),
		attempt,
	).Scan(&status); err != nil {
		return "", fmt.Errorf("prepare background job attempt: %w", err)
	}

	return jobsdomain.Status(status), nil
}

const markRetryingQuery = `
UPDATE background_jobs
SET status = 'retrying',
    current_attempt = LEAST($2::integer, max_attempts),
    updated_at = $3,
    last_error = $4
WHERE id = $1
  AND status NOT IN ('succeeded', 'failed')
RETURNING id;
`

func (r *Repository) MarkRetrying(ctx context.Context, jobID string, attempt int, failure string, at time.Time) error {
	return scanRequiredID(
		r.db.QueryRow(
			ctx,
			markRetryingQuery,
			jobID,
			attempt,
			at.UTC(),
			jobsdomain.NormalizeFailureMessage(failure),
		),
		"mark background job retrying",
	)
}

const markSucceededQuery = `
UPDATE background_jobs
SET status = 'succeeded',
    current_attempt = LEAST($2::integer, max_attempts),
    completed_at = $3,
    updated_at = $3,
    last_error = '',
    result_message = $4,
    output_path = $5,
    affected_rows = $6
WHERE id = $1
  AND status NOT IN ('succeeded', 'failed')
RETURNING id;
`

func (r *Repository) MarkSucceeded(
	ctx context.Context,
	jobID string,
	attempt int,
	result jobsdomain.ExecutionResult,
	at time.Time,
) error {
	return scanRequiredID(
		r.db.QueryRow(
			ctx,
			markSucceededQuery,
			jobID,
			attempt,
			at.UTC(),
			result.Message,
			result.OutputPath,
			result.AffectedRows,
		),
		"mark background job succeeded",
	)
}

const markFailedQuery = `
UPDATE background_jobs
SET status = 'failed',
    current_attempt = LEAST($2::integer, max_attempts),
    completed_at = $3,
    updated_at = $3,
    last_error = $4
WHERE id = $1
  AND status <> 'succeeded'
RETURNING id;
`

func (r *Repository) MarkFailed(ctx context.Context, jobID string, attempt int, failure string, at time.Time) error {
	return scanRequiredID(
		r.db.QueryRow(
			ctx,
			markFailedQuery,
			jobID,
			attempt,
			at.UTC(),
			jobsdomain.NormalizeFailureMessage(failure),
		),
		"mark background job failed",
	)
}

func scanOptionalID(row pgx.Row, operation string) error {
	var id string
	if err := row.Scan(&id); errors.Is(err, pgx.ErrNoRows) {
		return nil
	} else if err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}

	return nil
}

func scanRequiredID(row pgx.Row, operation string) error {
	var id string
	if err := row.Scan(&id); err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}

	return nil
}
