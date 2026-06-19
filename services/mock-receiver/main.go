package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var received = promauto.NewCounter(prometheus.CounterOpts{
	Name: "webhooks_received_total", Help: "Webhooks received by the mock receiver.",
})

func main() {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	mux.HandleFunc("/webhook", func(w http.ResponseWriter, _ *http.Request) {
		received.Inc()
		w.WriteHeader(http.StatusOK)
	})

	shutdown := initTracer(context.Background())
	defer shutdown()

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8091"
	}
	log.Printf("mock-receiver listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, otelhttp.NewHandler(mux, "mock-receiver")))
}
