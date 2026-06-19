package main

import (
	"context"
	"regexp"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// injectTrace must emit a W3C traceparent when a span is active, so the worker
// can continue the distributed trace.
func TestInjectTraceProducesTraceparent(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer func() { _ = tp.Shutdown(context.Background()) }()

	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	defer span.End()

	carrier := injectTrace(ctx)
	got := carrier["traceparent"]
	// 00-<32 hex trace id>-<16 hex span id>-<2 hex flags>
	if !regexp.MustCompile(`^00-[0-9a-f]{32}-[0-9a-f]{16}-[0-9a-f]{2}$`).MatchString(got) {
		t.Fatalf("bad traceparent: %q", got)
	}
	if traceID(ctx) == "" {
		t.Fatal("traceID should be non-empty under an active span")
	}
}
