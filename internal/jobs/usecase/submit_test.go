package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

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
	now := time.Date(2026, time.July, 14, 12, 0, 0, 0, time.UTC)
	uc := New(Dependencies{
		Publisher:   publisher,
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
}

func TestSubmitJobReturnsPublishError(t *testing.T) {
	publishError := errors.New("RabbitMQ unavailable")
	uc := New(Dependencies{
		Publisher:   &publisherStub{err: publishError},
		IDGenerator: idGeneratorStub{id: "job-1"},
		Clock:       clockStub{now: time.Now()},
	})

	_, err := uc.SubmitCleanup(context.Background(), SubmitCleanupInput{
		Before: time.Now().Add(-time.Hour),
	})
	if !errors.Is(err, publishError) {
		t.Fatalf("SubmitCleanup() error = %v, want wrapped %v", err, publishError)
	}
}
