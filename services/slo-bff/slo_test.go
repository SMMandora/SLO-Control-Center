package main

import (
	"context"
	"testing"
	"time"
)

type fakeProm map[string]float64

func (f fakeProm) Query(_ context.Context, q string) (float64, error) { return f[q], nil }
func (f fakeProm) QueryRange(_ context.Context, _ string, _, _ time.Duration) ([]Point, error) {
	return []Point{}, nil
}
func (f fakeProm) QueryVector(_ context.Context, _ string) ([]Sample, error) { return nil, nil }

func TestBuildSummary(t *testing.T) {
	f := fakeProm{
		"slo:orders_api:availability:ratio_28d":       0.9993,
		"slo:orders_api:latency_p95_seconds":          0.198,
		"slo:orders_api:error_budget_remaining_ratio": 0.842,
		"slo:orders_api:requests_total_28d":           100000,
		"slo:orders_api:burnrate_1h":                  0.3,
		"slo:orders_api:burnrate_6h":                  0.5,
		"slo:orders_api:burnrate_24h":                 0.4,
	}
	s, err := BuildSummary(context.Background(), f, ServiceSLO{
		Service: "orders-api", TargetPct: 99.9,
		Prefix: "slo:orders_api", SLIRule: "slo:orders_api:availability:ratio_28d",
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.SLIPct < 99.92 || s.SLIPct > 99.94 {
		t.Fatalf("sli %v", s.SLIPct)
	}
	if s.P95Ms != 198 {
		t.Fatalf("p95 %v", s.P95Ms)
	}
	if s.ErrorBudgetRemainingPct < 84.1 || s.ErrorBudgetRemainingPct > 84.3 {
		t.Fatalf("budget %v", s.ErrorBudgetRemainingPct)
	}
	// remaining(0.842) * budget(0.001) * requests(100000) = 84.2 -> 84
	if s.ErrorBudgetRemainingCount != 84 {
		t.Fatalf("count %v", s.ErrorBudgetRemainingCount)
	}
	if !s.Healthy {
		t.Fatal("should be healthy: sli > target")
	}
}

// Verifies BuildSummary uses the per-service prefix/SLIRule rather than
// hardcoded orders-api rule names.
func TestBuildSummaryPaymentsPrefix(t *testing.T) {
	f := fakeProm{
		"slo:payments_worker:good:ratio_28d":               0.994, // below 99.5 target
		"slo:payments_worker:error_budget_remaining_ratio": 0.2,
		"slo:payments_worker:requests_total_28d":           50000,
		"slo:payments_worker:latency_p95_seconds":          1.5,
		"slo:payments_worker:burnrate_1h":                  2.0,
	}
	s, err := BuildSummary(context.Background(), f, ServiceSLO{
		Service: "payments-worker", TargetPct: 99.5,
		Prefix: "slo:payments_worker", SLIRule: "slo:payments_worker:good:ratio_28d",
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.SLIPct != 99.4 {
		t.Fatalf("sli %v", s.SLIPct)
	}
	if s.P95Ms != 1500 {
		t.Fatalf("p95 %v", s.P95Ms)
	}
	if s.BurnRate.H1 != 2.0 {
		t.Fatalf("burn %v", s.BurnRate.H1)
	}
	if s.Healthy {
		t.Fatal("99.4 < 99.5 target must be unhealthy")
	}
}
