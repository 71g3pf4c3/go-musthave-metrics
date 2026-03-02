package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	repository "github.com/71g3pf4c3/go-musthave-metrics/internal/repository"
	metrics "github.com/71g3pf4c3/go-musthave-metrics/internal/service"
)

// newTestMux mirrors the mux setup in main() so tests run against real routing.
func newTestMux() *http.ServeMux {
	mux := http.NewServeMux()
	storage := repository.NewMemStorage()
	server := metrics.NewMetricsServer(storage)
	mux.HandleFunc(`POST /update/{kind}/{name}/{value}`, server.UpdateHandler)
	return mux
}

func TestAddGaugeMetric(t *testing.T) {
	mux := newTestMux()
	req := httptest.NewRequest(http.MethodPost, "/update/gauge/cpu/42.5", nil)
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAddCounterMetric(t *testing.T) {
	mux := newTestMux()
	req := httptest.NewRequest(http.MethodPost, "/update/counter/hits/10", nil)
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestEmptyString(t *testing.T) {
	// Content-Type is not required; a valid update without it must return 200.
	mux := newTestMux()
	req := httptest.NewRequest(http.MethodPost, "/update/gauge/cpu/42.5", nil)
	// Deliberately do NOT set Content-Type.
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for missing Content-Type, got %d", w.Code)
	}
}

func TestWrongTypeMetric(t *testing.T) {
	// A metric type that is neither "gauge" nor "counter" must be rejected.
	mux := newTestMux()
	req := httptest.NewRequest(http.MethodPost, "/update/histogram/latency/0.5", nil)
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unknown metric type, got %d", w.Code)
	}
}

func TestWrongValueMetric(t *testing.T) {
	// A non-numeric value for a counter must be rejected.
	mux := newTestMux()
	req := httptest.NewRequest(http.MethodPost, "/update/counter/hits/notanumber", nil)
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-numeric counter value, got %d", w.Code)
	}
}

func TestShortPath(t *testing.T) {
	// A path that is missing the value segment does not match the pattern and
	// must return 404 (no route matched).
	mux := newTestMux()
	req := httptest.NewRequest(http.MethodPost, "/update/gauge/cpu", nil)
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for incomplete path, got %d", w.Code)
	}
}
