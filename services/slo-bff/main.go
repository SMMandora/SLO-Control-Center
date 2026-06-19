package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	base := os.Getenv("PROMETHEUS_URL")
	if base == "" {
		base = "http://prometheus:9090"
	}
	tempoURL := os.Getenv("TEMPO_URL")
	if tempoURL == "" {
		tempoURL = "http://tempo:3200"
	}
	grafanaURL := os.Getenv("GRAFANA_URL")
	if grafanaURL == "" {
		grafanaURL = "http://localhost:3001"
	}
	lokiURL := os.Getenv("LOKI_URL")
	if lokiURL == "" {
		lokiURL = "http://loki:3100"
	}
	runbooksDir := os.Getenv("RUNBOOKS_DIR")
	if runbooksDir == "" {
		runbooksDir = "/runbooks"
	}
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":9090"
	}
	mux := NewAPI(NewProm(base), NewTempo(tempoURL, grafanaURL), NewAlertSource(base), NewLoki(lokiURL), runbooksDir)
	log.Printf("slo-bff listening on %s -> %s", addr, base)
	log.Fatal(http.ListenAndServe(addr, mux))
}
