package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/handlers"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/models"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/repository"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/service"
)

func newBenchHandler(b *testing.B) *handlers.MetricsHandler {
	b.Helper()
	return handlers.NewMetricsHandler(service.New(repository.NewMemStorage()), nil)
}

func BenchmarkUpdateHandler(b *testing.B) {
	h := newBenchHandler(b)
	for b.Loop() {
		b.StopTimer()
		r := httptest.NewRequest(http.MethodPost, "/update/gauge/cpu/42.5", nil)
		r.SetPathValue("kind", "gauge")
		r.SetPathValue("name", "cpu")
		r.SetPathValue("value", "42.5")
		w := httptest.NewRecorder()
		b.StartTimer()
		h.UpdateHandler(w, r)
	}
}

func BenchmarkJSONUpdateHandler(b *testing.B) {
	h := newBenchHandler(b)
	v := 42.5
	m := models.Metrics{ID: "cpu", MType: models.Gauge, Value: &v}
	body, _ := json.Marshal(m)
	for b.Loop() {
		b.StopTimer()
		r := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		b.StartTimer()
		h.JSONUpdateHandler(w, r)
	}
}

func BenchmarkBatchUpdateHandler(b *testing.B) {
	h := newBenchHandler(b)
	batch := make([]models.Metrics, 30)
	for i := range batch {
		v := float64(i)
		batch[i] = models.Metrics{ID: fmt.Sprintf("m%d", i), MType: models.Gauge, Value: &v}
	}
	body, _ := json.Marshal(batch)
	for b.Loop() {
		b.StopTimer()
		r := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		b.StartTimer()
		h.BatchUpdateHandler(w, r)
	}
}

func BenchmarkListHandler(b *testing.B) {
	repo := repository.NewMemStorage()
	ctx := context.Background()
	for i := 0; i < 30; i++ {
		v := float64(i)
		_ = repo.SetGauge(ctx, fmt.Sprintf("gauge%d", i), v)
		_ = repo.AddCounter(ctx, fmt.Sprintf("counter%d", i), int64(i))
	}
	h := handlers.NewMetricsHandler(service.New(repo), nil)
	for b.Loop() {
		b.StopTimer()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		b.StartTimer()
		h.ListHandler(w, r)
	}
}
