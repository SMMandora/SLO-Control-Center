package main

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// A traceparent emitted by an upstream service must be recovered by extractTrace
// so the delivery span joins the same trace.
func TestExtractTraceRoundTrip(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer func() { _ = tp.Shutdown(context.Background()) }()

	ctx, span := tp.Tracer("upstream").Start(context.Background(), "emit")
	want := traceID(ctx)
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	span.End()

	// Simulate the stream message fields (map[string]any from go-redis).
	fields := map[string]any{"traceparent": carrier["traceparent"]}
	got := traceID(extractTrace(context.Background(), fields))
	if got != want || got == "" {
		t.Fatalf("trace id not propagated: want %q got %q", want, got)
	}
}
