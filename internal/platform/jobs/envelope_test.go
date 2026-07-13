package jobs

import (
	"errors"
	"testing"
	"time"
)

func TestNewEnvelope(t *testing.T) {
	now := time.Date(2026, time.July, 14, 10, 0, 0, 0, time.FixedZone("test", 3*60*60))

	envelope, err := NewEnvelope("job-1", TypeWebhookDeliver, now, 5, WebhookPayload{
		URL:    "https://example.com/hook",
		Method: "POST",
	})
	if err != nil {
		t.Fatalf("NewEnvelope() error = %v", err)
	}

	if envelope.SchemaVersion != SchemaVersion {
		t.Fatalf("SchemaVersion = %d, want %d", envelope.SchemaVersion, SchemaVersion)
	}
	if envelope.Attempt != 1 {
		t.Fatalf("Attempt = %d, want 1", envelope.Attempt)
	}
	if envelope.CreatedAt.Location() != time.UTC {
		t.Fatalf("CreatedAt location = %v, want UTC", envelope.CreatedAt.Location())
	}
}

func TestEnvelopeValidateRejectsUnknownType(t *testing.T) {
	envelope := Envelope{
		SchemaVersion: SchemaVersion,
		ID:            "job-1",
		Type:          Type("unknown"),
		CreatedAt:     time.Now().UTC(),
		Attempt:       1,
		MaxAttempts:   5,
		Payload:       []byte(`{}`),
	}

	if err := envelope.Validate(); !errors.Is(err, ErrInvalidType) {
		t.Fatalf("Validate() error = %v, want %v", err, ErrInvalidType)
	}
}
