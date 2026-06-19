package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"
)

// LogLine is one structured log entry surfaced to the UI.
type LogLine struct {
	TsMs    int64  `json:"tsMs"`
	Level   string `json:"level"`
	Service string `json:"service"`
	TraceID string `json:"traceID"`
	Line    string `json:"line"`
}

// LogSource queries Loki for recent log lines.
type LogSource interface {
	Query(ctx context.Context, logql string, limit int) ([]byte, error)
}

type httpLoki struct {
	base string
	c    *http.Client
}

func NewLoki(base string) LogSource {
	return &httpLoki{base: base, c: &http.Client{Timeout: 5 * time.Second}}
}

func (l *httpLoki) Query(ctx context.Context, logql string, limit int) ([]byte, error) {
	end := time.Now()
	start := end.Add(-1 * time.Hour)
	u := fmt.Sprintf("%s/loki/api/v1/query_range?query=%s&limit=%d&start=%d&end=%d&direction=backward",
		l.base, url.QueryEscape(logql), limit, start.UnixNano(), end.UnixNano())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	resp, err := l.c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// parseLokiQuery maps a Loki query_range response into LogLines. Each Loki
// stream value is [ts_ns, line]; the line is JSON our services emitted.
func parseLokiQuery(body []byte) []LogLine {
	var p struct {
		Data struct {
			Result []struct {
				Stream map[string]string `json:"stream"`
				Values [][2]string       `json:"values"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &p); err != nil {
		return nil
	}
	out := []LogLine{}
	for _, r := range p.Data.Result {
		for _, v := range r.Values {
			tsNs, _ := strconv.ParseInt(v[0], 10, 64)
			ll := LogLine{TsMs: tsNs / 1e6, Level: r.Stream["level"], Line: v[1]}
			// The log line itself is the JSON our services logged.
			var fields map[string]any
			if json.Unmarshal([]byte(v[1]), &fields) == nil {
				if s, ok := fields["trace_id"].(string); ok {
					ll.TraceID = s
				}
				if s, ok := fields["msg"].(string); ok {
					ll.Line = s
				}
				if ll.Level == "" {
					if s, ok := fields["level"].(string); ok {
						ll.Level = s
					}
				}
			}
			ll.Service = r.Stream["container"]
			out = append(out, ll)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TsMs > out[j].TsMs })
	return out
}
