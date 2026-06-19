package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total", Help: "Total HTTP requests.",
	}, []string{"route", "method", "status"})

	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_request_duration_seconds", Help: "HTTP request latency.",
		Buckets: []float64{.005, .01, .025, .05, .1, .15, .2, .3, .5, 1},
	}, []string{"route", "method"})

	inFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "http_requests_in_flight", Help: "In-flight HTTP requests.",
	})
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(c int) {
	r.status = c
	r.ResponseWriter.WriteHeader(c)
}

// instrument wraps a handler with RED metrics under a fixed route label.
func instrument(route string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		inFlight.Inc()
		defer inFlight.Dec()
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		start := time.Now()
		h(rec, req)
		httpDuration.WithLabelValues(route, req.Method).Observe(time.Since(start).Seconds())
		httpRequests.WithLabelValues(route, req.Method, strconv.Itoa(rec.status)).Inc()
	}
}
