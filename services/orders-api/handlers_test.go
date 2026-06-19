package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthz(t *testing.T) {
	h := NewServer(nil, Chaos{}, func() float64 { return 0 }, nil)
	r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("healthz = %d", w.Code)
	}
}

func TestCreateOrderChaosError(t *testing.T) {
	// roll returns 0.0 so ShouldError is true for any positive rate; the 500
	// is returned before the nil store is ever touched.
	h := NewServer(nil, Chaos{ErrorRate: 1.0}, func() float64 { return 0 }, nil)
	r := httptest.NewRequest(http.MethodPost, "/orders", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}
