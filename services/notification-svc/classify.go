package main

// classify returns the delivery status and emit-to-delivered latency in seconds.
// latency is clamped to >= 0 to tolerate minor clock skew.
func classify(emittedMs, doneMs int64, ok bool) (string, float64) {
	latency := float64(doneMs-emittedMs) / 1000.0
	if latency < 0 {
		latency = 0
	}
	if ok {
		return "success", latency
	}
	return "failure", latency
}
