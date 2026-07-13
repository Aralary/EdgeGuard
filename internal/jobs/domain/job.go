package domain

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	platformjobs "github.com/aralary/edgeguard/internal/platform/jobs"
)

const (
	DefaultMaxAttempts = 5
	MaxMaxAttempts     = 10
	DefaultBatchSize   = 500
	MaxBatchSize       = 5000
)

type Receipt struct {
	ID        string
	Type      platformjobs.Type
	Status    string
	CreatedAt time.Time
}

func NewWebhookPayload(rawURL, method string, headers map[string]string, body json.RawMessage) (platformjobs.WebhookPayload, error) {
	parsedURL, err := url.ParseRequestURI(strings.TrimSpace(rawURL))
	if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return platformjobs.WebhookPayload{}, ErrInvalidWebhookURL
	}

	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = http.MethodPost
	}
	if method != http.MethodPost && method != http.MethodPut && method != http.MethodPatch {
		return platformjobs.WebhookPayload{}, ErrInvalidHTTPMethod
	}

	if len(body) != 0 && !json.Valid(body) {
		return platformjobs.WebhookPayload{}, ErrInvalidWebhookBody
	}

	return platformjobs.WebhookPayload{
		URL:     parsedURL.String(),
		Method:  method,
		Headers: cloneHeaders(headers),
		Body:    body,
	}, nil
}

func NewReportPayload(projectID string, from, to time.Time, format string) (platformjobs.ReportPayload, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return platformjobs.ReportPayload{}, ErrInvalidProjectID
	}
	if from.IsZero() || to.IsZero() || !from.Before(to) {
		return platformjobs.ReportPayload{}, ErrInvalidReportRange
	}

	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		format = "json"
	}
	if format != "json" && format != "csv" {
		return platformjobs.ReportPayload{}, ErrInvalidReportFormat
	}

	return platformjobs.ReportPayload{
		ProjectID: projectID,
		From:      from.UTC(),
		To:        to.UTC(),
		Format:    format,
	}, nil
}

func NewCleanupPayload(before time.Time, batchSize int) (platformjobs.CleanupPayload, error) {
	if before.IsZero() {
		return platformjobs.CleanupPayload{}, ErrInvalidCleanupTime
	}
	if batchSize == 0 {
		batchSize = DefaultBatchSize
	}
	if batchSize < 1 || batchSize > MaxBatchSize {
		return platformjobs.CleanupPayload{}, ErrInvalidBatchSize
	}

	return platformjobs.CleanupPayload{
		Before:    before.UTC(),
		BatchSize: batchSize,
	}, nil
}

func NormalizeMaxAttempts(value int) (int, error) {
	if value == 0 {
		return DefaultMaxAttempts, nil
	}
	if value < 1 || value > MaxMaxAttempts {
		return 0, ErrInvalidMaxAttempts
	}
	return value, nil
}

func cloneHeaders(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}

	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}

	return result
}
