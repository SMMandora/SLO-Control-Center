from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.trace import format_trace_id

import tracing


def test_extract_roundtrip():
    trace.set_tracer_provider(TracerProvider())
    tracer = trace.get_tracer("test")
    with tracer.start_as_current_span("emit"):
        carrier = tracing.inject_context()

    assert "traceparent" in carrier
    parent = tracing.extract_context({"traceparent": carrier["traceparent"]})
    span = trace.get_current_span(parent)
    tid = format_trace_id(span.get_span_context().trace_id)
    # traceparent = 00-<trace id>-<span id>-<flags>
    assert tid == carrier["traceparent"].split("-")[1]
