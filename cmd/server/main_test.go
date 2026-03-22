package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/config"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/server"
)

// newTestServer builds the real server (chi router + middleware + routes)
// and returns its Handler for use with httptest.
func newTestServer() http.Handler {
	cfg := &config.ServerConfig{Address: "localhost:0"}
	return server.New(cfg).Handler
}

func TestAddGaugeMetric(t *testing.T) {
	h := newTestServer()
	r := httptest.NewRequest(http.MethodPost, "/update/gauge/cpu/42.5", nil)
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAddCounterMetric(t *testing.T) {
	h := newTestServer()
	r := httptest.NewRequest(http.MethodPost, "/update/counter/hits/10", nil)
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestEmptyString(t *testing.T) {
	// Content-Type is not required; a valid update without it must return 200.
	h := newTestServer()
	r := httptest.NewRequest(http.MethodPost, "/update/gauge/cpu/42.5", nil)
	// Deliberately do NOT set Content-Type.
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for missing Content-Type, got %d", w.Code)
	}
}

func TestWrongTypeMetric(t *testing.T) {
	// A metric type that is neither "gauge" nor "counter" must be rejected.
	h := newTestServer()
	r := httptest.NewRequest(http.MethodPost, "/update/histogram/latency/0.5", nil)
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unknown metric type, got %d", w.Code)
	}
}

func TestWrongValueMetric(t *testing.T) {
	// A non-numeric value for a counter must be rejected.
	h := newTestServer()
	r := httptest.NewRequest(http.MethodPost, "/update/counter/hits/notanumber", nil)
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-numeric counter value, got %d", w.Code)
	}
}

func TestShortPath(t *testing.T) {
	// A path that is missing the value segment does not match the pattern and
	// must return 404 (no route matched).
	h := newTestServer()
	r := httptest.NewRequest(http.MethodPost, "/update/gauge/cpu", nil)
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for incomplete path, got %d", w.Code)
	}
}
