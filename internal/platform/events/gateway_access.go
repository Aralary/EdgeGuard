package events

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	GatewayAccessEventType          = "gateway.access"
	GatewayAccessEventSchemaVersion = 1

	ClientTypeAPIKey = "api_key"
	ClientTypeIP     = "ip"
)

type GatewayAccessEvent struct {
	SchemaVersion    int       `json:"schema_version"`
	EventID          string    `json:"event_id"`
	OccurredAt       time.Time `json:"occurred_at"`
	RequestID        string    `json:"request_id"`
	ProjectID        string    `json:"project_id,omitempty"`
	RouteName        string    `json:"route_name,omitempty"`
	RoutePathPrefix  string    `json:"route_path_prefix,omitempty"`
	Method           string    `json:"method"`
	RequestPath      string    `json:"request_path"`
	StatusCode       int       `json:"status_code"`
	DurationMS       int64     `json:"duration_ms"`
	ResponseBytes    int64     `json:"response_bytes"`
	ClientType       string    `json:"client_type"`
	ClientID         string    `json:"client_id"`
	AuthRequired     bool      `json:"auth_required"`
	RateLimitEnabled bool      `json:"rate_limit_enabled"`
	RateLimited      bool      `json:"rate_limited"`
}

func (e GatewayAccessEvent) Validate() error {
	if e.SchemaVersion != GatewayAccessEventSchemaVersion {
		return fmt.Errorf("unsupported gateway access event schema version: %d", e.SchemaVersion)
	}
	if strings.TrimSpace(e.EventID) == "" {
		return errors.New("gateway access event id is required")
	}
	if e.OccurredAt.IsZero() {
		return errors.New("gateway access event occurred_at is required")
	}
	if strings.TrimSpace(e.RequestID) == "" {
		return errors.New("gateway access event request_id is required")
	}
	if strings.TrimSpace(e.Method) == "" {
		return errors.New("gateway access event method is required")
	}
	if strings.TrimSpace(e.RequestPath) == "" {
		return errors.New("gateway access event request_path is required")
	}
	if e.StatusCode < 100 || e.StatusCode > 599 {
		return fmt.Errorf("invalid gateway access event status code: %d", e.StatusCode)
	}
	if e.DurationMS < 0 {
		return errors.New("gateway access event duration must not be negative")
	}
	if e.ResponseBytes < 0 {
		return errors.New("gateway access event response bytes must not be negative")
	}
	if e.ClientType != ClientTypeAPIKey && e.ClientType != ClientTypeIP {
		return fmt.Errorf("unsupported gateway access event client type: %q", e.ClientType)
	}
	if strings.TrimSpace(e.ClientID) == "" {
		return errors.New("gateway access event client_id is required")
	}

	return nil
}
