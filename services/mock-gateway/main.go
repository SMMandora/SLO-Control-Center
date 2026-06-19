package main

import (
	"context"
	"encoding/json"
	"hash/fnv"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var charges = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "gateway_charges_total", Help: "Mock gateway charge attempts.",
}, []string{"result"})

func envInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// shouldFail decides deterministically whether a charge for orderID fails.
// Failure when fnv32(orderID) % 100 < failPct. No RNG => reproducible.
func shouldFail(orderID string, failPct int) bool {
	h := fnv.New32a()
	_, _ = h.Write([]byte(orderID))
	return int(h.Sum32()%100) < failPct
}

func main() {
	failPct := 8
	if v := os.Getenv("FAIL_PCT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			failPct = n
		}
	}

	latencyMS := envInt("LATENCY_MS", 0)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	mux.HandleFunc("/charge", func(w http.ResponseWriter, r *http.Request) {
		if latencyMS > 0 {
			time.Sleep(time.Duration(latencyMS) * time.Millisecond)
		}
		var body struct {
			OrderID string `json:"order_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if shouldFail(body.OrderID, failPct) {
			charges.WithLabelValues("declined").Inc()
			http.Error(w, `{"status":"declined"}`, http.StatusBadGateway)
			return
		}
		charges.WithLabelValues("charged").Inc()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "charged"})
	})

	shutdown := initTracer(context.Background())
	defer shutdown()

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8090"
	}
	log.Printf("mock-gateway listening on %s (fail_pct=%d)", addr, failPct)
	log.Fatal(http.ListenAndServe(addr, otelhttp.NewHandler(mux, "mock-gateway")))
}
