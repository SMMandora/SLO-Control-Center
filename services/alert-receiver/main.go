package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var alertsReceived = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "alerts_received_total", Help: "Alerts received by severity.",
}, []string{"severity"})

// store keeps the most recent alerts in memory for inspection.
type store struct {
	mu     sync.Mutex
	recent []AlertRef
}

func (s *store) add(refs []AlertRef) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range refs {
		alertsReceived.WithLabelValues(r.Severity).Inc()
		log.Printf("alert: %s severity=%s service=%s status=%s", r.Alertname, r.Severity, r.Service, r.Status)
	}
	s.recent = append(s.recent, refs...)
	if len(s.recent) > 200 {
		s.recent = s.recent[len(s.recent)-200:]
	}
}

func (s *store) list() []AlertRef {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]AlertRef, len(s.recent))
	copy(out, s.recent)
	return out
}

func main() {
	st := &store{}
	receive := func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		st.add(parseAMWebhook(body))
		w.WriteHeader(http.StatusOK)
	}

	mux := http.NewServeMux()
	// Alertmanager routes page/ticket/warn to these paths.
	mux.HandleFunc("/page", receive)
	mux.HandleFunc("/ticket", receive)
	mux.HandleFunc("/warn", receive)
	mux.HandleFunc("/alerts", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(st.list())
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	mux.Handle("/metrics", promhttp.Handler())

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8092"
	}
	log.Printf("alert-receiver listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
