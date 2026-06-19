package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server is the HTTP server wiring routes, store, chaos config, and the payment
// publisher.
type Server struct {
	mux   *http.ServeMux
	store *Store
	chaos Chaos
	roll  func() float64 // injectable RNG for tests
	pub   Publisher      // nil disables enqueue (tests)
}

// NewServer builds the route table. roll supplies [0,1) values for chaos
// decisions; pub may be nil to disable payment enqueue.
func NewServer(store *Store, chaos Chaos, roll func() float64, pub Publisher) *Server {
	s := &Server{mux: http.NewServeMux(), store: store, chaos: chaos, roll: roll, pub: pub}
	s.mux.Handle("/metrics", promhttp.Handler())
	s.mux.HandleFunc("/healthz", instrument("/healthz", s.healthz))
	s.mux.HandleFunc("/orders", instrument("/orders", s.orders))
	s.mux.HandleFunc("/orders/", instrument("/orders/:id", s.getOrder))
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

// applyChaos injects latency and/or a 5xx based on the configured rates.
// Returns true if the request was failed (caller should stop).
func (s *Server) applyChaos(ctx context.Context, w http.ResponseWriter) bool {
	if d := s.chaos.LatencyFor(s.roll()); d > 0 {
		time.Sleep(d)
	}
	if s.chaos.ShouldError(s.roll()) {
		logger.Error("chaos injected failure", "trace_id", traceID(ctx))
		http.Error(w, "chaos: injected failure", http.StatusInternalServerError)
		return true
	}
	return false
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) orders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	if s.applyChaos(r.Context(), w) {
		return
	}
	var body struct {
		Item string `json:"item"`
		Qty  int    `json:"qty"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Item == "" {
		body.Item = "widget"
	}
	if body.Qty == 0 {
		body.Qty = 1
	}
	id, err := s.store.CreateOrder(r.Context(), body.Item, body.Qty)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Best-effort enqueue: a payments hiccup must not fail the order.
	if s.pub != nil {
		if err := s.pub.EnqueuePayment(r.Context(), id, body.Item, body.Qty); err != nil {
			log.Printf("enqueue payment for order %d: %v", id, err)
		}
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]int64{"id": id})
}

func (s *Server) getOrder(w http.ResponseWriter, r *http.Request) {
	if s.applyChaos(r.Context(), w) {
		return
	}
	idStr := r.URL.Path[len("/orders/"):]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	item, qty, err := s.store.GetOrder(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"id": id, "item": item, "qty": qty})
}
