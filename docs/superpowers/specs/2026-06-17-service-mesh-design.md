# Design: Sub-project #2 — Service Mesh

**Date:** 2026-06-17
**Status:** Approved (design)
**Sub-project:** #2 of the SLO Control Center / Reference Observability Stack

## Goal

Extend the slice-1 stack into a real multi-service mesh with cross-service
dependencies and two **latency-based** SLOs, so the dashboard shows 3 services,
dependency health, and the new burn-rate behaviour.

## Locked decisions

- **Transport:** Redis Streams (consumer groups + message IDs/timestamps support
  the "processed within N seconds" latency SLOs and durable delivery).
- **Mock dependencies:** separate tiny services (`mock-gateway`, `mock-receiver`)
  so the dependency graph has real network hops to monitor.
- **Deterministic flake:** mock-gateway fails on `hash(order_id) % 100 < FAIL_PCT`
  (default 8), reproducible, env-overridable. No RNG.
- **Compliance chart** stays orders-api-only for now (per-service selector is a
  later UI sub-project).

## Data flow

```
k6 → orders-api ──POST /orders──> Postgres
                   │ XADD payments {order_id, enqueued_at}
                   ▼
   Redis Stream "payments" ──group──> payments-worker (Python)
                                        │ POST /charge → mock-gateway (flake)
                                        │ success: XADD notifications {emitted_at}
                                        ▼
   Redis Stream "notifications" ──group──> notification-svc (Go)
                                        │ POST /webhook → mock-receiver
```

## Components

### orders-api (modify)
On successful `POST /orders`, `XADD` to the `payments` stream a payload of
`{order_id, item, qty, enqueued_at}` (enqueued_at = unix millis). New env
`REDIS_URL` (default `redis://redis:6379`). If Redis is unavailable the order
still succeeds (enqueue is best-effort, logged). No metric changes.

### payments-worker (Python, new)
- Consumer group `payments-workers` on stream `payments` via `XREADGROUP BLOCK`.
- For each message: `POST /charge` to mock-gateway with the order id.
  - 2xx → success; emit to `notifications` stream `{order_id, emitted_at}`; `XACK`.
  - non-2xx → retry up to 3×; on final failure record a failure and `XACK`
    (slice-2 does not implement a separate dead-letter stream — a failed payment
    is simply counted against the SLO).
- Metrics (`prometheus_client`, `/metrics` on :8000):
  - `payments_processed_total{status="success|failure"}` counter
  - `payments_e2e_latency_seconds` histogram (now − enqueued_at), buckets
    `[.5,1,2,5,10,30,60,120]`, labelled `status`.
- Pure, unit-tested helper `classify(enqueued_ms, done_ms, ok) -> (status, latency_s)`.

### notification-svc (Go, new)
- Consumer group `notification-svcs` on stream `notifications`.
- For each: `POST /webhook` to mock-receiver. 2xx → success, else failure (retry 3×).
- Metrics (`/metrics` on :8081):
  - `notifications_delivered_total{status}` counter
  - `notification_delivery_latency_seconds` histogram (now − emitted_at), buckets
    `[.1,.25,.5,1,2,5,10]`, labelled `status`.

### mock-gateway (Go, tiny)
`POST /charge {order_id}` → fail when `fnv(order_id) % 100 < FAIL_PCT` (env
`FAIL_PCT`, default 8): returns 502. Else 200 `{status:"charged"}`. `/healthz`,
`/metrics`. Pure helper `shouldFail(orderID string, failPct int) bool`.

### mock-receiver (Go, tiny)
`POST /webhook` → 200, increments `webhooks_received_total`. `/healthz`, `/metrics`.

## SLO recording rules (new, latency-based)

For window `w ∈ {5m,1h,6h,24h,28d}`:

- **payments-worker** (target 99.5%, budget 0.005):
  `slo:payments_worker:good:ratio_w =
   sum(rate(payments_e2e_latency_seconds_bucket{job="payments-worker",status="success",le="60"}[w]))
   / sum(rate(payments_processed_total{job="payments-worker"}[w]))`
- **notification-svc** (target 99%, budget 0.01):
  `slo:notification_svc:good:ratio_w =
   sum(rate(notification_delivery_latency_seconds_bucket{job="notification-svc",status="success",le="5"}[w]))
   / sum(rate(notifications_delivered_total{job="notification-svc"}[w]))`

Each service also gets: `burnrate_{1h,6h,24h}` = `(1 - ratio_w)/budget`,
`error_budget_remaining_ratio` (clamped), `requests_total_28d`, and a
`latency_p95_seconds` from its histogram. Same shape as orders-api's rules.

## BFF refactor

`ServiceSLO` gains a `Prefix` (e.g. `slo:payments_worker`) and a `SLIRule`
(e.g. `slo:payments_worker:good:ratio_28d`). `BuildSummary` builds query strings
from the prefix instead of hardcoding `slo:orders_api:*`. The `services` list:

| service | target | prefix | budget |
|---|---|---|---|
| orders-api | 99.9 | slo:orders_api:availability | 0.001 |
| payments-worker | 99.5 | slo:payments_worker:good | 0.005 |
| notification-svc | 99.0 | slo:notification_svc:good | 0.010 |

JSON contract is unchanged; the frontend receives 3 rows.

## Frontend

- Overview table, "Services Healthy" card, and burn-rate column auto-populate to 3.
- **ServiceHealthMap** upgrades to a left-to-right dependency chain
  (orders-api → payments-worker → notification-svc) with arrows between nodes,
  each node coloured by `healthy`.

## Testing

| Unit | Tooling | Covers |
|---|---|---|
| payments-worker | pytest | `classify()` deadline logic |
| mock-gateway | Go test | `shouldFail()` determinism |
| notification-svc | Go test | delivery classification helper |
| slo-bff | Go test | generalized `BuildSummary` with payments prefix |
| Prometheus rules | promtool | payments + notification latency-SLO ratios |
| frontend | vitest | health map renders 3 dependency nodes |

Load: existing k6 unchanged; orders flow through the whole chain.

## Compose

Add `redis`, `payments-worker`, `notification-svc`, `mock-gateway`,
`mock-receiver`. orders-api gets `REDIS_URL`. Prometheus scrapes the four new
`/metrics` endpoints.

## Out of scope (later sub-projects)

Loki/Tempo + trace correlation (#3); Alertmanager alerting + chaos scripts +
runbooks (#4); per-service compliance selector and remaining UI tabs (#5);
K8s (#6); dead-letter queue handling and payment idempotency hardening.
