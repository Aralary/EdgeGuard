package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/aralary/edgeguard/internal/jobworker/domain"
	platformjobs "github.com/aralary/edgeguard/internal/platform/jobs"
)

type webhookStub struct {
	called bool
	err    error
}

func (stub *webhookStub) Deliver(_ context.Context, _ platformjobs.WebhookPayload) error {
	stub.called = true
	return stub.err
}

type reportStub struct{}

func (reportStub) Generate(_ context.Context, _ string, _ platformjobs.ReportPayload) (string, int64, error) {
	return "/tmp/report.json", 3, nil
}

type cleanerStub struct{}

func (cleanerStub) CleanupExpiredTokens(_ context.Context, _ platformjobs.CleanupPayload) (int64, error) {
	return 2, nil
}

func TestProcessWebhook(t *testing.T) {
	webhook := &webhookStub{}
	uc := New(Dependencies{WebhookSender: webhook})
	envelope := platformjobs.Envelope{
		SchemaVersion: platformjobs.SchemaVersion,
		ID:            "job-1",
		Type:          platformjobs.TypeWebhookDeliver,
		CreatedAt:     time.Now(),
		Attempt:       1,
		MaxAttempts:   5,
		Payload:       json.RawMessage(`{"url":"https://example.com","method":"POST"}`),
	}

	result, err := uc.Process(context.Background(), envelope)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if !webhook.called || result.Message != "webhook delivered" {
		t.Fatalf("result = %#v, called = %v", result, webhook.called)
	}
}

func TestProcessInvalidPayloadIsPermanent(t *testing.T) {
	uc := New(Dependencies{})
	_, err := uc.Process(context.Background(), platformjobs.Envelope{
		Type:    platformjobs.TypeCleanupExpiredTokens,
		Payload: json.RawMessage(`{`),
	})
	if !errors.Is(err, ErrInvalidJobPayload) || !domain.IsPermanent(err) {
		t.Fatalf("Process() error = %v, want permanent invalid payload", err)
	}
}
