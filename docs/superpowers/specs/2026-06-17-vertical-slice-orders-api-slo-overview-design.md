# Design: Vertical Slice — `orders-api` → Prometheus → BFF → SLO Control Center (Overview)

**Date:** 2026-06-17
**Status:** Approved (design); pending implementation plan
**Sub-project:** #1 of the SLO Control Center / Reference Observability Stack

---

## 0. Context

This is the first sub-project of a larger build: a full SLO observability product that
pairs a **bespoke "SLO Control Center" web app** with a **real reference observability
stack** (Prometheus, Grafana/Grafonnet, Loki, Tempo, chaos, runbooks). The full build is
decomposed into independent sub-projects, each with its own spec → plan → implementation
cycle:

| # | Sub-project | Delivers |
|---|-------------|----------|
| **1** | **Vertical slice (this doc)** | `orders-api` + chaos + Prometheus + SLO math + BFF + custom Overview tab + one Grafonnet dashboard |
| 2 | Service mesh | `payments-worker` (Py) + `notification-svc` (Go) + Redis/Postgres + cross-service deps |
| 3 | Telemetry | Loki+promtail + OTel→Tempo + trace↔log↔metric correlation |
| 4 | Alerting + chaos | Burn-rate alerts (fast/slow), `promtool` alert tests, chaos scripts, runbooks |
| 5 | Full frontend | Remaining tabs: Services, Incidents, Alerts, Traces, Logs, Capacity, Runbooks |
| 6 | Deploy + demo | Kustomize/K8s, `make deploy`, demo video |

**Product decisions locked during brainstorming:**
- Deliverable = **full product** (custom frontend AND real backend, both first-class).
- Custom UI and Grafana/Grafonnet dashboards are **both maintained, fully independent**,
  reading the same recording rules.
- Frontend stack = **React + Vite + TypeScript + Tailwind + Tremor** (charts). Chosen for a
  tight dev loop and fast match to the polished dark-dashboard screenshot without
  hand-rolling SVG; no SSR needed for a polling dashboard.
- Frontend gets data via a **dedicated BFF** (not a raw Prometheus reverse-proxy). The UI
  never speaks PromQL; the BFF owns queries and the SLO-packaging contract. This seam is
  reused by every later tab.

---

## 1. Goal

One end-to-end thread that exercises **every architectural seam** we will reuse later, and
is **genuinely demoable**: `docker compose up` brings up a live, continuously-moving
dashboard within ~2 minutes, with reproducible synthetic traffic.

Success criteria:
- `docker compose up` → all services healthy, Overview tab shows a **live, moving** SLO for
  `orders-api`.
- Flipping a chaos env var (`CHAOS_ERROR_RATE`, `CHAOS_LATENCY_MS`) visibly changes the
  SLI, error budget, and burn rate in **both** the custom UI and the Grafana dashboard.
- All four test suites pass (`orders-api`, `slo-bff`, `promtool test rules`, frontend).

---

## 2. Architecture & data flow

```
k6 (steady RPS, fixed RNG seed)
        │ HTTP
        ▼
  orders-api (Go) ──────► Postgres
        │ /metrics  (RED metrics + chaos hooks)
        ▼
   Prometheus ── recording rules (SLI, error budget, burn rate, p95)
        │ HTTP API                         │ HTTP API
        ▼                                   ▼
   slo-bff (Go)                        Grafana (Grafonnet dashboard)
        │ GET /api/slo  (clean JSON contract)
        ▼
  SLO Control Center (React + Vite + TS + Tailwind + Tremor) — Overview tab
```

Both dashboard surfaces read the **same Prometheus recording rules** — single source of
truth, "dashboards-as-code" from day one.

---

## 3. Components

Each component has one clear purpose, a well-defined interface, and is independently
testable.

### 3.1 `orders-api` (Go)
- **Purpose:** sample microservice under observation; the thing whose SLO we track.
- **Endpoints:**
  - `POST /orders` — insert an order row into Postgres, return its id.
  - `GET /orders/:id` — fetch an order.
  - `GET /healthz` — liveness/readiness.
  - `GET /metrics` — Prometheus exposition.
- **RED metrics (Prometheus client_golang):**
  - `http_requests_total{route,method,status}` (counter)
  - `http_request_duration_seconds{route,method}` (histogram; buckets tuned around the
    200ms p95 target, e.g. 5,10,25,50,100,150,200,300,500,1000 ms)
  - `http_requests_in_flight` (gauge)
- **Chaos hooks (env vars, read at startup + optionally hot-reloaded via a `/chaos`
  admin endpoint — startup-only is acceptable for slice 1):**
  - `CHAOS_ERROR_RATE` — float 0..1; fraction of requests returned as 5xx.
  - `CHAOS_LATENCY_MS` — integer; injected delay.
  - `CHAOS_LATENCY_PCT` — float 0..1; fraction of requests that get the injected delay.
- **Storage:** Postgres (schema: single `orders` table). Migrations applied on startup.
- **Dependencies:** Postgres, Prometheus client lib.

### 3.2 Prometheus
- **Purpose:** scrape + store metrics; compute SLO math as recording rules.
- **Config-as-code:** `prometheus.yml` (scrape `orders-api`, `slo-bff`, self) +
  `rules/orders-api.slo.rules.yml`.
- **Recording rules** (see §4 for the math). Computed at multiple windows so the BFF and
  Grafana just read a single series instead of re-deriving.
- **Dependencies:** scrape targets.

### 3.3 `slo-bff` (Go)
- **Purpose:** backend-for-frontend. Owns PromQL; exposes a stable JSON contract to the UI.
- **Endpoints:**
  - `GET /api/slo` →
    ```json
    [
      {
        "service": "orders-api",
        "sliPct": 99.93,
        "targetPct": 99.9,
        "errorBudgetRemainingPct": 84.2,
        "errorBudgetRemainingCount": 1234,
        "burnRate": { "1h": 0.3, "6h": 0.5, "24h": 0.4 },
        "p95Ms": 198,
        "healthy": true
      }
    ]
    ```
  - `GET /api/slo/:service/compliance?window=28d` → `{ points: [{ t, sliPct }, ...] }` for
    the rolling compliance chart.
  - `GET /healthz`.
- **Behavior:** queries the Prometheus HTTP API (`/api/v1/query`, `/api/v1/query_range`),
  maps results into the contract above. Window/target are configured per service (slice 1
  hardcodes `orders-api`'s 99.9% / 28d; later sub-projects make this a config list).
- **Dependencies:** Prometheus HTTP API.

### 3.4 Frontend — SLO Control Center (Overview tab only)
- **Purpose:** the product surface. Slice 1 implements **only the Overview tab**.
- **Stack:** React + Vite + TypeScript + Tailwind + Tremor.
- **Layout (matches screenshot):**
  - Top stat cards: Availability, Latency p95, Error Budget Remaining, Current Burn Rate,
    Services Healthy, Open Incidents. (Open Incidents = static `0`/placeholder in slice 1;
    wired up in sub-project #2+.)
  - "Error Budgets by Service" table: one row per service from `GET /api/slo` (slice 1 =
    just `orders-api`): SLO target, availability, error-budget bar, per-window burn rate.
  - "28-Day Compliance Graph": line chart from `GET /api/slo/:service/compliance`.
  - "Live Service Health Map": minimal single-node render for `orders-api` (the full graph
    arrives with the service mesh in sub-project #2).
- **Data:** polls the BFF (~15s). No PromQL in the client.
- **Dependencies:** `slo-bff`.

### 3.5 Grafana + Grafonnet
- **Purpose:** establish dashboards-as-code; the independent ops-facing dashboard surface.
- **Slice 1 scope:** **one** dashboard ("SLO Overview") generated from a Grafonnet
  `.jsonnet` source, rendering the same recording-rule series (SLI, error budget, burn
  rate, p95). Built via a `make dashboards` (or container) step — zero manual UI clicking.
- **Dependencies:** Prometheus datasource, Grafonnet/jsonnet toolchain.

### 3.6 k6
- **Purpose:** deterministic synthetic load so the dashboard is always live and the demo is
  reproducible.
- **Slice 1 scope:** one script — steady RPS against `POST /orders` + `GET /orders/:id`,
  **fixed RNG seed**, documented pattern. Runs as a Compose service.
- **Dependencies:** `orders-api`.

### 3.7 Docker Compose
- **Services:** `postgres`, `orders-api`, `prometheus`, `slo-bff`, `frontend`, `grafana`,
  `k6`.
- **Goal:** `docker compose up` → full stack live in < 2 min. Healthchecks + `depends_on`
  ordering so the demo "just works".

---

## 4. SLO math (SRE correctness core)

For `orders-api`: **99.9% availability over a 28-day rolling window, 200ms p95 latency.**

Defined as Prometheus **recording rules** (single source of truth for both dashboards):

- **SLI (availability)**, per window `w`:
  `sli_w = sum(rate(http_requests_total{job="orders-api",status!~"5.."}[w])) / sum(rate(http_requests_total{job="orders-api"}[w]))`
  Recorded for `w ∈ {5m, 1h, 6h, 24h, 28d}`.
- **Error budget consumed** (28d): `(1 − sli_28d) / (1 − 0.999)`.
- **Error budget remaining %**: `1 − consumed`, clamped to `[0, 1]`.
- **Error budget remaining count**: `remaining% × total_requests_28d`, where
  `total_requests_28d = sum(increase(http_requests_total{job="orders-api"}[28d]))`.
- **Burn rate** (per window `w ∈ {1h, 6h, 24h}`): `(1 − sli_w) / (1 − 0.999)`.
  (Burn rate is *computed* here; *alerting* on it is sub-project #4.)
- **p95 latency**:
  `histogram_quantile(0.95, sum by (le) (rate(http_request_duration_seconds_bucket{job="orders-api"}[5m])))`.

Recording-rule names follow a `slo:orders_api:<metric>:<window>` convention (finalized in
the plan).

---

## 5. Testing

| Unit | Tooling | Covers |
|------|---------|--------|
| `orders-api` | Go `testing` | chaos injection (error rate / latency math), handlers, metrics emission |
| `slo-bff` | Go `testing` | PromQL-response → JSON contract mapping, against a mock Prometheus client |
| Prometheus rules | `promtool test rules` | recording-rule correctness with synthetic input series |
| Frontend | Vitest (+ Testing Library) | Overview cards/table render from a mocked BFF payload |

CI-readiness (wiring up an actual CI runner) is out of scope for slice 1; the test commands
must run locally via a documented `make test`.

---

## 6. Explicitly out of scope for this slice

Deferred to the named later sub-projects:
- Loki / promtail, OpenTelemetry / Tempo, trace↔log↔metric correlation. *(#3)*
- `payments-worker`, `notification-svc`, Redis, cross-service dependency graph. *(#2)*
- Alertmanager and **alerting** on burn rate / `/healthz` / disk (slice 1 only *computes*
  burn rate). *(#4)*
- Chaos **scenario scripts** + auto-verification + runbooks (slice 1 ships only the chaos
  *hooks* in `orders-api`). *(#4)*
- Frontend tabs other than Overview. *(#5)*
- The full Grafonnet dashboard set beyond the one SLO Overview dashboard. *(#5 / ongoing)*
- Kustomize / K8s manifests, `make deploy`, demo video. *(#6)*
- Production hardening (TLS, secrets vault, RBAC) — explicit v1 non-goal for the whole
  product.

---

## 7. Repository layout (proposed)

```
/services/orders-api/        Go service + Dockerfile + tests
/services/slo-bff/           Go BFF + Dockerfile + tests
/frontend/                   React+Vite+TS app (Overview tab)
/observability/prometheus/   prometheus.yml + rules/ + promtool tests
/observability/grafana/      Grafonnet source + provisioning + build step
/load/k6/                    k6 script(s)
/deploy/compose/             docker-compose.yml (+ env templates)
/docs/                       specs, ADRs, (runbooks later)
Makefile                     up / down / test / dashboards
```

(Exact paths confirmed in the implementation plan.)
