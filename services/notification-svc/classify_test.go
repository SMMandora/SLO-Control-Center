package main

import "testing"

func TestClassify(t *testing.T) {
	status, latency := classify(1000, 3000, true)
	if status != "success" || latency != 2.0 {
		t.Fatalf("got %s %v", status, latency)
	}
	if s, _ := classify(0, 0, false); s != "failure" {
		t.Fatalf("expected failure, got %s", s)
	}
	if _, l := classify(5000, 1000, true); l != 0 {
		t.Fatalf("negative skew must clamp to 0, got %v", l)
	}
}
