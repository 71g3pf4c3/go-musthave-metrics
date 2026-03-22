package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/handlers"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/models"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/repository"
)

// errorResponseWriter is an http.ResponseWriter whose Write always fails.
// It lets tests exercise the enc.Encode error branches in JsonGetHandler.
type errorResponseWriter struct {
	header http.Header
	code   int
}

func (e *errorResponseWriter) Header() http.Header {
	if e.header == nil {
		e.header = make(http.Header)
	}
	return e.header
}

func (e *errorResponseWriter) Write([]byte) (int, error) {
	return 0, errors.New("simulated write failure")
}

func (e *errorResponseWriter) WriteHeader(statusCode int) { e.code = statusCode }

func newHandler() *handlers.MetricsServer {
	return handlers.NewMetricsServer(repository.NewMemStorage())
}

func makeRequest(kind, name, value string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/update/"+kind+"/"+name+"/"+value, nil)
	req.Header.Set("Content-Type", "text/plain")
	req.SetPathValue("kind", kind)
	req.SetPathValue("name", name)
	req.SetPathValue("value", value)
	return req
}

func makeGetRequest(kind, name string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/value/"+kind+"/"+name, nil)
	req.SetPathValue("kind", kind)
	req.SetPathValue("name", name)
	return req
}

func makeJSONUpdateRequest(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func makeJSONGetRequest(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// --- ListHandler ---

func TestListHandlerEmpty(t *testing.T) {
	w := httptest.NewRecorder()
	newHandler().ListHandler(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html" {
		t.Errorf("expected Content-Type text/html, got %q", ct)
	}
}

func TestListHandlerGauge(t *testing.T) {
	h := newHandler()
	h.UpdateHandler(httptest.NewRecorder(), makeRequest("gauge", "cpu", "42.5"))

	w := httptest.NewRecorder()
	h.ListHandler(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "cpu") {
		t.Errorf("expected body to contain \"cpu\", got %q", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "42.5") {
		t.Errorf("expected body to contain \"42.5\", got %q", w.Body.String())
	}
}

func TestListHandlerCounter(t *testing.T) {
	h := newHandler()
	h.UpdateHandler(httptest.NewRecorder(), makeRequest("counter", "hits", "10"))
	h.UpdateHandler(httptest.NewRecorder(), makeRequest("counter", "hits", "5"))

	w := httptest.NewRecorder()
	h.ListHandler(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "hits") {
		t.Errorf("expected body to contain \"hits\", got %q", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "15") {
		t.Errorf("expected body to contain \"15\", got %q", w.Body.String())
	}
}

func TestListHandlerBoth(t *testing.T) {
	h := newHandler()
	h.UpdateHandler(httptest.NewRecorder(), makeRequest("gauge", "mem", "1.5"))
	h.UpdateHandler(httptest.NewRecorder(), makeRequest("counter", "reqs", "7"))

	w := httptest.NewRecorder()
	h.ListHandler(w, httptest.NewRequest(http.MethodGet, "/", nil))
	body := w.Body.String()
	for _, want := range []string{"mem", "1.5", "reqs", "7"} {
		if !strings.Contains(body, want) {
			t.Errorf("expected body to contain %q, got %q", want, body)
		}
	}
}

// --- counter: valid values ---

func TestUpdateHandlerAddCounterInt(t *testing.T) {
	w := httptest.NewRecorder()
	newHandler().UpdateHandler(w, makeRequest("counter", "hits", "100"))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestUpdateHandlerAddCounterInt32(t *testing.T) {
	w := httptest.NewRecorder()
	// 2147483647 is math.MaxInt32 — fits easily in int64.
	newHandler().UpdateHandler(w, makeRequest("counter", "hits", "2147483647"))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- counter: invalid values ---

func TestUpdateHandlerAddCounterFloat(t *testing.T) {
	w := httptest.NewRecorder()
	newHandler().UpdateHandler(w, makeRequest("counter", "hits", "1.5"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for float counter value, got %d", w.Code)
	}
}

func TestUpdateHandlerAddCouterFloat32(t *testing.T) {
	w := httptest.NewRecorder()
	newHandler().UpdateHandler(w, makeRequest("counter", "hits", "3.14"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for float32-like counter value, got %d", w.Code)
	}
}

func TestUpdateHandlerAddCouterString(t *testing.T) {
	w := httptest.NewRecorder()
	newHandler().UpdateHandler(w, makeRequest("counter", "hits", "abc"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for string counter value, got %d", w.Code)
	}
}

// --- gauge: valid values ---

func TestUpdateHandlerAddGaugeInt(t *testing.T) {
	w := httptest.NewRecorder()
	newHandler().UpdateHandler(w, makeRequest("gauge", "cpu", "100"))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestUpdateHandlerAddGaugeInt32(t *testing.T) {
	w := httptest.NewRecorder()
	newHandler().UpdateHandler(w, makeRequest("gauge", "cpu", "2147483647"))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestUpdateHandlerAddGaugeFloat(t *testing.T) {
	w := httptest.NewRecorder()
	newHandler().UpdateHandler(w, makeRequest("gauge", "cpu", "1.5"))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestUpdateHandlerAddGaugeFloat32(t *testing.T) {
	w := httptest.NewRecorder()
	newHandler().UpdateHandler(w, makeRequest("gauge", "cpu", "3.14"))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- GetHandler ---

func TestGetHandlerGauge(t *testing.T) {
	h := newHandler()
	h.UpdateHandler(httptest.NewRecorder(), makeRequest("gauge", "cpu", "42.5"))

	w := httptest.NewRecorder()
	h.GetHandler(w, makeGetRequest("gauge", "cpu"))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "42.5" {
		t.Errorf("expected body \"42.5\", got %q", w.Body.String())
	}
}

func TestGetHandlerCounter(t *testing.T) {
	h := newHandler()
	h.UpdateHandler(httptest.NewRecorder(), makeRequest("counter", "hits", "10"))
	h.UpdateHandler(httptest.NewRecorder(), makeRequest("counter", "hits", "5"))

	w := httptest.NewRecorder()
	h.GetHandler(w, makeGetRequest("counter", "hits"))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "15" {
		t.Errorf("expected body \"15\", got %q", w.Body.String())
	}
}

func TestGetHandlerNotFound(t *testing.T) {
	h := newHandler()

	w := httptest.NewRecorder()
	h.GetHandler(w, makeGetRequest("gauge", "nonexistent"))
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetHandlerUnknownKind(t *testing.T) {
	h := newHandler()

	w := httptest.NewRecorder()
	h.GetHandler(w, makeGetRequest("histogram", "cpu"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetHandlerMissingName(t *testing.T) {
	h := newHandler()

	w := httptest.NewRecorder()
	h.GetHandler(w, makeGetRequest("gauge", ""))
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- UpdateHandler: missing path-value branches ---

func TestUpdateHandlerUnknownKind(t *testing.T) {
	w := httptest.NewRecorder()
	newHandler().UpdateHandler(w, makeRequest("histogram", "latency", "0.5"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unknown kind, got %d", w.Code)
	}
}

func TestUpdateHandlerEmptyName(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/update/gauge//42.5", nil)
	req.SetPathValue("kind", "gauge")
	req.SetPathValue("name", "")
	req.SetPathValue("value", "42.5")
	w := httptest.NewRecorder()
	newHandler().UpdateHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty name, got %d", w.Code)
	}
}

func TestUpdateHandlerEmptyValue(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/update/gauge/cpu/", nil)
	req.SetPathValue("kind", "gauge")
	req.SetPathValue("name", "cpu")
	req.SetPathValue("value", "")
	w := httptest.NewRecorder()
	newHandler().UpdateHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty value, got %d", w.Code)
	}
}

// --- MainPageHandler ---

func TestMainPageHandler(t *testing.T) {
	w := httptest.NewRecorder()
	handlers.MainPageHandler(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.Len() == 0 {
		t.Error("expected non-empty body")
	}
}

// --- gauge: invalid values ---

func TestUpdateHandlerAddGaugeString(t *testing.T) {
	w := httptest.NewRecorder()
	newHandler().UpdateHandler(w, makeRequest("gauge", "cpu", "abc"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for string gauge value, got %d", w.Code)
	}
}

// --- JsonUpdateHandler ---

func TestJsonUpdateHandlerGauge(t *testing.T) {
	h := newHandler()
	v := 42.5
	m := models.Metrics{ID: "cpu", MType: models.Gauge, Value: &v}
	body, _ := json.Marshal(m)

	w := httptest.NewRecorder()
	h.JSONUpdateHandler(w, makeJSONUpdateRequest(string(body)))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Verify value was stored by reading it back.
	rw := httptest.NewRecorder()
	h.GetHandler(rw, makeGetRequest("gauge", "cpu"))
	if rw.Body.String() != "42.5" {
		t.Errorf("expected stored gauge \"42.5\", got %q", rw.Body.String())
	}
}

func TestJsonUpdateHandlerCounter(t *testing.T) {
	h := newHandler()
	var delta int64 = 10
	m := models.Metrics{ID: "hits", MType: models.Counter, Delta: &delta}
	body, _ := json.Marshal(m)

	w := httptest.NewRecorder()
	h.JSONUpdateHandler(w, makeJSONUpdateRequest(string(body)))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Verify value was stored.
	rw := httptest.NewRecorder()
	h.GetHandler(rw, makeGetRequest("counter", "hits"))
	if rw.Body.String() != "10" {
		t.Errorf("expected stored counter \"10\", got %q", rw.Body.String())
	}
}

func TestJsonUpdateHandlerCounterAccumulates(t *testing.T) {
	h := newHandler()

	for _, delta := range []int64{10, 5} {
		d := delta
		m := models.Metrics{ID: "hits", MType: models.Counter, Delta: &d}
		body, _ := json.Marshal(m)
		h.JSONUpdateHandler(httptest.NewRecorder(), makeJSONUpdateRequest(string(body)))
	}

	rw := httptest.NewRecorder()
	h.GetHandler(rw, makeGetRequest("counter", "hits"))
	if rw.Body.String() != "15" {
		t.Errorf("expected accumulated counter \"15\", got %q", rw.Body.String())
	}
}

func TestJsonUpdateHandlerGaugeOverrides(t *testing.T) {
	h := newHandler()

	for _, v := range []float64{1.0, 99.9} {
		val := v
		m := models.Metrics{ID: "cpu", MType: models.Gauge, Value: &val}
		body, _ := json.Marshal(m)
		h.JSONUpdateHandler(httptest.NewRecorder(), makeJSONUpdateRequest(string(body)))
	}

	rw := httptest.NewRecorder()
	h.GetHandler(rw, makeGetRequest("gauge", "cpu"))
	if rw.Body.String() != "99.9" {
		t.Errorf("expected overwritten gauge \"99.9\", got %q", rw.Body.String())
	}
}

func TestJsonUpdateHandlerInvalidJSON(t *testing.T) {
	h := newHandler()

	w := httptest.NewRecorder()
	h.JSONUpdateHandler(w, makeJSONUpdateRequest("not-json"))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for invalid JSON, got %d", w.Code)
	}
}

func TestJsonUpdateHandlerUnknownType(t *testing.T) {
	h := newHandler()
	m := models.Metrics{ID: "latency", MType: "histogram"}
	body, _ := json.Marshal(m)

	w := httptest.NewRecorder()
	h.JSONUpdateHandler(w, makeJSONUpdateRequest(string(body)))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unknown metric type, got %d", w.Code)
	}
}

// --- JsonGetHandler ---

func TestJsonGetHandlerGauge(t *testing.T) {
	h := newHandler()
	v := 42.5
	// Store the gauge first via JsonUpdateHandler.
	upd := models.Metrics{ID: "cpu", MType: models.Gauge, Value: &v}
	updBody, _ := json.Marshal(upd)
	h.JSONUpdateHandler(httptest.NewRecorder(), makeJSONUpdateRequest(string(updBody)))

	// Now retrieve it.
	get := models.Metrics{ID: "cpu", MType: models.Gauge}
	getBody, _ := json.Marshal(get)
	w := httptest.NewRecorder()
	h.JSONGetHandler(w, makeJSONGetRequest(string(getBody)))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp models.Metrics
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ID != "cpu" {
		t.Errorf("expected id \"cpu\", got %q", resp.ID)
	}
	if resp.MType != models.Gauge {
		t.Errorf("expected type %q, got %q", models.Gauge, resp.MType)
	}
	if resp.Value == nil || *resp.Value != 42.5 {
		t.Errorf("expected value 42.5, got %v", resp.Value)
	}
}

func TestJsonGetHandlerCounter(t *testing.T) {
	h := newHandler()
	var delta int64 = 7
	upd := models.Metrics{ID: "hits", MType: models.Counter, Delta: &delta}
	updBody, _ := json.Marshal(upd)
	h.JSONUpdateHandler(httptest.NewRecorder(), makeJSONUpdateRequest(string(updBody)))

	get := models.Metrics{ID: "hits", MType: models.Counter}
	getBody, _ := json.Marshal(get)
	w := httptest.NewRecorder()
	h.JSONGetHandler(w, makeJSONGetRequest(string(getBody)))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp models.Metrics
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ID != "hits" {
		t.Errorf("expected id \"hits\", got %q", resp.ID)
	}
	if resp.MType != models.Counter {
		t.Errorf("expected type %q, got %q", models.Counter, resp.MType)
	}
	if resp.Delta == nil || *resp.Delta != 7 {
		t.Errorf("expected delta 7, got %v", resp.Delta)
	}
}

func TestJsonGetHandlerGaugeNotFound(t *testing.T) {
	h := newHandler()
	get := models.Metrics{ID: "nonexistent", MType: models.Gauge}
	getBody, _ := json.Marshal(get)

	w := httptest.NewRecorder()
	h.JSONGetHandler(w, makeJSONGetRequest(string(getBody)))
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing gauge, got %d", w.Code)
	}
}

func TestJsonGetHandlerCounterNotFound(t *testing.T) {
	h := newHandler()
	get := models.Metrics{ID: "nonexistent", MType: models.Counter}
	getBody, _ := json.Marshal(get)

	w := httptest.NewRecorder()
	h.JSONGetHandler(w, makeJSONGetRequest(string(getBody)))
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing counter, got %d", w.Code)
	}
}

func TestJsonGetHandlerInvalidJSON(t *testing.T) {
	h := newHandler()

	w := httptest.NewRecorder()
	h.JSONGetHandler(w, makeJSONGetRequest("not-json"))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for invalid JSON, got %d", w.Code)
	}
}

func TestJsonGetHandlerUnknownType(t *testing.T) {
	h := newHandler()
	get := models.Metrics{ID: "latency", MType: "histogram"}
	getBody, _ := json.Marshal(get)

	w := httptest.NewRecorder()
	h.JSONGetHandler(w, makeJSONGetRequest(string(getBody)))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unknown metric type, got %d", w.Code)
	}
}

func TestJsonGetHandlerGaugeEncodeError(t *testing.T) {
	h := newHandler()
	v := 1.0
	upd := models.Metrics{ID: "cpu", MType: models.Gauge, Value: &v}
	updBody, _ := json.Marshal(upd)
	h.JSONUpdateHandler(httptest.NewRecorder(), makeJSONUpdateRequest(string(updBody)))

	get := models.Metrics{ID: "cpu", MType: models.Gauge}
	getBody, _ := json.Marshal(get)
	// errorResponseWriter forces enc.Encode to fail, covering the error branch.
	h.JSONGetHandler(&errorResponseWriter{}, makeJSONGetRequest(string(getBody)))
}

func TestJsonGetHandlerCounterEncodeError(t *testing.T) {
	h := newHandler()
	var delta int64 = 5
	upd := models.Metrics{ID: "hits", MType: models.Counter, Delta: &delta}
	updBody, _ := json.Marshal(upd)
	h.JSONUpdateHandler(httptest.NewRecorder(), makeJSONUpdateRequest(string(updBody)))

	get := models.Metrics{ID: "hits", MType: models.Counter}
	getBody, _ := json.Marshal(get)
	h.JSONGetHandler(&errorResponseWriter{}, makeJSONGetRequest(string(getBody)))
}

func TestJsonGetHandlerContentType(t *testing.T) {
	h := newHandler()
	v := 1.0
	upd := models.Metrics{ID: "cpu", MType: models.Gauge, Value: &v}
	updBody, _ := json.Marshal(upd)
	h.JSONUpdateHandler(httptest.NewRecorder(), makeJSONUpdateRequest(string(updBody)))

	get := models.Metrics{ID: "cpu", MType: models.Gauge}
	getBody, _ := json.Marshal(get)
	w := httptest.NewRecorder()
	h.JSONGetHandler(w, makeJSONGetRequest(string(getBody)))

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
}
