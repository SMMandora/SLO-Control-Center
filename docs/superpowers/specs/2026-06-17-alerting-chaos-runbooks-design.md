# Design: Sub-project #4 — Alerting, Chaos & Runbooks

**Date:** 2026-06-17
**Status:** Approved (design)
**Sub-project:** #4 of the SLO Control Center / Reference Observability Stack

## Goal

Burn-rate + liveness alerting wired to Alertmanager, each alert linked to a
runbook, plus chaos scenarios that perturb the stack and **auto-verify the
expected alert fires**.

## Locked decisions

- **No Pushgateway** (spec lists it) — no batch jobs push metrics here. YAGNI.
- **Disk alert** rule + runbook + cadvisor metric ship and are promtool-tested,
  but the live "fill disk to 95%" chaos run is deferred to #6 (K8s ephemeral
  storage); 3 chaos scenarios run live in #4.
- Latency chaos uses a new `LATENCY_MS` knob on **mock-gateway**.
- Alerts route by severity to a **mock alert-receiver** (webhook sink) so firing
  is observable; chaos scripts auto-verify via Prometheus `/api/v1/alerts`.

## Alert rules (`observability/prometheus/rules/alerts.yml`)

Thresholds derived from budget math (burn rate = `(1-SLI)/budget`, already
normalized per service, so one threshold fits all services):

| Alert | Expr (per service) | For | Severity |
|---|---|---|---|
| Fast burn | `slo:<svc>:burnrate_1h > 13.44` (≈2% budget/1h) | 2m | page |
| Slow burn | `slo:<svc>:burnrate_6h > 11.2` (≈10% budget/6h) | 15m | ticket |
| Service down | `up{job=~"orders-api\|payments-worker\|notification-svc"} == 0` | 2m | page |
| Disk space | `max by (name)(container_fs_usage_bytes/container_fs_limit_bytes) > 0.9` | 5m | warn |

Services covered: orders-api, payments-worker, notification-svc. Every alert sets
`annotations.runbook_url` → `docs/runbooks/<name>.md` and a summary/description.

## Alertmanager (`observability/alertmanager/alertmanager.yml`)

Route by `severity`: `page` → receiver `pager`, `ticket` → `ticketing`,
`warn` → `warnings`. All three are `webhook_configs` pointing at the mock
alert-receiver (`/page`, `/ticket`, `/warn`). `group_wait: 5s` for a snappy demo.

## mock alert-receiver (Go, new)

Tiny webhook sink: `POST /{page,ticket,warn}` parse the Alertmanager payload and
append `{alertname, severity, status}` to an in-memory ring; `GET /alerts` returns
them; `/healthz`, `/metrics` (`alerts_received_total{severity}`). Pure helper
`parseAMWebhook([]byte) []AlertRef` unit-tested.

## cadvisor

`gcr.io/cadvisor/cadvisor` scraped by Prometheus (`job=cadvisor`) for
`container_fs_*` (disk alert) + CPU/mem (future Capacity dashboard).

## Runbooks (`docs/runbooks/*.md`)

`fast-burn.md`, `slow-burn.md`, `service-down.md`, `disk-space.md` — each:
symptoms · likely causes · first checks (with PromQL/commands) · mitigation ·
related dashboards. Plus a `README.md` index.

## Chaos scenarios (`chaos/*.sh`)

Each script: print intent → perturb → poll Prometheus `/api/v1/alerts` until the
expected alert is `firing` (timeout) → print PASS/FAIL → revert. A `chaos/README.md`
documents each scenario + expected outcome.

1. **error-burst.sh** — recreate orders-api with `CHAOS_ERROR_RATE=0.3` →
   expect `FastBurn` (orders-api) firing → revert to 0.
2. **payments-latency.sh** — recreate mock-gateway with `LATENCY_MS=900` →
   expect payments p95 breach + `SlowBurn` (payments-worker) → revert.
3. **service-down.sh** — `docker compose stop orders-api` → expect
   `ServiceDown` → `start` orders-api.

## Tests

- **promtool** `alerts.test.yml` — synthetic series assert each alert fires at
  threshold and stays quiet below it.
- **alert-receiver** Go test — `parseAMWebhook` mapping.

## Compose

Add `alertmanager`, `alert-receiver`, `cadvisor`. Prometheus gains `alerting:`
(→ alertmanager:9093) + the `alerts.yml` rule file. mock-gateway gains `LATENCY_MS`.

## Out of scope (later)

Real paging integrations (PagerDuty/Slack); the live disk-fill scenario (#6);
Capacity/USE dashboard build (uses the cadvisor metrics added here) — #5.
