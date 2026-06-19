package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// TraceRef is a recent trace surfaced to the UI.
type TraceRef struct {
	TraceID    string `json:"traceID"`
	Service    string `json:"service"`
	Name       string `json:"name"`
	DurationMs int64  `json:"durationMs"`
	StartedMs  int64  `json:"startedMs"`
	GrafanaURL string `json:"grafanaUrl"`
}

// Tempo is the minimal Tempo search surface the BFF needs.
type Tempo interface {
	SearchErrors(ctx context.Context) ([]TraceRef, error)
	SearchRecent(ctx context.Context) ([]TraceRef, error)
}

type httpTempo struct {
	base    string
	grafana string
	c       *http.Client
}

func NewTempo(base, grafana string) Tempo {
	return &httpTempo{base: base, grafana: grafana, c: &http.Client{Timeout: 5 * time.Second}}
}

// SearchErrors returns recent traces that recorded an error (e.g. failed payments).
func (t *httpTempo) SearchErrors(ctx context.Context) ([]TraceRef, error) {
	return t.search(ctx, `{ status = error }`)
}

// SearchRecent returns recent traces of any status (for the Traces tab).
func (t *httpTempo) SearchRecent(ctx context.Context) ([]TraceRef, error) {
	return t.search(ctx, `{ kind = server }`)
}

func (t *httpTempo) search(ctx context.Context, traceql string) ([]TraceRef, error) {
	u := fmt.Sprintf("%s/api/search?q=%s&limit=20", t.base, url.QueryEscape(traceql))
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	resp, err := t.c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parseTempoSearch(raw, t.grafana)
}

// parseTempoSearch maps a Tempo /api/search response body into TraceRefs.
func parseTempoSearch(body []byte, grafana string) ([]TraceRef, error) {
	var raw struct {
		Traces []struct {
			TraceID           string `json:"traceID"`
			RootServiceName   string `json:"rootServiceName"`
			RootTraceName     string `json:"rootTraceName"`
			StartTimeUnixNano string `json:"startTimeUnixNano"`
			DurationMs        int64  `json:"durationMs"`
		} `json:"traces"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	out := make([]TraceRef, 0, len(raw.Traces))
	for _, tr := range raw.Traces {
		out = append(out, mapTrace(tr.TraceID, tr.RootServiceName, tr.RootTraceName, tr.StartTimeUnixNano, tr.DurationMs, grafana))
	}
	return out, nil
}

// mapTrace builds a TraceRef incl. a best-effort Grafana Explore deep link.
func mapTrace(id, service, name, startNano string, durMs int64, grafana string) TraceRef {
	startMs := int64(0)
	if n, err := strconv.ParseInt(startNano, 10, 64); err == nil {
		startMs = n / 1e6
	}
	explore := url.QueryEscape(fmt.Sprintf(
		`{"datasource":"Tempo","queries":[{"refId":"A","queryType":"traceql","query":"%s"}],"range":{"from":"now-1h","to":"now"}}`, id))
	return TraceRef{
		TraceID:    id,
		Service:    service,
		Name:       name,
		DurationMs: durMs,
		StartedMs:  startMs,
		GrafanaURL: fmt.Sprintf("%s/explore?left=%s", grafana, explore),
	}
}
