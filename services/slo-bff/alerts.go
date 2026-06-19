package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"
)

// Alert is a flattened active alert from Prometheus.
type Alert struct {
	Alertname  string `json:"alertname"`
	Severity   string `json:"severity"`
	Service    string `json:"service"`
	State      string `json:"state"`
	ActiveAt   string `json:"activeAt"`
	Summary    string `json:"summary"`
	RunbookURL string `json:"runbookUrl"`
}

// Incident is a derived grouping of firing alerts.
type Incident struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Service    string `json:"service"`
	Severity   string `json:"severity"`
	StartedAt  string `json:"startedAt"`
	AlertCount int    `json:"alertCount"`
}

// AlertSource fetches the raw Prometheus alerts payload.
type AlertSource interface {
	FetchAlerts(ctx context.Context) ([]byte, error)
}

type httpAlertSource struct {
	base string
	c    *http.Client
}

func NewAlertSource(promBase string) AlertSource {
	return &httpAlertSource{base: promBase, c: &http.Client{Timeout: 5 * time.Second}}
}

func (a *httpAlertSource) FetchAlerts(ctx context.Context) ([]byte, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, a.base+"/api/v1/alerts", nil)
	resp, err := a.c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// parseAlerts maps a Prometheus /api/v1/alerts body into Alerts.
func parseAlerts(body []byte) []Alert {
	var p struct {
		Data struct {
			Alerts []struct {
				Labels      map[string]string `json:"labels"`
				Annotations map[string]string `json:"annotations"`
				State       string            `json:"state"`
				ActiveAt    string            `json:"activeAt"`
			} `json:"alerts"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &p); err != nil {
		return nil
	}
	out := make([]Alert, 0, len(p.Data.Alerts))
	for _, a := range p.Data.Alerts {
		out = append(out, Alert{
			Alertname:  a.Labels["alertname"],
			Severity:   a.Labels["severity"],
			Service:    firstNonEmpty(a.Labels["service"], a.Labels["job"]),
			State:      a.State,
			ActiveAt:   a.ActiveAt,
			Summary:    a.Annotations["summary"],
			RunbookURL: a.Annotations["runbook_url"],
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ActiveAt > out[j].ActiveAt })
	return out
}

// deriveIncidents groups firing alerts by (alertname, service) into incidents.
func deriveIncidents(alerts []Alert) []Incident {
	type key struct{ name, svc string }
	groups := map[key]*Incident{}
	order := []key{}
	for _, a := range alerts {
		if a.State != "firing" {
			continue
		}
		k := key{a.Alertname, a.Service}
		inc, ok := groups[k]
		if !ok {
			inc = &Incident{
				ID:        fmt.Sprintf("%s-%s", a.Alertname, a.Service),
				Title:     firstNonEmpty(a.Summary, a.Alertname),
				Service:   a.Service,
				Severity:  a.Severity,
				StartedAt: a.ActiveAt,
			}
			groups[k] = inc
			order = append(order, k)
		}
		inc.AlertCount++
		if a.ActiveAt < inc.StartedAt {
			inc.StartedAt = a.ActiveAt
		}
	}
	out := make([]Incident, 0, len(order))
	for _, k := range order {
		out = append(out, *groups[k])
	}
	return out
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
