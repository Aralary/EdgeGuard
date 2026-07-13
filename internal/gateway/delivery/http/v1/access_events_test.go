package httpdelivery

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aralary/edgeguard/internal/gateway/domain"
	"github.com/aralary/edgeguard/internal/platform/events"
	"github.com/labstack/echo/v5"
)

type accessEventUsecaseStub struct {
	event      events.GatewayAccessEvent
	publishErr error
}

func (s *accessEventUsecaseStub) PublishAccessEvent(
	_ context.Context,
	event events.GatewayAccessEvent,
) error {
	s.event = event
	return s.publishErr
}

func TestAccessEventsPublishesCompletedRequest(t *testing.T) {
	publisher := &accessEventUsecaseStub{}
	e := echo.New()
	e.Use(RequestID())
	e.Use(AccessEvents(publisher, testLogger{}))
	e.GET("/api/orders", func(c *echo.Context) error {
		setAccessEventRoute(c, domain.Route{
			ProjectID:    "project-1",
			Name:         "orders",
			PathPrefix:   "/api",
			AuthRequired: true,
			RateLimit: domain.RateLimitPolicy{
				Enabled: true,
				Limit:   10,
				Window:  time.Minute,
			},
		})
		setAccessEventAPIKey(c, "key-1")
		return c.String(http.StatusCreated, "created")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/orders", nil)
	req.RemoteAddr = "192.0.2.10:12345"
	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", recorder.Code)
	}
	if publisher.event.SchemaVersion != events.GatewayAccessEventSchemaVersion {
		t.Fatalf("SchemaVersion = %d", publisher.event.SchemaVersion)
	}
	if publisher.event.RequestID == "" || publisher.event.EventID == "" {
		t.Fatal("request/event id is empty")
	}
	if publisher.event.ProjectID != "project-1" || publisher.event.RouteName != "orders" {
		t.Fatalf("route metadata = %#v", publisher.event)
	}
	if publisher.event.ClientType != events.ClientTypeAPIKey || publisher.event.ClientID != "key-1" {
		t.Fatalf("client = %s/%s", publisher.event.ClientType, publisher.event.ClientID)
	}
	if publisher.event.StatusCode != http.StatusCreated {
		t.Fatalf("StatusCode = %d", publisher.event.StatusCode)
	}
	if publisher.event.ResponseBytes != int64(len("created")) {
		t.Fatalf("ResponseBytes = %d", publisher.event.ResponseBytes)
	}
	if !publisher.event.AuthRequired || !publisher.event.RateLimitEnabled {
		t.Fatalf("route policies are missing: %#v", publisher.event)
	}
}

func TestAccessEventsHashesPublicClientIP(t *testing.T) {
	publisher := &accessEventUsecaseStub{}
	e := echo.New()
	e.Use(RequestID())
	e.Use(AccessEvents(publisher, testLogger{}))
	e.GET("/public", func(c *echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/public", nil)
	req.RemoteAddr = "192.0.2.55:54321"
	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, req)

	if publisher.event.ClientType != events.ClientTypeIP {
		t.Fatalf("ClientType = %q", publisher.event.ClientType)
	}
	if publisher.event.ClientID == "" || publisher.event.ClientID == "192.0.2.55" {
		t.Fatalf("ClientID = %q, want non-empty hash", publisher.event.ClientID)
	}
}

func TestAccessEventsMarksRateLimitedRequest(t *testing.T) {
	publisher := &accessEventUsecaseStub{}
	e := echo.New()
	e.Use(RequestID())
	e.Use(AccessEvents(publisher, testLogger{}))
	e.GET("/limited", func(c *echo.Context) error {
		setAccessEventRoute(c, domain.Route{Name: "limited", PathPrefix: "/limited"})
		setAccessEventRateLimited(c)
		return c.String(http.StatusTooManyRequests, "rate limit exceeded")
	})

	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/limited", nil))

	if !publisher.event.RateLimited {
		t.Fatal("RateLimited = false, want true")
	}
	if publisher.event.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("StatusCode = %d", publisher.event.StatusCode)
	}
}

func TestAccessEventsPublisherFailureDoesNotChangeResponse(t *testing.T) {
	publisher := &accessEventUsecaseStub{publishErr: errors.New("kafka unavailable")}
	e := echo.New()
	e.Use(RequestID())
	e.Use(AccessEvents(publisher, testLogger{}))
	e.GET("/ok", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/ok", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
}
