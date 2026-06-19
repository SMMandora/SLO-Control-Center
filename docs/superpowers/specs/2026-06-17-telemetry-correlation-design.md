# Design: Sub-project #3 — Telemetry & Correlation

**Date:** 2026-06-17
**Status:** Approved (design)
**Sub-project:** #3 of the SLO Control Center / Reference Observability Stack

## Goal

Distributed traces, logs, and metrics that cross-link, so an SLI breach can be
followed exemplar → trace → logs → root cause. The demonstrable payload: a flaky
`mock-gateway` charge yields a failed payment whose **single distributed trace
spans orders-api → payments-worker → notification-svc**, with logs correlated by
`trace_id`.

## Locked decisions

- Trace context **propagates through Redis Streams** (inject `traceparent` into
  the message on XADD, extract in the consumer) → one payment = one end-to-end
  trace across all three services.
- Logs via **promtail tailing Docker container logs**; services emit structured
  JSON logs carrying `trace_id`/`span_id`. No per-service log shipping.
- Correlation surfaced **primarily in Grafana** (Tempo + Loki datasources,
  derived fields, exemplars). The custom UI gets only a thin "recent violation
  traces" list; full Traces/Logs tabs remain sub-project #5.

## Components

- **otel-collector** — OTLP gRPC (:4317) / HTTP (:4318) receiver; exports traces
  to Tempo. Single pipeline, batch processor.
- **tempo** — trace store; Grafana datasource; HTTP API (:3200) for the BFF.
- **loki** — log store; Grafana datasource.
- **promtail** — tails `/var/lib/docker/containers/*/*-json.log`, parses the JSON
  message body, promotes `trace_id` to a label-free structured field for
  derived-field linking.

## Instrumentation

### Go services (orders-api, slo-bff, notification-svc, mock-gateway, mock-receiver)
Each gains a small `tracing.go`: OTLP gRPC exporter → `OTEL_EXPORTER_OTLP_ENDPOINT`
(default `otel-collector:4317`), `sdk/trace` provider, `otel/propagation`
TraceContext. HTTP servers wrapped with `otelhttp.NewHandler`; outbound clients
use `otelhttp.NewTransport`. Each service logs structured JSON including the active
`trace_id`.

### Redis propagation (orders-api producer, payments-worker + notification-svc consumers)
- Producer: a `propagation.MapCarrier` is injected into the XADD field map under
  standard keys (`traceparent`). Pure helper `injectTrace(ctx) map[string]string`.
- Consumer: extract the carrier from the message fields to set the parent context,
  then start a child span. Pure helper `extractTrace(fields) context.Context`.

### Python payments-worker
`opentelemetry-sdk` + OTLP exporter + `opentelemetry-instrumentation-requests`.
Extracts `traceparent` from the stream message via `TraceContextTextMapPropagator`,
starts a span as the payment's processing root, and logs JSON with `trace_id`.

### Exemplars
orders-api's `http_request_duration_seconds` observed via `ObserveWithExemplar`
attaching `{trace_id}`. Prometheus runs with `--enable-feature=exemplar-storage`.
Grafana shows exemplars on latency panels → click → Tempo.

## Grafana (provisioned, as-code)

- Datasources: Tempo (with `tracesToLogsV2` → Loki on `trace_id`) and Loki (with a
  derived field `trace_id` → Tempo).
- Prometheus datasource gains `exemplarTraceIdDestinations` → Tempo.
- New **Incident Investigation** dashboard (Grafonnet): error-rate timeseries +
  a Loki logs panel (errors) + a Tempo search panel (slow/error traces), all
  driven by the dashboard time range.

## BFF + custom UI (minimal)

- BFF `GET /api/traces/recent` → queries Tempo's search API for recent
  error/slow payment traces; returns `[{traceID, service, startedMs, durationMs,
  grafanaUrl}]`. Pure mapping `parseTempoSearch(json) []TraceRef` unit-tested
  against a mock Tempo response.
- SLO Overview gains a **"Recent violations → trace"** list rendering those, each
  linking to Grafana's trace view.

## Testing

| Unit | Tooling | Covers |
|---|---|---|
| orders-api | Go test | `injectTrace` writes a valid `traceparent` |
| notification-svc / payments-worker | Go test / pytest | `extractTrace` round-trips a carrier |
| slo-bff | Go test | `parseTempoSearch` mapping |
| frontend | vitest | RecentTraces renders rows + links |
| integration | manual | one failed payment → one multi-service trace in Tempo + correlated logs in Loki |

## Compose

Add `otel-collector`, `tempo`, `loki`, `promtail`. All services get
`OTEL_EXPORTER_OTLP_ENDPOINT`. Prometheus gets `--enable-feature=exemplar-storage`.
Grafana gets Tempo/Loki datasource provisioning. promtail mounts the Docker
container log dir + socket.

## Out of scope (later)

Full Traces/Logs custom-UI tabs (#5); Alertmanager (#4); metrics-from-spans
(span metrics connector); tail-based sampling; K8s (#6).
