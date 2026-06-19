package main

import "testing"

func TestChaosShouldError(t *testing.T) {
	c := Chaos{ErrorRate: 1.0}
	if !c.ShouldError(0.5) {
		t.Fatal("rate 1.0 must always error")
	}
	c = Chaos{ErrorRate: 0.0}
	if c.ShouldError(0.0) {
		t.Fatal("rate 0.0 must never error")
	}
	c = Chaos{ErrorRate: 0.3}
	if !c.ShouldError(0.2) {
		t.Fatal("roll 0.2 < 0.3 should error")
	}
	if c.ShouldError(0.4) {
		t.Fatal("roll 0.4 >= 0.3 should not error")
	}
}

func TestChaosLatency(t *testing.T) {
	c := Chaos{LatencyMS: 800, LatencyPct: 1.0}
	if c.LatencyFor(0.9).Milliseconds() != 800 {
		t.Fatal("expected 800ms")
	}
	c = Chaos{LatencyMS: 800, LatencyPct: 0.0}
	if c.LatencyFor(0.0) != 0 {
		t.Fatal("pct 0 => no latency")
	}
}
