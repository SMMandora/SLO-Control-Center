package main

import "time"

// Chaos holds the fault-injection configuration for the service.
type Chaos struct {
	ErrorRate  float64
	LatencyMS  int
	LatencyPct float64
}

// ShouldError reports whether a request with the given [0,1) roll should 5xx.
func (c Chaos) ShouldError(roll float64) bool { return roll < c.ErrorRate }

// LatencyFor returns the injected delay for a request with the given [0,1) roll.
func (c Chaos) LatencyFor(roll float64) time.Duration {
	if roll < c.LatencyPct {
		return time.Duration(c.LatencyMS) * time.Millisecond
	}
	return 0
}
