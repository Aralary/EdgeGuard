package kafka

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/aralary/edgeguard/internal/platform/events"
)

type testLogger struct{}

func (testLogger) Debug(args ...interface{})                 {}
func (testLogger) Debugf(format string, args ...interface{}) {}
func (testLogger) Info(args ...interface{})                  {}
func (testLogger) Infof(format string, args ...interface{})  {}
func (testLogger) Warn(args ...interface{})                  {}
func (testLogger) Warnf(format string, args ...interface{})  {}
func (testLogger) Error(args ...interface{})                 {}
func (testLogger) Errorf(format string, args ...interface{}) {}

func TestNewRejectsEmptyBrokers(t *testing.T) {
	_, err := New(Config{Topic: "events", ClientID: "gateway"}, testLogger{})
	if err == nil {
		t.Fatal("New() error = nil, want error")
	}
}

func TestNewAcceptsIdempotentProducerConfiguration(t *testing.T) {
	producer, err := New(Config{
		Brokers:  []string{"kafka:9092"},
		Topic:    "gateway.access.v1",
		ClientID: "gateway",
	}, testLogger{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	producer.client.Close()
}

func TestAccessEventRecord(t *testing.T) {
	occurredAt := time.Date(2026, time.July, 13, 12, 0, 0, 0, time.UTC)
	event := events.GatewayAccessEvent{
		SchemaVersion: events.GatewayAccessEventSchemaVersion,
		EventID:       "event-1",
		OccurredAt:    occurredAt,
		RequestID:     "request-1",
		ProjectID:     "project-1",
		RouteName:     "orders",
		Method:        "GET",
		RequestPath:   "/api/orders",
		StatusCode:    200,
		ClientType:    events.ClientTypeAPIKey,
		ClientID:      "key-1",
	}

	record, err := accessEventRecord("gateway.access.v1", event)
	if err != nil {
		t.Fatalf("accessEventRecord() error = %v", err)
	}

	if record.Topic != "gateway.access.v1" {
		t.Fatalf("Topic = %q", record.Topic)
	}
	if string(record.Key) != "project-1:orders" {
		t.Fatalf("Key = %q", record.Key)
	}
	if !record.Timestamp.Equal(occurredAt) {
		t.Fatalf("Timestamp = %s", record.Timestamp)
	}

	var decoded events.GatewayAccessEvent
	if err := json.Unmarshal(record.Value, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded.EventID != event.EventID {
		t.Fatalf("EventID = %q", decoded.EventID)
	}
}

func TestNormalizedBrokers(t *testing.T) {
	brokers := normalizedBrokers([]string{" kafka:9092 ", "", "kafka:9092", "kafka-2:9092"})

	if len(brokers) != 2 {
		t.Fatalf("len(brokers) = %d, want 2", len(brokers))
	}
	if brokers[0] != "kafka:9092" || brokers[1] != "kafka-2:9092" {
		t.Fatalf("brokers = %#v", brokers)
	}
}
