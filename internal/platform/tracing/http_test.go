package tracing

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestRouteMiddlewareExposesTraceID(t *testing.T) {
	spanContext := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    oteltrace.TraceID{1, 2, 3},
		SpanID:     oteltrace.SpanID{4, 5, 6},
		TraceFlags: oteltrace.FlagsSampled,
	})

	e := echo.New()
	e.Use(RouteMiddleware())
	e.GET("/orders/:id", func(c *echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/orders/123", nil)
	request = request.WithContext(oteltrace.ContextWithSpanContext(request.Context(), spanContext))
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)

	if got := response.Header().Get(TraceIDHeader); got != spanContext.TraceID().String() {
		t.Fatalf("%s = %q, want %q", TraceIDHeader, got, spanContext.TraceID().String())
	}
}
