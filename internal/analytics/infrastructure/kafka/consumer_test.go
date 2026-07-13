package kafka

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/aralary/edgeguard/internal/platform/events"
)

func TestDecodeGatewayAccessEvent(t *testing.T) {
	want := events.GatewayAccessEvent{
		SchemaVersion:    events.GatewayAccessEventSchemaVersion,
		EventID:          "event-1",
		OccurredAt:       time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC),
		RequestID:        "request-1",
		ProjectID:        "11111111-1111-1111-1111-111111111111",
		RouteName:        "orders",
		RoutePathPrefix:  "/api",
		Method:           "GET",
		RequestPath:      "/api/orders",
		StatusCode:       200,
		DurationMS:       17,
		ResponseBytes:    128,
		ClientType:       events.ClientTypeIP,
		ClientID:         "client-hash",
		AuthRequired:     false,
		RateLimitEnabled: true,
		RateLimited:      false,
	}

	payload, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	got, err := decodeGatewayAccessEvent(payload)
	if err != nil {
		t.Fatalf("decodeGatewayAccessEvent() error = %v", err)
	}
	if got.EventID != want.EventID || got.StatusCode != want.StatusCode || got.ProjectID != want.ProjectID {
		t.Fatalf("unexpected event: %#v", got)
	}
}

func TestDecodeGatewayAccessEventRejectsInvalidPayload(t *testing.T) {
	if _, err := decodeGatewayAccessEvent([]byte(`{"schema_version":1}`)); err == nil {
		t.Fatal("decodeGatewayAccessEvent() error = nil, want error")
	}
}

func TestNextRetryDelay(t *testing.T) {
	maximum := 30 * time.Second

	if got := nextRetryDelay(time.Second, maximum); got != 2*time.Second {
		t.Fatalf("nextRetryDelay(1s) = %s, want 2s", got)
	}
	if got := nextRetryDelay(20*time.Second, maximum); got != maximum {
		t.Fatalf("nextRetryDelay(20s) = %s, want %s", got, maximum)
	}
	if got := nextRetryDelay(maximum, maximum); got != maximum {
		t.Fatalf("nextRetryDelay(max) = %s, want %s", got, maximum)
	}
}

func TestNormalized(t *testing.T) {
	got := normalized([]string{" kafka-1:9092 ", "", "kafka-1:9092", "kafka-2:9092"})
	if len(got) != 2 || got[0] != "kafka-1:9092" || got[1] != "kafka-2:9092" {
		t.Fatalf("normalized() = %#v", got)
	}
}
