package main

import "context"

// ServiceDetail is the per-service view for the Services tab: SLO + RED + deps.
type ServiceDetail struct {
	Service      string   `json:"service"`
	SLIPct       float64  `json:"sliPct"`
	TargetPct    float64  `json:"targetPct"`
	Healthy      bool     `json:"healthy"`
	RatePerSec   float64  `json:"ratePerSec"`
	ErrorPct     float64  `json:"errorPct"`
	P95Ms        int64    `json:"p95Ms"`
	Dependencies []string `json:"dependencies"`
}

func buildServiceDetail(ctx context.Context, p Prom, svc ServiceSLO) (ServiceDetail, error) {
	sum, err := BuildSummary(ctx, p, svc)
	if err != nil {
		return ServiceDetail{}, err
	}
	rate := 0.0
	if svc.RateQuery != "" {
		rate, _ = p.Query(ctx, svc.RateQuery)
	}
	return ServiceDetail{
		Service:      svc.Service,
		SLIPct:       sum.SLIPct,
		TargetPct:    sum.TargetPct,
		Healthy:      sum.Healthy,
		RatePerSec:   round2(rate),
		ErrorPct:     round2(100 - sum.SLIPct),
		P95Ms:        sum.P95Ms,
		Dependencies: svc.Dependencies,
	}, nil
}
