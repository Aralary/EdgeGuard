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
