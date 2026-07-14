package tracing

import (
	"net/http"
	"strings"

	"github.com/aralary/edgeguard/internal/platform/httpresponse"

	"github.com/labstack/echo/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const TraceIDHeader = "X-Trace-ID"

func WrapHTTPHandler(service string, handler http.Handler) http.Handler {
	return otelhttp.NewHandler(
		handler,
		service+".http.server",
		otelhttp.WithFilter(func(request *http.Request) bool {
			return !isInfrastructurePath(request.URL.Path)
		}),
		otelhttp.WithSpanNameFormatter(func(_ string, request *http.Request) string {
			return request.Method + " " + request.URL.Path
		}),
	)
}

func HTTPTransport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return otelhttp.NewTransport(base)
}

func RouteMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			request := c.Request()
			if isInfrastructurePath(request.URL.Path) {
				return next(c)
			}

			span := oteltrace.SpanFromContext(request.Context())
			if span.SpanContext().IsValid() {
				c.Response().Header().Set(TraceIDHeader, span.SpanContext().TraceID().String())
			}

			originalResponse := c.Response()
			recorder := httpresponse.NewRecorder(originalResponse)
			c.SetResponse(recorder)
			defer c.SetResponse(originalResponse)

			err := next(c)
			if !span.SpanContext().IsValid() {
				return err
			}

			route := strings.TrimSpace(c.Path())
			if route == "" {
				route = "unmatched"
			}

			statusCode := recorder.StatusCode()
			if err != nil && statusCode < http.StatusBadRequest {
				statusCode = http.StatusInternalServerError
			}

			span.SetName(request.Method + " " + route)
			span.SetAttributes(
				attribute.String("http.route", route),
				attribute.Int("http.response.status_code", statusCode),
			)
			if err != nil {
				span.RecordError(err)
			}
			if statusCode >= http.StatusInternalServerError {
				span.SetStatus(codes.Error, http.StatusText(statusCode))
			}

			return err
		}
	}
}

func isInfrastructurePath(path string) bool {
	switch path {
	case "/health", "/ready", "/metrics":
		return true
	default:
		return false
	}
}
