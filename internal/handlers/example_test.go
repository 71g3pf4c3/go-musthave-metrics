package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/handlers"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/models"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/repository"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/service"
)

func newExampleHandler() *handlers.MetricsHandler {
	return handlers.NewMetricsHandler(service.New(repository.NewMemStorage()), nil)
}

// Store a gauge via URL: POST /update/{kind}/{name}/{value}
func ExampleMetricsHandler_UpdateHandler() {
	h := newExampleHandler()

	r := httptest.NewRequest(http.MethodPost, "/update/gauge/cpu/42.5", nil)
	r.SetPathValue("kind", "gauge")
	r.SetPathValue("name", "cpu")
	r.SetPathValue("value", "42.5")
	w := httptest.NewRecorder()

	h.UpdateHandler(w, r)

	fmt.Println(w.Code)
	// Output:
	// 200
}

// Read a gauge via URL: GET /value/{kind}/{name}
func ExampleMetricsHandler_GetHandler() {
	h := newExampleHandler()

	// store first
	rUpd := httptest.NewRequest(http.MethodPost, "/update/gauge/cpu/42.5", nil)
	rUpd.SetPathValue("kind", "gauge")
	rUpd.SetPathValue("name", "cpu")
	rUpd.SetPathValue("value", "42.5")
	h.UpdateHandler(httptest.NewRecorder(), rUpd)

	// then read
	rGet := httptest.NewRequest(http.MethodGet, "/value/gauge/cpu", nil)
	rGet.SetPathValue("kind", "gauge")
	rGet.SetPathValue("name", "cpu")
	w := httptest.NewRecorder()

	h.GetHandler(w, rGet)

	body, _ := io.ReadAll(w.Body)
	fmt.Println(w.Code)
	fmt.Println(string(body))
	// Output:
	// 200
	// 42.5
}

// Store a metric via JSON: POST /update
func ExampleMetricsHandler_JSONUpdateHandler() {
	h := newExampleHandler()

	v := 99.9
	m := models.Metrics{ID: "memory", MType: models.Gauge, Value: &v}
	body, _ := json.Marshal(m)

	r := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.JSONUpdateHandler(w, r)

	fmt.Println(w.Code)
	// Output:
	// 200
}

// Read a metric via JSON: POST /value
func ExampleMetricsHandler_JSONGetHandler() {
	h := newExampleHandler()

	// store first
	v := 7.0
	upd := models.Metrics{ID: "temp", MType: models.Gauge, Value: &v}
	updBody, _ := json.Marshal(upd)
	rUpd := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(updBody))
	rUpd.Header.Set("Content-Type", "application/json")
	h.JSONUpdateHandler(httptest.NewRecorder(), rUpd)

	// then read
	get := models.Metrics{ID: "temp", MType: models.Gauge}
	getBody, _ := json.Marshal(get)
	r := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(getBody))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.JSONGetHandler(w, r)

	var resp models.Metrics
	json.NewDecoder(w.Body).Decode(&resp)

	fmt.Println(w.Code)
	fmt.Println(*resp.Value)
	// Output:
	// 200
	// 7
}

// Store multiple metrics at once: POST /updates
func ExampleMetricsHandler_BatchUpdateHandler() {
	h := newExampleHandler()

	v1 := 1.5
	var d2 int64 = 10
	batch := []models.Metrics{
		{ID: "cpu", MType: models.Gauge, Value: &v1},
		{ID: "requests", MType: models.Counter, Delta: &d2},
	}
	body, _ := json.Marshal(batch)

	r := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.BatchUpdateHandler(w, r)

	fmt.Println(w.Code)
	// Output:
	// 200
}
