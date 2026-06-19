package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var services = []ServiceSLO{
	{Service: "orders-api", TargetPct: 99.9, Prefix: "slo:orders_api", SLIRule: "slo:orders_api:availability:ratio_28d",
		RateQuery: `sum(rate(http_requests_total{job="orders-api"}[5m]))`, Dependencies: []string{"postgres", "redis"}},
	{Service: "payments-worker", TargetPct: 99.5, Prefix: "slo:payments_worker", SLIRule: "slo:payments_worker:good:ratio_28d",
		RateQuery: `sum(rate(payments_processed_total{job="payments-worker"}[5m]))`, Dependencies: []string{"redis", "mock-gateway"}},
	{Service: "notification-svc", TargetPct: 99.0, Prefix: "slo:notification_svc", SLIRule: "slo:notification_svc:good:ratio_28d",
		RateQuery: `sum(rate(notifications_delivered_total{job="notification-svc"}[5m]))`, Dependencies: []string{"redis", "mock-receiver"}},
}

// API holds the backend clients and serves the frontend contract.
type API struct {
	prom        Prom
	tempo       Tempo
	alerts      AlertSource
	logs        LogSource
	runbooksDir string
}

// NewAPI builds the BFF route table.
func NewAPI(prom Prom, tempo Tempo, alerts AlertSource, logs LogSource, runbooksDir string) *http.ServeMux {
	a := &API{prom: prom, tempo: tempo, alerts: alerts, logs: logs, runbooksDir: runbooksDir}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/slo", a.listSLO)
	mux.HandleFunc("/api/slo/", a.compliance) // /api/slo/{service}/compliance
	mux.HandleFunc("/api/traces/recent", a.recentTraces)
	mux.HandleFunc("/api/services", a.listServices)
	mux.HandleFunc("/api/alerts", a.listAlerts)
	mux.HandleFunc("/api/incidents", a.listIncidents)
	mux.HandleFunc("/api/logs", a.listLogs)
	mux.HandleFunc("/api/capacity", a.listCapacity)
	mux.HandleFunc("/api/runbooks", a.listRunbooks)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	mux.Handle("/metrics", promhttp.Handler())
	return mux
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (a *API) recentTraces(w http.ResponseWriter, r *http.Request) {
	cors(w)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	var refs []TraceRef
	var err error
	if r.URL.Query().Get("status") == "any" {
		refs, err = a.tempo.SearchRecent(ctx)
	} else {
		refs, err = a.tempo.SearchErrors(ctx)
	}
	if err != nil {
		refs = []TraceRef{}
	}
	writeJSON(w, refs)
}

func (a *API) listServices(w http.ResponseWriter, r *http.Request) {
	cors(w)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	out := make([]ServiceDetail, 0, len(services))
	for _, s := range services {
		d, err := buildServiceDetail(ctx, a.prom, s)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out = append(out, d)
	}
	writeJSON(w, out)
}

func (a *API) listAlerts(w http.ResponseWriter, r *http.Request) {
	cors(w)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	body, err := a.alerts.FetchAlerts(ctx)
	if err != nil {
		writeJSON(w, []Alert{})
		return
	}
	writeJSON(w, parseAlerts(body))
}

func (a *API) listIncidents(w http.ResponseWriter, r *http.Request) {
	cors(w)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	body, err := a.alerts.FetchAlerts(ctx)
	if err != nil {
		writeJSON(w, []Incident{})
		return
	}
	writeJSON(w, deriveIncidents(parseAlerts(body)))
}

func (a *API) listLogs(w http.ResponseWriter, r *http.Request) {
	cors(w)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	level := r.URL.Query().Get("level")
	logql := `{container=~"compose-.+"}`
	if level != "" {
		logql = `{level="` + level + `"}`
	}
	body, err := a.logs.Query(ctx, logql, 100)
	if err != nil {
		writeJSON(w, []LogLine{})
		return
	}
	writeJSON(w, parseLokiQuery(body))
}

func (a *API) listCapacity(w http.ResponseWriter, r *http.Request) {
	cors(w)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	writeJSON(w, buildCapacity(ctx, a.prom))
}

func (a *API) listRunbooks(w http.ResponseWriter, _ *http.Request) {
	cors(w)
	writeJSON(w, readRunbooks(a.runbooksDir))
}

func cors(w http.ResponseWriter) { w.Header().Set("Access-Control-Allow-Origin", "*") }

func (a *API) listSLO(w http.ResponseWriter, r *http.Request) {
	cors(w)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	out := make([]SloSummary, 0, len(services))
	for _, s := range services {
		sum, err := BuildSummary(ctx, a.prom, s)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out = append(out, sum)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (a *API) compliance(w http.ResponseWriter, r *http.Request) {
	cors(w)
	// Path is /api/slo/{service}/compliance; service is currently informational
	// (slice 1 tracks orders-api only) but parsed for forward-compatibility.
	_ = strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/slo/"), "/compliance")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	pts, err := a.prom.QueryRange(ctx, "slo:orders_api:availability:ratio_5m", 28*24*time.Hour, time.Hour)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"points": pts})
}
