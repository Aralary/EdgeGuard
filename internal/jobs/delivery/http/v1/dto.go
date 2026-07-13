package httpdelivery

import (
	"encoding/json"
	"time"

	"github.com/aralary/edgeguard/internal/jobs/domain"
)

type submitWebhookRequest struct {
	URL         string            `json:"url"`
	Method      string            `json:"method"`
	Headers     map[string]string `json:"headers"`
	Body        json.RawMessage   `json:"body"`
	MaxAttempts int               `json:"max_attempts"`
}

type submitReportRequest struct {
	ProjectID   string    `json:"project_id"`
	From        time.Time `json:"from"`
	To          time.Time `json:"to"`
	Format      string    `json:"format"`
	MaxAttempts int       `json:"max_attempts"`
}

type submitCleanupRequest struct {
	Before      time.Time `json:"before"`
	BatchSize   int       `json:"batch_size"`
	MaxAttempts int       `json:"max_attempts"`
}

type jobReceiptResponse struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type jobStatusResponse struct {
	ID             string     `json:"id"`
	Type           string     `json:"type"`
	Status         string     `json:"status"`
	CurrentAttempt int        `json:"current_attempt"`
	MaxAttempts    int        `json:"max_attempts"`
	CreatedAt      time.Time  `json:"created_at"`
	QueuedAt       *time.Time `json:"queued_at,omitempty"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at"`
	LastError      string     `json:"last_error,omitempty"`
	ResultMessage  string     `json:"result_message,omitempty"`
	OutputPath     string     `json:"output_path,omitempty"`
	AffectedRows   int64      `json:"affected_rows"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func newJobReceiptResponse(receipt domain.Receipt) jobReceiptResponse {
	return jobReceiptResponse{
		ID:        receipt.ID,
		Type:      string(receipt.Type),
		Status:    receipt.Status,
		CreatedAt: receipt.CreatedAt,
	}
}

func newJobStatusResponse(job domain.Job) jobStatusResponse {
	return jobStatusResponse{
		ID:             job.ID,
		Type:           string(job.Type),
		Status:         string(job.Status),
		CurrentAttempt: job.CurrentAttempt,
		MaxAttempts:    job.MaxAttempts,
		CreatedAt:      job.CreatedAt,
		QueuedAt:       job.QueuedAt,
		StartedAt:      job.StartedAt,
		CompletedAt:    job.CompletedAt,
		UpdatedAt:      job.UpdatedAt,
		LastError:      job.LastError,
		ResultMessage:  job.ResultMessage,
		OutputPath:     job.OutputPath,
		AffectedRows:   job.AffectedRows,
	}
}
