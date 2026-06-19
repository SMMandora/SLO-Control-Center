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

// Point is one (unix-seconds, value) sample of a compliance series.
type Point struct {
	T int64   `json:"t"`
	V float64 `json:"sliPct"`
}

// Sample is one labeled instant value from a vector query.
type Sample struct {
	Labels map[string]string
	Value  float64
}

// Prom is the minimal Prometheus query surface the BFF needs.
type Prom interface {
	Query(ctx context.Context, q string) (float64, error)
	QueryRange(ctx context.Context, q string, span, step time.Duration) ([]Point, error)
	QueryVector(ctx context.Context, q string) ([]Sample, error)
}

type httpProm struct {
	base string
	c    *http.Client
}

// NewProm returns a Prometheus HTTP API client rooted at base (e.g. http://prometheus:9090).
func NewProm(base string) Prom {
	return &httpProm{base: base, c: &http.Client{Timeout: 5 * time.Second}}
}

func (p *httpProm) Query(ctx context.Context, q string) (float64, error) {
	u := fmt.Sprintf("%s/api/v1/query?query=%s", p.base, url.QueryEscape(q))
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	resp, err := p.c.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var out struct {
		Data struct {
			Result []struct {
				Value [2]any `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, err
	}
	if len(out.Data.Result) == 0 {
		return 0, nil
	}
	s, _ := out.Data.Result[0].Value[1].(string)
	return strconv.ParseFloat(s, 64)
}

func (p *httpProm) QueryVector(ctx context.Context, q string) ([]Sample, error) {
	u := fmt.Sprintf("%s/api/v1/query?query=%s", p.base, url.QueryEscape(q))
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	resp, err := p.c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out struct {
		Data struct {
			Result []struct {
				Metric map[string]string `json:"metric"`
				Value  [2]any            `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	samples := make([]Sample, 0, len(out.Data.Result))
	for _, r := range out.Data.Result {
		s, _ := r.Value[1].(string)
		f, _ := strconv.ParseFloat(s, 64)
		samples = append(samples, Sample{Labels: r.Metric, Value: f})
	}
	return samples, nil
}

func (p *httpProm) QueryRange(ctx context.Context, q string, span, step time.Duration) ([]Point, error) {
	end := time.Now()
	start := end.Add(-span)
	u := fmt.Sprintf("%s/api/v1/query_range?query=%s&start=%d&end=%d&step=%d",
		p.base, url.QueryEscape(q), start.Unix(), end.Unix(), int(step.Seconds()))
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	resp, err := p.c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out struct {
		Data struct {
			Result []struct {
				Values [][2]any `json:"values"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
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
