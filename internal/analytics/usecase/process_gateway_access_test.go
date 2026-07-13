package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aralary/edgeguard/internal/analytics/domain"
	"github.com/aralary/edgeguard/internal/platform/events"
)

type gatewayAccessRepositoryStub struct {
	result domain.IngestionResult
	err    error
	calls  int
	event  events.GatewayAccessEvent
}

func (s *gatewayAccessRepositoryStub) StoreGatewayAccess(
	_ context.Context,
	event events.GatewayAccessEvent,
) (domain.IngestionResult, error) {
	s.calls++
	s.event = event
	return s.result, s.err
}

func TestProcessGatewayAccess(t *testing.T) {
	repository := &gatewayAccessRepositoryStub{
		result: domain.IngestionResult{Aggregated: true},
	}
	uc := New(Dependencies{GatewayAccessRepository: repository})
	event := validGatewayAccessEvent()

	result, err := uc.ProcessGatewayAccess(context.Background(), event)
	if err != nil {
		t.Fatalf("ProcessGatewayAccess() error = %v", err)
	}
	if !result.Aggregated || result.Duplicate {
		t.Fatalf("unexpected result: %#v", result)
	}
	if repository.calls != 1 || repository.event.EventID != event.EventID {
		t.Fatalf("repository call mismatch: calls=%d event=%#v", repository.calls, repository.event)
	}
}

func TestProcessGatewayAccessRejectsInvalidEvent(t *testing.T) {
	repository := &gatewayAccessRepositoryStub{}
	uc := New(Dependencies{GatewayAccessRepository: repository})

	_, err := uc.ProcessGatewayAccess(context.Background(), events.GatewayAccessEvent{})
	if !errors.Is(err, ErrInvalidGatewayAccessEvent) {
		t.Fatalf("ProcessGatewayAccess() error = %v, want ErrInvalidGatewayAccessEvent", err)
	}
	if repository.calls != 0 {
		t.Fatalf("repository calls = %d, want 0", repository.calls)
	}
}

func TestProcessGatewayAccessWrapsRepositoryError(t *testing.T) {
	repositoryErr := errors.New("postgres unavailable")
	repository := &gatewayAccessRepositoryStub{err: repositoryErr}
	uc := New(Dependencies{GatewayAccessRepository: repository})

	_, err := uc.ProcessGatewayAccess(context.Background(), validGatewayAccessEvent())
	if !errors.Is(err, repositoryErr) {
		t.Fatalf("ProcessGatewayAccess() error = %v, want wrapped repository error", err)
	}
}

func validGatewayAccessEvent() events.GatewayAccessEvent {
	return events.GatewayAccessEvent{
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
		DurationMS:       15,
		ResponseBytes:    128,
		ClientType:       events.ClientTypeIP,
		ClientID:         "client-hash",
		AuthRequired:     false,
		RateLimitEnabled: true,
		RateLimited:      false,
	}
}
