"""OpenTelemetry setup + W3C trace-context propagation helpers for the worker."""

import os

from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.propagate import extract, inject
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.trace import format_trace_id


def init_tracer() -> None:
    """Wire an OTLP/gRPC exporter from OTEL_EXPORTER_OTLP_ENDPOINT (no-op if unset)."""
    endpoint = os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
    if not endpoint:
        return
    name = os.getenv("OTEL_SERVICE_NAME", "payments-worker")
    provider = TracerProvider(resource=Resource.create({"service.name": name}))
    provider.add_span_processor(
        BatchSpanProcessor(OTLPSpanExporter(endpoint=endpoint, insecure=True))
    )
    trace.set_tracer_provider(provider)


def extract_context(fields: dict):
    """Continue the distributed trace carried in a stream message's fields."""
    carrier = {}
    if "traceparent" in fields:
        carrier["traceparent"] = fields["traceparent"]
    return extract(carrier)


def inject_context() -> dict:
    """Serialize the current trace context for embedding in the next message."""
    carrier: dict = {}
    inject(carrier)
    return carrier


def current_trace_id() -> str:
    ctx = trace.get_current_span().get_span_context()
    if ctx.trace_id:
        return format_trace_id(ctx.trace_id)
    return ""
