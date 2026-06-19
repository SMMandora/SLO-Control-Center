package main

import "testing"

func TestParseLokiQuery(t *testing.T) {
	body := []byte(`{"data":{"result":[
	  {"stream":{"container":"compose-payments-worker-1","level":"error"},
	   "values":[
	     ["1700000002000000000","{\"level\":\"error\",\"msg\":\"payment failed\",\"trace_id\":\"abc123\"}"],
	     ["1700000001000000000","{\"level\":\"error\",\"msg\":\"payment failed\",\"trace_id\":\"def456\"}"]
	   ]}
	]}}`)
	lines := parseLokiQuery(body)
	if len(lines) != 2 {
		t.Fatalf("want 2 lines, got %d", len(lines))
	}
	// sorted newest first
	if lines[0].TsMs != 1700000002000 {
		t.Fatalf("ts %d", lines[0].TsMs)
	}
	if lines[0].TraceID != "abc123" || lines[0].Line != "payment failed" || lines[0].Level != "error" {
		t.Fatalf("bad line: %+v", lines[0])
	}
	if lines[0].Service != "compose-payments-worker-1" {
		t.Fatalf("service: %q", lines[0].Service)
	}
}
