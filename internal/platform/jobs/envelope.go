package jobs

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const SchemaVersion = 1

type Type string

const (
	TypeWebhookDeliver       Type = "jobs.webhook.deliver"
	TypeReportGenerate       Type = "jobs.report.generate"
	TypeCleanupExpiredTokens Type = "jobs.cleanup.expired_tokens"
)

var (
	ErrInvalidID          = errors.New("invalid job id")
	ErrInvalidType        = errors.New("invalid job type")
	ErrInvalidCreatedAt   = errors.New("invalid job created_at")
	ErrInvalidAttempt     = errors.New("invalid job attempt")
	ErrInvalidMaxAttempts = errors.New("invalid job max_attempts")
	ErrInvalidPayload     = errors.New("invalid job payload")
)

type Envelope struct {
	SchemaVersion int             `json:"schema_version"`
	ID            string          `json:"id"`
	Type          Type            `json:"type"`
	CreatedAt     time.Time       `json:"created_at"`
	Attempt       int             `json:"attempt"`
	MaxAttempts   int             `json:"max_attempts"`
	Payload       json.RawMessage `json:"payload"`
}

func NewEnvelope(id string, jobType Type, createdAt time.Time, maxAttempts int, payload any) (Envelope, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return Envelope{}, ErrInvalidPayload
	}

	envelope := Envelope{
		SchemaVersion: SchemaVersion,
		ID:            strings.TrimSpace(id),
		Type:          jobType,
		CreatedAt:     createdAt.UTC(),
		Attempt:       1,
		MaxAttempts:   maxAttempts,
		Payload:       payloadJSON,
	}

	if err := envelope.Validate(); err != nil {
		return Envelope{}, err
	}

	return envelope, nil
}

func (e Envelope) Validate() error {
	if e.SchemaVersion != SchemaVersion {
		return ErrInvalidPayload
	}
	if strings.TrimSpace(e.ID) == "" {
		return ErrInvalidID
	}
	if !e.Type.Valid() {
		return ErrInvalidType
	}
	if e.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if e.Attempt < 1 {
		return ErrInvalidAttempt
	}
	if e.MaxAttempts < e.Attempt {
		return ErrInvalidMaxAttempts
	}
	if len(e.Payload) == 0 || !json.Valid(e.Payload) {
		return ErrInvalidPayload
	}

	return nil
}

func (t Type) Valid() bool {
	switch t {
	case TypeWebhookDeliver, TypeReportGenerate, TypeCleanupExpiredTokens:
		return true
	default:
		return false
	}
}
