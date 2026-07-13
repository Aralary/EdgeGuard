package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aralary/edgeguard/internal/platform/events"
)

type accessEventPublisherStub struct {
	event events.GatewayAccessEvent
	err   error
}

func (s *accessEventPublisherStub) PublishGatewayAccess(
	_ context.Context,
	event events.GatewayAccessEvent,
) error {
	s.event = event
	return s.err
}

func TestPublishAccessEventWithoutPublisherIsNoop(t *testing.T) {
	uc := New(Dependencies{})

	if err := uc.PublishAccessEvent(context.Background(), events.GatewayAccessEvent{}); err != nil {
		t.Fatalf("PublishAccessEvent() error = %v", err)
	}
}

func TestPublishAccessEventDelegatesToPublisher(t *testing.T) {
	publisher := &accessEventPublisherStub{}
	uc := New(Dependencies{AccessEventPublisher: publisher})
	event := events.GatewayAccessEvent{
		SchemaVersion: events.GatewayAccessEventSchemaVersion,
		EventID:       "event-1",
		OccurredAt:    time.Now().UTC(),
	}

	if err := uc.PublishAccessEvent(context.Background(), event); err != nil {
		t.Fatalf("PublishAccessEvent() error = %v", err)
	}
	if publisher.event.EventID != event.EventID {
		t.Fatalf("published event id = %q", publisher.event.EventID)
	}
}

func TestPublishAccessEventReturnsPublisherError(t *testing.T) {
	publisherErr := errors.New("kafka unavailable")
	uc := New(Dependencies{
		AccessEventPublisher: &accessEventPublisherStub{err: publisherErr},
	})

	err := uc.PublishAccessEvent(context.Background(), events.GatewayAccessEvent{})
	if !errors.Is(err, publisherErr) {
		t.Fatalf("PublishAccessEvent() error = %v, want %v", err, publisherErr)
	}
}
