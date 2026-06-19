# Design: Sub-project #5 — Full Frontend + Drilldown/Capacity Dashboards

**Date:** 2026-06-17
**Status:** Approved (design)
**Sub-project:** #5 of the SLO Control Center / Reference Observability Stack

## Goal

Complete the SLO Control Center UI: the 7 remaining tabs (Services, Incidents,
Alerts, Traces, Logs, Capacity, Runbooks) wired to real backends via the BFF,
plus the Grafana **Service Drilldown** and **Capacity / USE** dashboards.

## Locked decisions

- **react-router-dom** for client-side routing + a top nav bar matching the
  screenshot.
- **Incidents = derived from active alerts** (grouped by alertname×service with a
  start time); no separate incident datastore.
- **Traces/Logs tabs** = bespoke summary lists in the custom UI, each with
  "open in Grafana" deep links for the full waterfall/explorer.
- UI never speaks PromQL/LogQL/TraceQL — every tab goes through a BFF endpoint.

## BFF endpoints (Go, each with a unit-tested pure mapper)

| Endpoint | Backend | Returns |
|----------|---------|---------|
| `GET /api/services` | Prometheus (SLO rules + per-service rate) | `[{service, slo, ratePerSec, errorPct, p95Ms, dependencies[]}]` |
| `GET /api/alerts` | Prometheus `/api/v1/alerts` | `[{alertname, severity, service, state, activeAt, summary, runbookUrl}]` |
| `GET /api/incidents` | derived from `/api/alerts` (firing only) | `[{id, title, service, severity, startedAt, alertCount}]` |
| `GET /api/traces/recent?status=error\|any` | Tempo search | extend existing; `any` drops the error filter |
| `GET /api/logs?level=` | Loki `query_range` | `[{tsMs, level, service, traceID, line}]` |
| `GET /api/capacity` | Prometheus (cadvisor) | `[{name, cpuPct, memMB, diskPct}]` |
| `GET /api/runbooks` | files mounted at `/runbooks` | `[{name, title, markdown}]` |

New BFF deps: a Loki client (`logs.go`) and a runbook reader (`runbooks.go`,
reads `RUNBOOKS_DIR`, default `/runbooks`). `ServiceSLO` gains `RateQuery` +
`Dependencies`.

## Frontend

- `react-router-dom` with routes for each tab; `NavBar` + `Layout` (header,
  environment/time pills are static placeholders).
- Pages (each split into a prop-driven `*View` for testing + a container hook):
  - **Services** — cards per service: SLO %, RED (rate/error/p95), dependency chips.
  - **Alerts** — table of active alerts with severity badges + runbook links.
  - **Incidents** — list of derived incidents (title, service, since, severity).
  - **Traces** — recent traces (service, op, duration, status) → Grafana link.
  - **Logs** — recent log lines (time, level badge, service, message) with a
    level filter; trace_id → Grafana link.
  - **Capacity** — per-container CPU/mem/disk bars.
  - **Runbooks** — list + rendered markdown (`react-markdown`).
- New deps: `react-router-dom`, `react-markdown`.

## Grafana (Grafonnet)

- **service-drilldown.jsonnet** — `service` template variable; RED panels
  (request rate, error rate, p95) + SLI/burn-rate + dependency `up` panel.
- **capacity-use.jsonnet** — per-container CPU, memory, disk, network from
  cadvisor. Completes the 4 required dashboards. Build wired into `make dashboards`.

## Testing

- BFF: Go unit tests for `parseAlerts`, `deriveIncidents`, `parseLokiQuery`,
  `parseCapacity`, runbook title extraction.
- Frontend: vitest render tests for each `*View`.
- Grafana: both dashboards compile via jsonnet.
- Live: every tab renders real data; both dashboards provision (200).

## Compose

- slo-bff: mount `../../docs/runbooks:/runbooks:ro`, add `LOKI_URL=http://loki:3100`.
- Grafana: two new provisioned dashboards.

## Out of scope (later)

Full bespoke trace-waterfall / log-explorer (Grafana covers the deep dive);
auth/multi-tenant; the static header pills becoming live env/time selectors.
K8s (#6).
