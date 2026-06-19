package main

import (
	"strings"
	"testing"
)

func TestParseTempoSearch(t *testing.T) {
	body := []byte(`{"traces":[
		{"traceID":"abc123","rootServiceName":"orders-api","rootTraceName":"POST /orders","startTimeUnixNano":"1700000000000000000","durationMs":42}
	]}`)
	refs, err := parseTempoSearch(body, "http://localhost:3001")
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 {
		t.Fatalf("want 1 ref, got %d", len(refs))
	}
	r := refs[0]
	if r.TraceID != "abc123" || r.Service != "orders-api" || r.DurationMs != 42 {
		t.Fatalf("bad mapping: %+v", r)
	}
	if r.StartedMs != 1700000000000 {
		t.Fatalf("startedMs = %d", r.StartedMs)
	}
	if !strings.Contains(r.GrafanaURL, "abc123") || !strings.HasPrefix(r.GrafanaURL, "http://localhost:3001/explore") {
		t.Fatalf("grafana url: %s", r.GrafanaURL)
	}
}

func TestParseTempoSearchEmpty(t *testing.T) {
	refs, err := parseTempoSearch([]byte(`{}`), "http://g")
	if err != nil || len(refs) != 0 {
		t.Fatalf("empty body should yield no refs: %v %d", err, len(refs))
	}
}
