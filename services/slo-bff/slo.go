package main

import (
	"context"
	"math"
)

// ServiceSLO names a service, its target (e.g. 99.9), and the recording-rule
// names that back it. Prefix is the common rule namespace (e.g.
// "slo:payments_worker"); SLIRule is the full 28d SLI ratio rule name, whose
// middle segment differs per service ("availability" vs "good").
type ServiceSLO struct {
	Service      string
	TargetPct    float64
	Prefix       string
	SLIRule      string
	RateQuery    string   // PromQL for current request rate (per second)
	Dependencies []string // downstream services this one depends on
}

// BurnRate holds the multi-window error-budget burn rates.
type BurnRate struct {
	H1  float64 `json:"1h"`
	H6  float64 `json:"6h"`
	H24 float64 `json:"24h"`
}

// SloSummary is the JSON contract the frontend consumes.
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

// BuildSummary queries the recording rules for one service and packages the contract.
func BuildSummary(ctx context.Context, p Prom, svc ServiceSLO) (SloSummary, error) {
	q := func(s string) float64 { v, _ := p.Query(ctx, s); return v }
	budget := 1 - svc.TargetPct/100 // 0.001 for 99.9, 0.005 for 99.5, etc.
	sli := q(svc.SLIRule)
	rem := q(svc.Prefix + ":error_budget_remaining_ratio")
	reqs := q(svc.Prefix + ":requests_total_28d")
	return SloSummary{
		Service:                   svc.Service,
		SLIPct:                    round2(sli * 100),
		TargetPct:                 svc.TargetPct,
		ErrorBudgetRemainingPct:   round2(rem * 100),
		ErrorBudgetRemainingCount: int64(math.Round(rem * budget * reqs)),
		BurnRate: BurnRate{
			H1:  round2(q(svc.Prefix + ":burnrate_1h")),
			H6:  round2(q(svc.Prefix + ":burnrate_6h")),
			H24: round2(q(svc.Prefix + ":burnrate_24h")),
		},
		P95Ms:   int64(math.Round(q(svc.Prefix+":latency_p95_seconds") * 1000)),
		Healthy: sli*100 >= svc.TargetPct,
	}, nil
}
