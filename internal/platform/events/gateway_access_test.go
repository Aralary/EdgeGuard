package events

import (
	"testing"
	"time"
)

func TestGatewayAccessEventValidate(t *testing.T) {
	event := GatewayAccessEvent{
		SchemaVersion: GatewayAccessEventSchemaVersion,
		EventID:       "event-1",
		OccurredAt:    time.Now().UTC(),
		RequestID:     "request-1",
		Method:        "GET",
		RequestPath:   "/api/orders",
		StatusCode:    200,
		DurationMS:    15,
		ResponseBytes: 128,
		ClientType:    ClientTypeIP,
		ClientID:      "hashed-ip",
	}

	if err := event.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestGatewayAccessEventValidateRejectsUnsupportedSchema(t *testing.T) {
	event := GatewayAccessEvent{SchemaVersion: 99}

	if err := event.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want error")
	}
}

func TestGatewayAccessEventValidateRejectsInvalidProjectID(t *testing.T) {
	event := GatewayAccessEvent{
		SchemaVersion: GatewayAccessEventSchemaVersion,
		EventID:       "event-1",
		OccurredAt:    time.Now().UTC(),
		RequestID:     "request-1",
		ProjectID:     "not-a-uuid",
		Method:        "GET",
		RequestPath:   "/api/orders",
		StatusCode:    200,
		ClientType:    ClientTypeIP,
		ClientID:      "hashed-ip",
	}

	if err := event.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want error")
	}
}
