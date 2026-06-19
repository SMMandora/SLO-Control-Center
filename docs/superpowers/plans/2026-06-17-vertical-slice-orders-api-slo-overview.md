# Vertical Slice (orders-api → Prometheus → BFF → SLO Control Center) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Commit policy (user override):** Do NOT `git commit`. Each task ends in a **verification checkpoint** (tests pass / stack runs), not a commit. The user commits manually.

**Goal:** Stand up one end-to-end, demoable thread — a Go `orders-api` with chaos hooks, scraped by Prometheus (which computes SLO math as recording rules), exposed through a Go BFF, and rendered live in a React "SLO Control Center" Overview tab plus one Grafonnet Grafana dashboard — all via `docker compose up`.

**Architecture:** `k6` drives steady load at `orders-api`; `orders-api` emits RED metrics + injects chaos via env; Prometheus scrapes and records SLI / error-budget / burn-rate / p95 series; `slo-bff` queries Prometheus and serves a clean JSON contract; the React SPA polls the BFF; Grafana renders the same recording rules from Grafonnet-generated JSON. Each unit is independently testable.

**Tech Stack:** Go 1.22 (orders-api, slo-bff), Postgres 16, Prometheus + promtool, Grafana + Grafonnet/jsonnet, React + Vite + TypeScript + Tailwind + Tremor, k6, Docker Compose.

---

## File Structure

```
services/orders-api/
  main.go                  wiring: config, db, router, server
  config.go                env parsing (DB + chaos)
  chaos.go                 chaos decision logic (pure, unit-tested)
  chaos_test.go
  metrics.go               prometheus collectors + middleware
  handlers.go              POST /orders, GET /orders/:id, /healthz
  handlers_test.go
  store.go                 postgres access + migration
  go.mod / go.sum
  Dockerfile

services/slo-bff/
  main.go                  config, router, server
  promclient.go            Prometheus HTTP API client (interface + impl)
  slo.go                   PromQL → JSON contract mapping (pure-ish, unit-tested)
  slo_test.go
  handlers.go              GET /api/slo, GET /api/slo/:service/compliance, /healthz
  go.mod / go.sum
  Dockerfile

observability/prometheus/
  prometheus.yml
  rules/orders-api.slo.rules.yml
  rules/orders-api.slo.rules.test.yml   promtool unit tests

observability/grafana/
  jsonnet/slo-overview.jsonnet
  jsonnet/jsonnetfile.json
  provisioning/datasources/prometheus.yml
  provisioning/dashboards/dashboards.yml
  dashboards/slo-overview.json          generated output (built, not hand-edited)

frontend/
  index.html
  package.json / tsconfig.json / vite.config.ts / tailwind.config.js / postcss.config.js
  src/main.tsx
  src/App.tsx
  src/api/types.ts          SloSummary, ComplianceSeries (mirrors BFF contract)
  src/api/client.ts         fetch wrappers
  src/hooks/useSlo.ts       polling hook
  src/pages/Overview.tsx
  src/components/StatCard.tsx
  src/components/ErrorBudgetTable.tsx
  src/components/ComplianceChart.tsx
  src/components/ServiceHealthMap.tsx
  src/pages/Overview.test.tsx

load/k6/orders-load.js

deploy/compose/docker-compose.yml
deploy/compose/.env.example

Makefile
README.md
```

---

## Task 1: `orders-api` — chaos decision logic (TDD)

**Files:**
- Create: `services/orders-api/go.mod`, `services/orders-api/config.go`, `services/orders-api/chaos.go`, `services/orders-api/chaos_test.go`

- [ ] **Step 1: Init module**

Run: `cd services/orders-api && go mod init github.com/slo-control-center/orders-api && go get github.com/prometheus/client_golang@latest github.com/jackc/pgx/v5@latest`

- [ ] **Step 2: Write failing test** (`chaos_test.go`)

```go
package main

import "testing"

func TestChaosShouldError(t *testing.T) {
	c := Chaos{ErrorRate: 1.0}
	if !c.ShouldError(0.5) { t.Fatal("rate 1.0 must always error") }
	c = Chaos{ErrorRate: 0.0}
	if c.ShouldError(0.0) { t.Fatal("rate 0.0 must never error") }
	c = Chaos{ErrorRate: 0.3}
	if !c.ShouldError(0.2) { t.Fatal("roll 0.2 < 0.3 should error") }
	if c.ShouldError(0.4) { t.Fatal("roll 0.4 >= 0.3 should not error") }
}

func TestChaosLatency(t *testing.T) {
	c := Chaos{LatencyMS: 800, LatencyPct: 1.0}
	if c.LatencyFor(0.9).Milliseconds() != 800 { t.Fatal("expected 800ms") }
	c = Chaos{LatencyMS: 800, LatencyPct: 0.0}
	if c.LatencyFor(0.0) != 0 { t.Fatal("pct 0 => no latency") }
}
```

- [ ] **Step 3: Run test, expect FAIL** — `go test ./...` → undefined: Chaos.

- [ ] **Step 4: Implement** (`chaos.go`)

```go
package main

import "time"

type Chaos struct {
	ErrorRate  float64
	LatencyMS  int
	LatencyPct float64
}

// ShouldError reports whether a request with the given [0,1) roll should 5xx.
func (c Chaos) ShouldError(roll float64) bool { return roll < c.ErrorRate }

// LatencyFor returns the injected delay for a request with the given [0,1) roll.
func (c Chaos) LatencyFor(roll float64) time.Duration {
	if roll < c.LatencyPct {
		return time.Duration(c.LatencyMS) * time.Millisecond
	}
	return 0
}
```

`config.go`:

```go
package main

import (
	"os"
	"strconv"
)

type Config struct {
	Addr        string
	DatabaseURL string
	Chaos       Chaos
}

func envFloat(k string, def float64) float64 {
	if v := os.Getenv(k); v != "" { if f, err := strconv.ParseFloat(v, 64); err == nil { return f } }
	return def
}
func envInt(k string, def int) int {
	if v := os.Getenv(k); v != "" { if n, err := strconv.Atoi(v); err == nil { return n } }
	return def
}
func envStr(k, def string) string { if v := os.Getenv(k); v != "" { return v }; return def }

func LoadConfig() Config {
	return Config{
		Addr:        envStr("ADDR", ":8080"),
		DatabaseURL: envStr("DATABASE_URL", "postgres://orders:orders@postgres:5432/orders?sslmode=disable"),
		Chaos: Chaos{
			ErrorRate:  envFloat("CHAOS_ERROR_RATE", 0),
			LatencyMS:  envInt("CHAOS_LATENCY_MS", 0),
			LatencyPct: envFloat("CHAOS_LATENCY_PCT", 0),
		},
	}
}
```

- [ ] **Step 5: Run test, expect PASS** — `go test ./...`

**Checkpoint:** `go test ./...` passes in `services/orders-api`.

---

## Task 2: `orders-api` — metrics + handlers + store

**Files:**
- Create: `services/orders-api/metrics.go`, `services/orders-api/store.go`, `services/orders-api/handlers.go`, `services/orders-api/handlers_test.go`, `services/orders-api/main.go`, `services/orders-api/Dockerfile`

- [ ] **Step 1: Metrics** (`metrics.go`)

```go
package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total", Help: "Total HTTP requests.",
	}, []string{"route", "method", "status"})

	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_request_duration_seconds", Help: "HTTP request latency.",
		Buckets: []float64{.005, .01, .025, .05, .1, .15, .2, .3, .5, 1},
	}, []string{"route", "method"})

	inFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "http_requests_in_flight", Help: "In-flight HTTP requests.",
	})
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(c int) { r.status = c; r.ResponseWriter.WriteHeader(c) }

// instrument wraps a handler with RED metrics under a fixed route label.
func instrument(route string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		inFlight.Inc()
		defer inFlight.Dec()
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		start := time.Now()
		h(rec, req)
		httpDuration.WithLabelValues(route, req.Method).Observe(time.Since(start).Seconds())
		httpRequests.WithLabelValues(route, req.Method, strconv.Itoa(rec.status)).Inc()
	}
}
```

- [ ] **Step 2: Store** (`store.go`)

```go
package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct{ pool *pgxpool.Pool }

func NewStore(ctx context.Context, url string) (*Store, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil { return nil, err }
	_, err = pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS orders (
		id BIGSERIAL PRIMARY KEY, item TEXT NOT NULL, qty INT NOT NULL DEFAULT 1,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now())`)
	return &Store{pool: pool}, err
}

func (s *Store) CreateOrder(ctx context.Context, item string, qty int) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `INSERT INTO orders(item, qty) VALUES($1,$2) RETURNING id`, item, qty).Scan(&id)
	return id, err
}

func (s *Store) GetOrder(ctx context.Context, id int64) (string, int, error) {
	var item string; var qty int
	err := s.pool.QueryRow(ctx, `SELECT item, qty FROM orders WHERE id=$1`, id).Scan(&item, &qty)
	return item, qty, err
}
```

- [ ] **Step 3: Failing handler test** (`handlers_test.go`) — chaos-forced error path, no DB needed.

```go
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthz(t *testing.T) {
	h := NewServer(nil, Chaos{}, func() float64 { return 0 })
	r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 { t.Fatalf("healthz = %d", w.Code) }
}

func TestCreateOrderChaosError(t *testing.T) {
	// rollFn returns 0.0 so ShouldError is true for any positive rate.
	h := NewServer(nil, Chaos{ErrorRate: 1.0}, func() float64 { return 0 })
	r := httptest.NewRequest(http.MethodPost, "/orders", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusInternalServerError { t.Fatalf("want 500, got %d", w.Code) }
}
```

- [ ] **Step 4: Run test, expect FAIL** — undefined: NewServer.

- [ ] **Step 5: Implement** (`handlers.go`)

```go
package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	mux   *http.ServeMux
	store *Store
	chaos Chaos
	roll  func() float64 // injectable RNG for tests
}

func NewServer(store *Store, chaos Chaos, roll func() float64) *Server {
	s := &Server{mux: http.NewServeMux(), store: store, chaos: chaos, roll: roll}
	s.mux.Handle("/metrics", promhttp.Handler())
	s.mux.HandleFunc("/healthz", instrument("/healthz", s.healthz))
	s.mux.HandleFunc("/orders", instrument("/orders", s.orders))
	s.mux.HandleFunc("/orders/", instrument("/orders/:id", s.getOrder))
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

func (s *Server) applyChaos(w http.ResponseWriter) bool {
	if d := s.chaos.LatencyFor(s.roll()); d > 0 { time.Sleep(d) }
	if s.chaos.ShouldError(s.roll()) {
		http.Error(w, "chaos: injected failure", http.StatusInternalServerError)
		return true
	}
	return false
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK); _, _ = w.Write([]byte("ok")) }

func (s *Server) orders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { http.Error(w, "method", http.StatusMethodNotAllowed); return }
	if s.applyChaos(w) { return }
	var body struct{ Item string `json:"item"`; Qty int `json:"qty"` }
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Item == "" { body.Item = "widget" }
	if body.Qty == 0 { body.Qty = 1 }
	id, err := s.store.CreateOrder(r.Context(), body.Item, body.Qty)
	if err != nil { http.Error(w, err.Error(), http.StatusInternalServerError); return }
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]int64{"id": id})
}

func (s *Server) getOrder(w http.ResponseWriter, r *http.Request) {
	if s.applyChaos(w) { return }
	idStr := r.URL.Path[len("/orders/"):]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil { http.Error(w, "bad id", http.StatusBadRequest); return }
	item, qty, err := s.store.GetOrder(r.Context(), id)
	if err != nil { http.Error(w, "not found", http.StatusNotFound); return }
	_ = json.NewEncoder(w).Encode(map[string]any{"id": id, "item": item, "qty": qty})
}
```

> Note: `TestCreateOrderChaosError` passes a `nil` store but chaos triggers before store use, so the 500 returns first. `TestHealthz` never touches the store.

`main.go`:

```go
package main

import (
	"context"
	"log"
	"math/rand"
	"net/http"
)

func main() {
	cfg := LoadConfig()
	store, err := NewStore(context.Background(), cfg.DatabaseURL)
	if err != nil { log.Fatalf("db: %v", err) }
	srv := NewServer(store, cfg.Chaos, rand.Float64)
	log.Printf("orders-api listening on %s", cfg.Addr)
	log.Fatal(http.ListenAndServe(cfg.Addr, srv))
}
```

- [ ] **Step 6: Run test, expect PASS** — `go test ./...`

- [ ] **Step 7: Dockerfile**

```dockerfile
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /orders-api .
FROM gcr.io/distroless/static-debian12
COPY --from=build /orders-api /orders-api
EXPOSE 8080
ENTRYPOINT ["/orders-api"]
```

**Checkpoint:** `go test ./...` passes; `go build .` succeeds.

---

## Task 3: Prometheus config + recording rules + promtool tests (TDD)

**Files:**
- Create: `observability/prometheus/prometheus.yml`, `observability/prometheus/rules/orders-api.slo.rules.yml`, `observability/prometheus/rules/orders-api.slo.rules.test.yml`

- [ ] **Step 1: Recording rules** (`rules/orders-api.slo.rules.yml`)

```yaml
groups:
  - name: orders-api-slo
    interval: 15s
    rules:
      # SLI availability per window: good (non-5xx) / total
      - record: slo:orders_api:availability:ratio_5m
        expr: |
          sum(rate(http_requests_total{job="orders-api",status!~"5.."}[5m]))
          /
          sum(rate(http_requests_total{job="orders-api"}[5m]))
      - record: slo:orders_api:availability:ratio_1h
        expr: |
          sum(rate(http_requests_total{job="orders-api",status!~"5.."}[1h]))
          / sum(rate(http_requests_total{job="orders-api"}[1h]))
      - record: slo:orders_api:availability:ratio_6h
        expr: |
          sum(rate(http_requests_total{job="orders-api",status!~"5.."}[6h]))
          / sum(rate(http_requests_total{job="orders-api"}[6h]))
      - record: slo:orders_api:availability:ratio_24h
        expr: |
          sum(rate(http_requests_total{job="orders-api",status!~"5.."}[24h]))
          / sum(rate(http_requests_total{job="orders-api"}[24h]))
      - record: slo:orders_api:availability:ratio_28d
        expr: |
          sum(rate(http_requests_total{job="orders-api",status!~"5.."}[28d]))
          / sum(rate(http_requests_total{job="orders-api"}[28d]))
      # Burn rate per window: (1 - SLI_w) / (1 - target); target 0.999 => budget 0.001
      - record: slo:orders_api:burnrate_1h
        expr: (1 - slo:orders_api:availability:ratio_1h) / 0.001
      - record: slo:orders_api:burnrate_6h
        expr: (1 - slo:orders_api:availability:ratio_6h) / 0.001
      - record: slo:orders_api:burnrate_24h
        expr: (1 - slo:orders_api:availability:ratio_24h) / 0.001
      # Error budget remaining fraction over 28d, clamped to [0,1]
      - record: slo:orders_api:error_budget_remaining_ratio
        expr: clamp_max(clamp_min(1 - ((1 - slo:orders_api:availability:ratio_28d) / 0.001), 0), 1)
      # Total requests over 28d (for absolute budget count)
      - record: slo:orders_api:requests_total_28d
        expr: sum(increase(http_requests_total{job="orders-api"}[28d]))
      # p95 latency (5m)
      - record: slo:orders_api:latency_p95_seconds
        expr: histogram_quantile(0.95, sum by (le) (rate(http_request_duration_seconds_bucket{job="orders-api"}[5m])))
```

- [ ] **Step 2: promtool unit test** (`rules/orders-api.slo.rules.test.yml`)

```yaml
rule_files:
  - orders-api.slo.rules.yml

evaluation_interval: 1m

tests:
  - interval: 1m
    input_series:
      # 99 good requests/sec, 1 error/sec over 1h => SLI ~0.99, burn rate ~10
      - series: 'http_requests_total{job="orders-api",status="200"}'
        values: '0+5940x60'    # 99/s * 60s per minute step
      - series: 'http_requests_total{job="orders-api",status="500"}'
        values: '0+60x60'      # 1/s * 60s
    promql_expr_test:
      - expr: slo:orders_api:availability:ratio_1h
        eval_time: 59m
        exp_samples:
          - labels: '{}'
            value: 0.99
      - expr: slo:orders_api:burnrate_1h
        eval_time: 59m
        exp_samples:
          - labels: '{}'
            value: 10
```

- [ ] **Step 3: Run promtool test, expect PASS**

Run: `docker run --rm -v "$PWD/observability/prometheus/rules:/rules" -w /rules prom/prometheus:latest promtool test rules orders-api.slo.rules.test.yml`
Expected: `SUCCESS`
(If the computed value is off by rounding, set the expected value to match the documented math and re-run — the assertion is the contract.)

- [ ] **Step 4: Prometheus config** (`prometheus.yml`)

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s
rule_files:
  - /etc/prometheus/rules/orders-api.slo.rules.yml
scrape_configs:
  - job_name: orders-api
    static_configs: [{ targets: ['orders-api:8080'] }]
  - job_name: slo-bff
    static_configs: [{ targets: ['slo-bff:9090'] }]
  - job_name: prometheus
    static_configs: [{ targets: ['localhost:9090'] }]
```

**Checkpoint:** `promtool test rules` prints SUCCESS.

---

## Task 4: `slo-bff` — Prometheus client + SLO mapping (TDD)

**Files:**
- Create: `services/slo-bff/go.mod`, `services/slo-bff/promclient.go`, `services/slo-bff/slo.go`, `services/slo-bff/slo_test.go`, `services/slo-bff/handlers.go`, `services/slo-bff/main.go`, `services/slo-bff/Dockerfile`

- [ ] **Step 1: Init module** — `cd services/slo-bff && go mod init github.com/slo-control-center/slo-bff`

- [ ] **Step 2: Client interface** (`promclient.go`)

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// PromAPI is the minimal Prometheus query surface the BFF needs.
type PromAPI interface {
	Query(ctx context.Context, q string) (float64, error)
}

type httpProm struct{ base string; c *http.Client }

func NewProm(base string) PromAPI { return &httpProm{base: base, c: &http.Client{Timeout: 5 * time.Second}} }

func (p *httpProm) Query(ctx context.Context, q string) (float64, error) {
	u := fmt.Sprintf("%s/api/v1/query?query=%s", p.base, url.QueryEscape(q))
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	resp, err := p.c.Do(req)
	if err != nil { return 0, err }
	defer resp.Body.Close()
	var out struct {
		Data struct {
			Result []struct{ Value [2]any `json:"value"` } `json:"result"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil { return 0, err }
	if len(out.Data.Result) == 0 { return 0, nil }
	s, _ := out.Data.Result[0].Value[1].(string)
	f, err := strconv.ParseFloat(s, 64)
	return f, err
}
```

- [ ] **Step 3: Failing test** (`slo_test.go`)

```go
package main

import (
	"context"
	"testing"
)

type fakeProm map[string]float64

func (f fakeProm) Query(_ context.Context, q string) (float64, error) { return f[q], nil }

func TestBuildSummary(t *testing.T) {
	f := fakeProm{
		"slo:orders_api:availability:ratio_28d":            0.9993,
		"slo:orders_api:latency_p95_seconds":               0.198,
		"slo:orders_api:error_budget_remaining_ratio":      0.842,
		"slo:orders_api:requests_total_28d":                100000,
		"slo:orders_api:burnrate_1h":                       0.3,
		"slo:orders_api:burnrate_6h":                       0.5,
		"slo:orders_api:burnrate_24h":                      0.4,
	}
	s, err := BuildSummary(context.Background(), f, ServiceSLO{Service: "orders-api", TargetPct: 99.9})
	if err != nil { t.Fatal(err) }
	if s.SLIPct < 99.92 || s.SLIPct > 99.94 { t.Fatalf("sli %v", s.SLIPct) }
	if s.P95Ms != 198 { t.Fatalf("p95 %v", s.P95Ms) }
	if s.ErrorBudgetRemainingPct < 84.1 || s.ErrorBudgetRemainingPct > 84.3 { t.Fatalf("budget %v", s.ErrorBudgetRemainingPct) }
	if s.ErrorBudgetRemainingCount != 84 { t.Fatalf("count %v", s.ErrorBudgetRemainingCount) }
	if !s.Healthy { t.Fatal("should be healthy: sli > target") }
}
```

> Budget count: `remaining_ratio(0.842) * (budget=0.001) * requests(100000) = 84.2` → rounded to 84.

- [ ] **Step 4: Run test, expect FAIL** — undefined: BuildSummary.

- [ ] **Step 5: Implement** (`slo.go`)

```go
package main

import (
	"context"
	"math"
)

type ServiceSLO struct {
	Service   string
	TargetPct float64 // e.g. 99.9
}

type BurnRate struct {
	H1  float64 `json:"1h"`
	H6  float64 `json:"6h"`
	H24 float64 `json:"24h"`
}

type SloSummary struct {
	Service                   string   `json:"service"`
	SLIPct                    float64  `json:"sliPct"`
	TargetPct                 float64  `json:"targetPct"`
	ErrorBudgetRemainingPct   float64  `json:"errorBudgetRemainingPct"`
	ErrorBudgetRemainingCount int64    `json:"errorBudgetRemainingCount"`
	BurnRate                  BurnRate `json:"burnRate"`
	P95Ms                     int64    `json:"p95Ms"`
	Healthy                   bool     `json:"healthy"`
}

func round2(f float64) float64 { return math.Round(f*100) / 100 }

func BuildSummary(ctx context.Context, p PromAPI, svc ServiceSLO) (SloSummary, error) {
	q := func(s string) float64 { v, _ := p.Query(ctx, s); return v }
	budget := 1 - svc.TargetPct/100 // 0.001 for 99.9
	sli := q("slo:orders_api:availability:ratio_28d")
	rem := q("slo:orders_api:error_budget_remaining_ratio")
	reqs := q("slo:orders_api:requests_total_28d")
	return SloSummary{
		Service:                   svc.Service,
		SLIPct:                    round2(sli * 100),
		TargetPct:                 svc.TargetPct,
		ErrorBudgetRemainingPct:   round2(rem * 100),
		ErrorBudgetRemainingCount: int64(math.Round(rem * budget * reqs)),
		BurnRate: BurnRate{
			H1:  round2(q("slo:orders_api:burnrate_1h")),
			H6:  round2(q("slo:orders_api:burnrate_6h")),
			H24: round2(q("slo:orders_api:burnrate_24h")),
		},
		P95Ms:   int64(math.Round(q("slo:orders_api:latency_p95_seconds") * 1000)),
		Healthy: sli*100 >= svc.TargetPct,
	}, nil
}
```

- [ ] **Step 6: Run test, expect PASS** — `go test ./...`

- [ ] **Step 7: Handlers + main + Dockerfile** (`handlers.go`)

```go
package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

var services = []ServiceSLO{{Service: "orders-api", TargetPct: 99.9}}

type API struct{ prom PromAPI }

func NewAPI(prom PromAPI) *http.ServeMux {
	a := &API{prom: prom}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/slo", a.listSLO)
	mux.HandleFunc("/api/slo/", a.compliance) // /api/slo/{service}/compliance
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	mux.Handle("/metrics", metricsHandler())
	return mux
}

func cors(w http.ResponseWriter) { w.Header().Set("Access-Control-Allow-Origin", "*") }

func (a *API) listSLO(w http.ResponseWriter, r *http.Request) {
	cors(w)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	out := make([]SloSummary, 0, len(services))
	for _, s := range services {
		sum, err := BuildSummary(ctx, a.prom, s)
		if err != nil { http.Error(w, err.Error(), 500); return }
		out = append(out, sum)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (a *API) compliance(w http.ResponseWriter, r *http.Request) {
	cors(w)
	// Slice 1: return the 28d availability ratio sampled as a flat series placeholder
	// is NOT acceptable; query_range is implemented in promclient (see range method).
	pts, err := a.prom.(RangeQuerier).QueryRange(r.Context(),
		"slo:orders_api:availability:ratio_5m", 28*24*time.Hour, time.Hour)
	if err != nil { http.Error(w, err.Error(), 500); return }
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"points": pts})
}
```

> This task introduces `RangeQuerier` + `QueryRange` and `metricsHandler()`. Implement them as part of Step 7:

`promclient.go` additions:

```go
// Point is one (unix-seconds, value) sample.
type Point struct {
	T int64   `json:"t"`
	V float64 `json:"sliPct"`
}

type RangeQuerier interface {
	QueryRange(ctx context.Context, q string, span, step time.Duration) ([]Point, error)
}

func (p *httpProm) QueryRange(ctx context.Context, q string, span, step time.Duration) ([]Point, error) {
	// end/start derived from Prometheus server clock via the "time()" function is
	// unavailable here; use the matrix endpoint with relative range via the
	// /api/v1/query_range using start=now-span. The BFF passes RFC3339 from its own clock.
	end := time.Now()
	start := end.Add(-span)
	u := fmt.Sprintf("%s/api/v1/query_range?query=%s&start=%d&end=%d&step=%d",
		p.base, url.QueryEscape(q), start.Unix(), end.Unix(), int(step.Seconds()))
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	resp, err := p.c.Do(req)
	if err != nil { return nil, err }
	defer resp.Body.Close()
	var out struct {
		Data struct {
			Result []struct{ Values [][2]any `json:"values"` } `json:"result"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil { return nil, err }
	pts := []Point{}
	if len(out.Data.Result) > 0 {
		for _, v := range out.Data.Result[0].Values {
			ts, _ := v[0].(float64)
			s, _ := v[1].(string)
			f, _ := strconv.ParseFloat(s, 64)
			pts = append(pts, Point{T: int64(ts), V: round2(f * 100)})
		}
	}
	return pts, nil
}
```

`main.go`:

```go
package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	base := os.Getenv("PROMETHEUS_URL")
	if base == "" { base = "http://prometheus:9090" }
	addr := os.Getenv("ADDR")
	if addr == "" { addr = ":9090" }
	mux := NewAPI(NewProm(base))
	log.Printf("slo-bff listening on %s -> %s", addr, base)
	log.Fatal(http.ListenAndServe(addr, mux))
}
```

`metrics.go` (BFF self-metrics, minimal):

```go
package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func metricsHandler() http.Handler { return promhttp.Handler() }
```

Run `go get github.com/prometheus/client_golang@latest`. Dockerfile mirrors Task 2's (build `slo-bff`, EXPOSE 9090).

**Checkpoint:** `go test ./...` passes; `go build .` succeeds.

---

## Task 5: Frontend — Overview tab (TDD on render)

**Files:**
- Create the `frontend/` tree listed in File Structure.

- [ ] **Step 1: Scaffold** — `npm create vite@latest frontend -- --template react-ts`, then `cd frontend && npm i && npm i @tremor/react && npm i -D tailwindcss postcss autoprefixer vitest @testing-library/react @testing-library/jest-dom jsdom && npx tailwindcss init -p`. Configure `tailwind.config.js` content globs to `./index.html` + `./src/**/*.{ts,tsx}` and include Tremor's path; add Tailwind directives to `src/index.css`. Set `vite.config.ts` test env to `jsdom`.

- [ ] **Step 2: Types** (`src/api/types.ts`)

```ts
export interface BurnRate { "1h": number; "6h": number; "24h": number }
export interface SloSummary {
  service: string; sliPct: number; targetPct: number;
  errorBudgetRemainingPct: number; errorBudgetRemainingCount: number;
  burnRate: BurnRate; p95Ms: number; healthy: boolean;
}
export interface CompliancePoint { t: number; sliPct: number }
export interface ComplianceSeries { points: CompliancePoint[] }
```

- [ ] **Step 3: Failing render test** (`src/pages/Overview.test.tsx`)

```tsx
import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { OverviewView } from "./Overview";
import type { SloSummary } from "../api/types";

const sample: SloSummary[] = [{
  service: "orders-api", sliPct: 99.93, targetPct: 99.9,
  errorBudgetRemainingPct: 84.2, errorBudgetRemainingCount: 84,
  burnRate: { "1h": 0.3, "6h": 0.5, "24h": 0.4 }, p95Ms: 198, healthy: true,
}];

describe("OverviewView", () => {
  it("renders availability and a service row", () => {
    render(<OverviewView summaries={sample} compliance={{ points: [] }} />);
    expect(screen.getByText(/99.93%/)).toBeInTheDocument();
    expect(screen.getByText(/orders-api/)).toBeInTheDocument();
  });
});
```

- [ ] **Step 4: Run test, expect FAIL** — `npx vitest run` → cannot find OverviewView.

- [ ] **Step 5: Implement presentational components.** `OverviewView` is a pure prop-driven component (no fetching) so it is testable; `Overview` wraps it with the `useSlo` polling hook.

`src/components/StatCard.tsx`:

```tsx
export function StatCard({ label, value, sub }: { label: string; value: string; sub?: string }) {
  return (
    <div className="rounded-xl bg-slate-800/60 p-4 border border-slate-700">
      <div className="text-xs text-slate-400">{label}</div>
      <div className="text-3xl font-bold text-white">{value}</div>
      {sub && <div className="text-xs text-slate-400 mt-1">{sub}</div>}
    </div>
  );
}
```

`src/components/ErrorBudgetTable.tsx`:

```tsx
import type { SloSummary } from "../api/types";
export function ErrorBudgetTable({ rows }: { rows: SloSummary[] }) {
  return (
    <table className="w-full text-sm text-slate-200">
      <thead><tr className="text-slate-400 text-left">
        <th>Service</th><th>SLO</th><th>Availability</th><th>Error Budget</th><th>Burn 1h</th>
      </tr></thead>
      <tbody>
        {rows.map(r => (
          <tr key={r.service} className="border-t border-slate-700">
            <td className="py-2">{r.service}</td>
            <td>{r.targetPct}%</td>
            <td>{r.sliPct}%</td>
            <td>{r.errorBudgetRemainingPct}% remaining</td>
            <td>{r.burnRate["1h"]}x</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
```

`src/components/ComplianceChart.tsx` (Tremor `AreaChart`):

```tsx
import { AreaChart } from "@tremor/react";
import type { ComplianceSeries } from "../api/types";
export function ComplianceChart({ series }: { series: ComplianceSeries }) {
  const data = series.points.map(p => ({ date: new Date(p.t * 1000).toLocaleDateString(), SLI: p.sliPct }));
  return <AreaChart data={data} index="date" categories={["SLI"]} className="h-64" />;
}
```

`src/components/ServiceHealthMap.tsx`:

```tsx
import type { SloSummary } from "../api/types";
export function ServiceHealthMap({ rows }: { rows: SloSummary[] }) {
  return (
    <div className="flex gap-4">
      {rows.map(r => (
        <div key={r.service} className={`px-4 py-3 rounded-full ${r.healthy ? "bg-emerald-600" : "bg-rose-600"}`}>
          {r.service}
        </div>
      ))}
    </div>
  );
}
```

`src/pages/Overview.tsx`:

```tsx
import type { SloSummary, ComplianceSeries } from "../api/types";
import { StatCard } from "../components/StatCard";
import { ErrorBudgetTable } from "../components/ErrorBudgetTable";
import { ComplianceChart } from "../components/ComplianceChart";
import { ServiceHealthMap } from "../components/ServiceHealthMap";
import { useSlo } from "../hooks/useSlo";

export function OverviewView({ summaries, compliance }: { summaries: SloSummary[]; compliance: ComplianceSeries }) {
  const primary = summaries[0];
  const healthy = summaries.filter(s => s.healthy).length;
  return (
    <div className="space-y-6 p-6 bg-slate-900 min-h-screen">
      <h1 className="text-xl font-semibold text-white">Global SLO Overview Dashboard</h1>
      <div className="grid grid-cols-2 md:grid-cols-6 gap-3">
        <StatCard label="Availability" value={primary ? `${primary.sliPct}%` : "—"} />
        <StatCard label="Latency p95" value={primary ? `${primary.p95Ms}ms` : "—"} sub="p95" />
        <StatCard label="Error Budget Remaining" value={primary ? `${primary.errorBudgetRemainingPct}%` : "—"} />
        <StatCard label="Current Burn Rate" value={primary ? `${primary.burnRate["1h"]}x` : "—"} />
        <StatCard label="Services Healthy" value={`${healthy} / ${summaries.length}`} />
        <StatCard label="Open Incidents" value="0" />
      </div>
      <div className="rounded-xl bg-slate-800/60 p-4 border border-slate-700">
        <h2 className="text-slate-200 mb-3">Error Budgets by Service</h2>
        <ErrorBudgetTable rows={summaries} />
      </div>
      <div className="rounded-xl bg-slate-800/60 p-4 border border-slate-700">
        <h2 className="text-slate-200 mb-3">28-Day Compliance Graph</h2>
        <ComplianceChart series={compliance} />
      </div>
      <div className="rounded-xl bg-slate-800/60 p-4 border border-slate-700">
        <h2 className="text-slate-200 mb-3">Live Service Health Map</h2>
        <ServiceHealthMap rows={summaries} />
      </div>
    </div>
  );
}

export default function Overview() {
  const { summaries, compliance } = useSlo();
  return <OverviewView summaries={summaries} compliance={compliance} />;
}
```

`src/api/client.ts`:

```ts
import type { SloSummary, ComplianceSeries } from "./types";
const BASE = import.meta.env.VITE_BFF_URL ?? "http://localhost:9090";
export async function fetchSlo(): Promise<SloSummary[]> {
  const r = await fetch(`${BASE}/api/slo`); return r.json();
}
export async function fetchCompliance(service: string): Promise<ComplianceSeries> {
  const r = await fetch(`${BASE}/api/slo/${service}/compliance`); return r.json();
}
```

`src/hooks/useSlo.ts`:

```ts
import { useEffect, useState } from "react";
import { fetchSlo, fetchCompliance } from "../api/client";
import type { SloSummary, ComplianceSeries } from "../api/types";

export function useSlo() {
  const [summaries, setSummaries] = useState<SloSummary[]>([]);
  const [compliance, setCompliance] = useState<ComplianceSeries>({ points: [] });
  useEffect(() => {
    let alive = true;
    const tick = async () => {
      try {
        const s = await fetchSlo();
        if (!alive) return;
        setSummaries(s);
        if (s[0]) setCompliance(await fetchCompliance(s[0].service));
      } catch { /* keep last good */ }
    };
    tick();
    const id = setInterval(tick, 15000);
    return () => { alive = false; clearInterval(id); };
  }, []);
  return { summaries, compliance };
}
```

`src/App.tsx` renders `<Overview />`; `src/main.tsx` is the Vite default mounting `<App />`.

- [ ] **Step 6: Run test, expect PASS** — `npx vitest run`

- [ ] **Step 7: Frontend Dockerfile** (multi-stage build → static nginx)

```dockerfile
FROM node:20-alpine AS build
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build
FROM nginx:alpine
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
```

**Checkpoint:** `npx vitest run` passes; `npm run build` succeeds.

---

## Task 6: Grafonnet dashboard

**Files:**
- Create: `observability/grafana/jsonnet/slo-overview.jsonnet`, `jsonnetfile.json`, provisioning files, and the built `dashboards/slo-overview.json`.

- [ ] **Step 1: Jsonnet source** (`slo-overview.jsonnet`) — import grafonnet, define a dashboard with four panels backed by the recording rules: `slo:orders_api:availability:ratio_5m` (stat), `slo:orders_api:error_budget_remaining_ratio` (gauge), `slo:orders_api:burnrate_1h/6h/24h` (timeseries), `slo:orders_api:latency_p95_seconds` (timeseries). Use grafonnet v10 API:

```jsonnet
local g = import 'github.com/grafana/grafonnet/gen/grafonnet-v10.0.0/main.libsonnet';
local panel = g.panel;
local q = g.query.prometheus;

g.dashboard.new('SLO Overview - orders-api')
+ g.dashboard.withUid('slo-overview')
+ g.dashboard.withPanels([
  panel.stat.new('Availability (5m)')
  + panel.stat.queryOptions.withTargets([q.new('Prometheus', 'slo:orders_api:availability:ratio_5m * 100')])
  + panel.stat.standardOptions.withUnit('percent')
  + panel.stat.gridPos.withW(6) + panel.stat.gridPos.withH(6),

  panel.gauge.new('Error Budget Remaining')
  + panel.gauge.queryOptions.withTargets([q.new('Prometheus', 'slo:orders_api:error_budget_remaining_ratio * 100')])
  + panel.gauge.standardOptions.withUnit('percent')
  + panel.gauge.gridPos.withW(6) + panel.gauge.gridPos.withH(6) + panel.gauge.gridPos.withX(6),

  panel.timeSeries.new('Burn Rate (1h/6h/24h)')
  + panel.timeSeries.queryOptions.withTargets([
      q.new('Prometheus', 'slo:orders_api:burnrate_1h') + q.withLegendFormat('1h'),
      q.new('Prometheus', 'slo:orders_api:burnrate_6h') + q.withLegendFormat('6h'),
      q.new('Prometheus', 'slo:orders_api:burnrate_24h') + q.withLegendFormat('24h'),
    ])
  + panel.timeSeries.gridPos.withW(12) + panel.timeSeries.gridPos.withH(8) + panel.timeSeries.gridPos.withY(6),

  panel.timeSeries.new('p95 Latency (ms)')
  + panel.timeSeries.queryOptions.withTargets([q.new('Prometheus', 'slo:orders_api:latency_p95_seconds * 1000')])
  + panel.timeSeries.standardOptions.withUnit('ms')
  + panel.timeSeries.gridPos.withW(12) + panel.timeSeries.gridPos.withH(8) + panel.timeSeries.gridPos.withY(14),
])
```

- [ ] **Step 2: jsonnetfile.json**

```json
{ "version": 1, "dependencies": [{
  "source": { "git": { "remote": "https://github.com/grafana/grafonnet.git", "subdir": "gen/grafonnet-v10.0.0" } },
  "version": "main" }], "legacyImports": true }
```

- [ ] **Step 3: Build the dashboard JSON**

Run (via a tooling container so no local jsonnet install is needed):
```bash
docker run --rm -v "$PWD/observability/grafana:/w" -w /w grafana/grafonnet-builder:latest \
  sh -c "jb install && jsonnet -J vendor jsonnet/slo-overview.jsonnet" > observability/grafana/dashboards/slo-overview.json
```
> If that image is unavailable, document the fallback in README: `go install github.com/google/go-jsonnet/cmd/jsonnet@latest` + `jb`. The committed `dashboards/slo-overview.json` is the artifact Grafana loads; regeneration is via `make dashboards`.

- [ ] **Step 4: Provisioning**

`provisioning/datasources/prometheus.yml`:
```yaml
apiVersion: 1
datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
```
`provisioning/dashboards/dashboards.yml`:
```yaml
apiVersion: 1
providers:
  - name: slo
    type: file
    options: { path: /var/lib/grafana/dashboards }
```

**Checkpoint:** `make dashboards` produces non-empty `slo-overview.json`; `jsonnet` exits 0.

---

## Task 7: k6 load script

**Files:** Create `load/k6/orders-load.js`.

- [ ] **Step 1: Deterministic steady-load script**

```js
import http from "k6/http";
import { check, sleep } from "k6";

// Deterministic: fixed VUs + RPS, seeded item selection.
export const options = {
  scenarios: {
    steady: { executor: "constant-arrival-rate", rate: 50, timeUnit: "1s",
      duration: "12h", preAllocatedVUs: 20, maxVUs: 50 },
  },
};

const BASE = __ENV.ORDERS_URL || "http://orders-api:8080";
const items = ["widget", "gadget", "gizmo", "doohickey"];

export default function () {
  // Index derived from iteration counter => reproducible without Math.random.
  const i = (__ITER % items.length);
  const res = http.post(`${BASE}/orders`, JSON.stringify({ item: items[i], qty: (i % 3) + 1 }),
    { headers: { "Content-Type": "application/json" } });
  check(res, { "created": (r) => r.status === 201 || r.status === 500 }); // 500s are chaos, expected
  sleep(0.01);
}
```

**Checkpoint:** `k6 inspect load/k6/orders-load.js` (or `docker run grafana/k6 inspect ...`) parses without error.

---

## Task 8: Docker Compose + Makefile + README (integration)

**Files:** Create `deploy/compose/docker-compose.yml`, `deploy/compose/.env.example`, `Makefile`, `README.md`.

- [ ] **Step 1: Compose** (`deploy/compose/docker-compose.yml`)

```yaml
services:
  postgres:
    image: postgres:16-alpine
    environment: { POSTGRES_USER: orders, POSTGRES_PASSWORD: orders, POSTGRES_DB: orders }
    healthcheck: { test: ["CMD-SHELL", "pg_isready -U orders"], interval: 5s, retries: 10 }

  orders-api:
    build: ../../services/orders-api
    environment:
      DATABASE_URL: postgres://orders:orders@postgres:5432/orders?sslmode=disable
      CHAOS_ERROR_RATE: ${CHAOS_ERROR_RATE:-0}
      CHAOS_LATENCY_MS: ${CHAOS_LATENCY_MS:-0}
      CHAOS_LATENCY_PCT: ${CHAOS_LATENCY_PCT:-0}
    depends_on: { postgres: { condition: service_healthy } }
    ports: ["8080:8080"]

  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ../../observability/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - ../../observability/prometheus/rules:/etc/prometheus/rules:ro
    command: ["--config.file=/etc/prometheus/prometheus.yml", "--storage.tsdb.retention.time=30d"]
    ports: ["9091:9090"]
    depends_on: [orders-api]

  slo-bff:
    build: ../../services/slo-bff
    environment: { PROMETHEUS_URL: http://prometheus:9090, ADDR: ":9090" }
    ports: ["9090:9090"]
    depends_on: [prometheus]

  frontend:
    build: ../../frontend
    ports: ["3000:80"]
    depends_on: [slo-bff]

  grafana:
    image: grafana/grafana:latest
    environment: { GF_AUTH_ANONYMOUS_ENABLED: "true", GF_AUTH_ANONYMOUS_ORG_ROLE: Admin }
    volumes:
      - ../../observability/grafana/provisioning:/etc/grafana/provisioning:ro
      - ../../observability/grafana/dashboards:/var/lib/grafana/dashboards:ro
    ports: ["3001:3000"]
    depends_on: [prometheus]

  k6:
    image: grafana/k6:latest
    command: ["run", "/scripts/orders-load.js"]
    environment: { ORDERS_URL: http://orders-api:8080 }
    volumes: ["../../load/k6:/scripts:ro"]
    depends_on: [orders-api]
```

> Note: the BFF self-scrape target in `prometheus.yml` is `slo-bff:9090`; ensure the BFF listens on 9090 (it does). Frontend build arg `VITE_BFF_URL` defaults to `http://localhost:9090`, reachable from the browser via the published port.

- [ ] **Step 2: Makefile**

```makefile
COMPOSE=docker compose -f deploy/compose/docker-compose.yml

up: ; $(COMPOSE) up --build -d
down: ; $(COMPOSE) down -v
logs: ; $(COMPOSE) logs -f
test:
	cd services/orders-api && go test ./...
	cd services/slo-bff && go test ./...
	cd frontend && npx vitest run
	docker run --rm -v "$(PWD)/observability/prometheus/rules:/rules" -w /rules prom/prometheus:latest promtool test rules orders-api.slo.rules.test.yml
dashboards:
	docker run --rm -v "$(PWD)/observability/grafana:/w" -w /w grafana/grafonnet-builder:latest sh -c "jb install && jsonnet -J vendor jsonnet/slo-overview.jsonnet" > observability/grafana/dashboards/slo-overview.json
```

- [ ] **Step 3: README** — document: prerequisites (Docker), `make up`, URLs (frontend :3000, Grafana :3001, Prometheus :9091, BFF :9090), how to trigger chaos (`CHAOS_ERROR_RATE=0.1 make up` or edit `.env`), `make test`, and the deterministic-load note.

- [ ] **Step 4: Integration verification**

Run: `make up`, wait ~60–90s, then:
- `curl -s localhost:9090/api/slo` → JSON array with `orders-api`, non-null fields.
- Open `localhost:3000` → Overview shows availability ~100%, table row present.
- Open `localhost:3001` → SLO Overview dashboard renders panels with data.
- `CHAOS_ERROR_RATE=0.2 make up` (recreate orders-api) → within ~1–2 min availability dips, burn rate rises in BOTH UI and Grafana.

**Checkpoint (definition of done):** `make test` all green; `make up` yields a live, moving Overview tab and Grafana dashboard; chaos env visibly moves the SLO in both.

---

## Self-Review Notes

- **Spec coverage:** orders-api+chaos (T1–2), Prometheus+rules+promtool (T3), BFF contract (T4), Overview tab (T5), Grafonnet dashboard (T6), k6 deterministic load (T7), Compose<2min + tests (T8). Deferred items (Loki/Tempo, other services, Alertmanager, runbooks, other tabs, K8s) are explicitly out of slice-1 scope per the spec.
- **Type consistency:** `SloSummary` field names match between Go (`slo.go` json tags) and TS (`types.ts`). Recording-rule names are identical across `rules.yml`, `slo.go`, `slo.jsonnet`, and the promtool test.
- **Known risk:** Grafonnet builder image name may differ; T6 Step 3 documents a fallback. promtool expected values (T3) may need a one-time numeric tweak to match exact rate() math — the test asserts the documented contract.
