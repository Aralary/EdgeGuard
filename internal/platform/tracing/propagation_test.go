package tracing

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestInjectExtractTraceContext(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})

	spanContext := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    oteltrace.TraceID{1, 2, 3},
		SpanID:     oteltrace.SpanID{4, 5, 6},
		TraceFlags: oteltrace.FlagsSampled,
	})
	ctx := oteltrace.ContextWithSpanContext(context.Background(), spanContext)

	headers := Inject(ctx)
	extracted := oteltrace.SpanContextFromContext(Extract(context.Background(), headers))
	if extracted.TraceID() != spanContext.TraceID() {
		t.Fatalf("TraceID = %s, want %s", extracted.TraceID(), spanContext.TraceID())
	}
}
