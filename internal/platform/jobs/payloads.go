package jobs

import (
	"encoding/json"
	"time"
)

type WebhookPayload struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    json.RawMessage   `json:"body,omitempty"`
}

type ReportPayload struct {
	ProjectID string    `json:"project_id"`
	From      time.Time `json:"from"`
	To        time.Time `json:"to"`
	Format    string    `json:"format"`
}

type CleanupPayload struct {
	Before    time.Time `json:"before"`
	BatchSize int       `json:"batch_size"`
}
