package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/config"
)

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	cfg := &config.ServerConfig{Address: "localhost:0"}
	app, err := newServer(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	return app.http.Handler
}

func TestAddGaugeMetric(t *testing.T) {
	h := newTestServer(t)
	r := httptest.NewRequest(http.MethodPost, "/update/gauge/cpu/42.5", nil)
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAddCounterMetric(t *testing.T) {
	h := newTestServer(t)
	r := httptest.NewRequest(http.MethodPost, "/update/counter/hits/10", nil)
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestEmptyString(t *testing.T) {
	h := newTestServer(t)
	r := httptest.NewRequest(http.MethodPost, "/update/gauge/cpu/42.5", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for missing Content-Type, got %d", w.Code)
	}
}

func TestWrongTypeMetric(t *testing.T) {
	h := newTestServer(t)
	r := httptest.NewRequest(http.MethodPost, "/update/histogram/latency/0.5", nil)
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unknown metric type, got %d", w.Code)
	}
}

func TestWrongValueMetric(t *testing.T) {
	h := newTestServer(t)
	r := httptest.NewRequest(http.MethodPost, "/update/counter/hits/notanumber", nil)
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-numeric counter value, got %d", w.Code)
	}
}

func TestShortPath(t *testing.T) {
	h := newTestServer(t)
	r := httptest.NewRequest(http.MethodPost, "/update/gauge/cpu", nil)
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for incomplete path, got %d", w.Code)
	}
}
