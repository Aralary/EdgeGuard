package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/aralary/edgeguard/internal/jobs/domain"
	platformjobs "github.com/aralary/edgeguard/internal/platform/jobs"
)

type publisherStub struct {
	envelope platformjobs.Envelope
	err      error
}

func (p *publisherStub) Publish(_ context.Context, envelope platformjobs.Envelope) error {
	p.envelope = envelope
	return p.err
}

type jobRepositoryStub struct {
	created       domain.Job
	queuedJobID   string
	failedJobID   string
	failure       string
	job           domain.Job
	createErr     error
	markQueuedErr error
	markFailedErr error
	getErr        error
}

func (r *jobRepositoryStub) CreateJob(_ context.Context, job domain.Job) error {
	r.created = job
	return r.createErr
}

func (r *jobRepositoryStub) MarkQueued(_ context.Context, jobID string, _ time.Time) error {
	r.queuedJobID = jobID
	return r.markQueuedErr
}

func (r *jobRepositoryStub) MarkSubmissionFailed(_ context.Context, jobID string, failure string, _ time.Time) error {
	r.failedJobID = jobID
	r.failure = failure
	return r.markFailedErr
}

func (r *jobRepositoryStub) GetJob(_ context.Context, _ string) (domain.Job, error) {
	return r.job, r.getErr
}

type idGeneratorStub struct {
	id  string
	err error
}

func (g idGeneratorStub) NewID() (string, error) {
	return g.id, g.err
}

type clockStub struct {
	now time.Time
}

func (c clockStub) Now() time.Time {
	return c.now
}

func TestSubmitWebhook(t *testing.T) {
	publisher := &publisherStub{}
	repository := &jobRepositoryStub{}
	now := time.Date(2026, time.July, 14, 12, 0, 0, 0, time.UTC)
	uc := New(Dependencies{
		Publisher:   publisher,
		Repository:  repository,
		IDGenerator: idGeneratorStub{id: "job-1"},
		Clock:       clockStub{now: now},
	})

	receipt, err := uc.SubmitWebhook(context.Background(), SubmitWebhookInput{
		URL:    "https://example.com/webhook",
		Method: "post",
		Body:   json.RawMessage(`{"status":"ok"}`),
	})
	if err != nil {
		t.Fatalf("SubmitWebhook() error = %v", err)
	}

	if receipt.ID != "job-1" || receipt.Status != "queued" {
		t.Fatalf("receipt = %#v", receipt)
	}
	if publisher.envelope.Type != platformjobs.TypeWebhookDeliver {
		t.Fatalf("job type = %q, want %q", publisher.envelope.Type, platformjobs.TypeWebhookDeliver)
	}
	if publisher.envelope.MaxAttempts != 5 {
		t.Fatalf("MaxAttempts = %d, want 5", publisher.envelope.MaxAttempts)
	}
	if repository.created.Status != domain.StatusPublishing {
		t.Fatalf("created status = %q, want %q", repository.created.Status, domain.StatusPublishing)
	}
	if repository.queuedJobID != "job-1" {
		t.Fatalf("queued job id = %q, want job-1", repository.queuedJobID)
	}
}

func TestSubmitJobReturnsPublishErrorAndMarksSubmissionFailed(t *testing.T) {
	publishError := errors.New("RabbitMQ unavailable")
	repository := &jobRepositoryStub{}
	uc := New(Dependencies{
		Publisher:   &publisherStub{err: publishError},
		Repository:  repository,
		IDGenerator: idGeneratorStub{id: "job-1"},
		Clock:       clockStub{now: time.Now()},
	})

	_, err := uc.SubmitCleanup(context.Background(), SubmitCleanupInput{
		Before: time.Now().Add(-time.Hour),
	})
	if !errors.Is(err, publishError) {
		t.Fatalf("SubmitCleanup() error = %v, want wrapped %v", err, publishError)
	}
	if repository.failedJobID != "job-1" {
		t.Fatalf("failed job id = %q, want job-1", repository.failedJobID)
	}
	if repository.failure == "" {
		t.Fatal("failure must be recorded")
	}
}

func TestSubmitJobDoesNotPublishWhenPersistenceFails(t *testing.T) {
	createError := errors.New("postgres unavailable")
	publisher := &publisherStub{}
	uc := New(Dependencies{
		Publisher:   publisher,
		Repository:  &jobRepositoryStub{createErr: createError},
		IDGenerator: idGeneratorStub{id: "job-1"},
		Clock:       clockStub{now: time.Now()},
	})

	_, err := uc.SubmitCleanup(context.Background(), SubmitCleanupInput{
		Before: time.Now().Add(-time.Hour),
	})
	if !errors.Is(err, createError) {
		t.Fatalf("SubmitCleanup() error = %v, want wrapped %v", err, createError)
	}
	if publisher.envelope.ID != "" {
		t.Fatalf("publisher received envelope %#v", publisher.envelope)
	}
}

func TestSubmitJobIgnoresQueuedStatusUpdateFailureAfterConfirmation(t *testing.T) {
	repository := &jobRepositoryStub{markQueuedErr: errors.New("postgres unavailable")}
	uc := New(Dependencies{
		Publisher:   &publisherStub{},
		Repository:  repository,
		IDGenerator: idGeneratorStub{id: "job-1"},
		Clock:       clockStub{now: time.Now()},
	})

	receipt, err := uc.SubmitCleanup(context.Background(), SubmitCleanupInput{
		Before: time.Now().Add(-time.Hour),
	})
	if err != nil {
		t.Fatalf("SubmitCleanup() error = %v", err)
	}
	if receipt.ID != "job-1" {
		t.Fatalf("receipt id = %q, want job-1", receipt.ID)
	}
}
