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
	"github.com/71g3pf4c3/go-musthave-metrics/internal/service"
)

// errorResponseWriter is an http.ResponseWriter whose Write always fails.
// It lets tests exercise the enc.Encode error branches in JSONGetHandler.
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

func newHandler() *handlers.MetricsHandler {
	return handlers.NewMetricsHandler(service.New(repository.NewMemStorage()), nil)
}

func makeRequest(kind, name, value string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/update/"+kind+"/"+name+"/"+value, nil)
	r.Header.Set("Content-Type", "text/plain")
	r.SetPathValue("kind", kind)
	r.SetPathValue("name", name)
	r.SetPathValue("value", value)
	return r
}

func makeGetRequest(kind, name string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/value/"+kind+"/"+name, nil)
	r.SetPathValue("kind", kind)
	r.SetPathValue("name", name)
	return r
}

func makeJSONUpdateRequest(body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func makeJSONGetRequest(body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return r
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
	s := newHandler()
	s.UpdateHandler(httptest.NewRecorder(), makeRequest("gauge", "cpu", "42.5"))

	w := httptest.NewRecorder()
	s.ListHandler(w, httptest.NewRequest(http.MethodGet, "/", nil))
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
	s := newHandler()
	s.UpdateHandler(httptest.NewRecorder(), makeRequest("counter", "hits", "10"))
	s.UpdateHandler(httptest.NewRecorder(), makeRequest("counter", "hits", "5"))

	w := httptest.NewRecorder()
	s.ListHandler(w, httptest.NewRequest(http.MethodGet, "/", nil))
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
	s := newHandler()
	s.UpdateHandler(httptest.NewRecorder(), makeRequest("gauge", "mem", "1.5"))
	s.UpdateHandler(httptest.NewRecorder(), makeRequest("counter", "reqs", "7"))

	w := httptest.NewRecorder()
	s.ListHandler(w, httptest.NewRequest(http.MethodGet, "/", nil))
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
	s := newHandler()
	s.UpdateHandler(httptest.NewRecorder(), makeRequest("gauge", "cpu", "42.5"))

	w := httptest.NewRecorder()
	s.GetHandler(w, makeGetRequest("gauge", "cpu"))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "42.5" {
		t.Errorf("expected body \"42.5\", got %q", w.Body.String())
	}
}

func TestGetHandlerCounter(t *testing.T) {
	s := newHandler()
	s.UpdateHandler(httptest.NewRecorder(), makeRequest("counter", "hits", "10"))
	s.UpdateHandler(httptest.NewRecorder(), makeRequest("counter", "hits", "5"))

	w := httptest.NewRecorder()
	s.GetHandler(w, makeGetRequest("counter", "hits"))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "15" {
		t.Errorf("expected body \"15\", got %q", w.Body.String())
	}
}

func TestGetHandlerNotFound(t *testing.T) {
	s := newHandler()

	w := httptest.NewRecorder()
	s.GetHandler(w, makeGetRequest("gauge", "nonexistent"))
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetHandlerUnknownKind(t *testing.T) {
	s := newHandler()

	w := httptest.NewRecorder()
	s.GetHandler(w, makeGetRequest("histogram", "cpu"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetHandlerMissingName(t *testing.T) {
	s := newHandler()

	w := httptest.NewRecorder()
	s.GetHandler(w, makeGetRequest("gauge", ""))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
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
	r := httptest.NewRequest(http.MethodPost, "/update/gauge//42.5", nil)
	r.SetPathValue("kind", "gauge")
	r.SetPathValue("name", "")
	r.SetPathValue("value", "42.5")
	w := httptest.NewRecorder()
	newHandler().UpdateHandler(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty name, got %d", w.Code)
	}
}

func TestUpdateHandlerEmptyValue(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/update/gauge/cpu/", nil)
	r.SetPathValue("kind", "gauge")
	r.SetPathValue("name", "cpu")
	r.SetPathValue("value", "")
	w := httptest.NewRecorder()
	newHandler().UpdateHandler(w, r)
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

// --- JSONUpdateHandler ---

func TestJsonUpdateHandlerGauge(t *testing.T) {
	s := newHandler()
	v := 42.5
	m := models.Metrics{ID: "cpu", MType: models.Gauge, Value: &v}
	body, _ := json.Marshal(m)

	w := httptest.NewRecorder()
	s.JSONUpdateHandler(w, makeJSONUpdateRequest(string(body)))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	rw := httptest.NewRecorder()
	s.GetHandler(rw, makeGetRequest("gauge", "cpu"))
	if rw.Body.String() != "42.5" {
		t.Errorf("expected stored gauge \"42.5\", got %q", rw.Body.String())
	}
}

func TestJsonUpdateHandlerCounter(t *testing.T) {
	s := newHandler()
	var delta int64 = 10
	m := models.Metrics{ID: "hits", MType: models.Counter, Delta: &delta}
	body, _ := json.Marshal(m)

	w := httptest.NewRecorder()
	s.JSONUpdateHandler(w, makeJSONUpdateRequest(string(body)))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	rw := httptest.NewRecorder()
	s.GetHandler(rw, makeGetRequest("counter", "hits"))
	if rw.Body.String() != "10" {
		t.Errorf("expected stored counter \"10\", got %q", rw.Body.String())
	}
}

func TestJsonUpdateHandlerCounterAccumulates(t *testing.T) {
	s := newHandler()

	for _, delta := range []int64{10, 5} {
		d := delta
		m := models.Metrics{ID: "hits", MType: models.Counter, Delta: &d}
		body, _ := json.Marshal(m)
		s.JSONUpdateHandler(httptest.NewRecorder(), makeJSONUpdateRequest(string(body)))
	}

	rw := httptest.NewRecorder()
	s.GetHandler(rw, makeGetRequest("counter", "hits"))
	if rw.Body.String() != "15" {
		t.Errorf("expected accumulated counter \"15\", got %q", rw.Body.String())
	}
}

func TestJsonUpdateHandlerGaugeOverrides(t *testing.T) {
	s := newHandler()

	for _, v := range []float64{1.0, 99.9} {
		val := v
		m := models.Metrics{ID: "cpu", MType: models.Gauge, Value: &val}
		body, _ := json.Marshal(m)
		s.JSONUpdateHandler(httptest.NewRecorder(), makeJSONUpdateRequest(string(body)))
	}

	rw := httptest.NewRecorder()
	s.GetHandler(rw, makeGetRequest("gauge", "cpu"))
	if rw.Body.String() != "99.9" {
		t.Errorf("expected overwritten gauge \"99.9\", got %q", rw.Body.String())
	}
}

func TestJsonUpdateHandlerInvalidJSON(t *testing.T) {
	s := newHandler()

	w := httptest.NewRecorder()
	s.JSONUpdateHandler(w, makeJSONUpdateRequest("not-json"))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for invalid JSON, got %d", w.Code)
	}
}

func TestJsonUpdateHandlerUnknownType(t *testing.T) {
	s := newHandler()
	m := models.Metrics{ID: "latency", MType: "histogram"}
	body, _ := json.Marshal(m)

	w := httptest.NewRecorder()
	s.JSONUpdateHandler(w, makeJSONUpdateRequest(string(body)))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unknown metric type, got %d", w.Code)
	}
}

// --- JSONGetHandler ---

func TestJsonGetHandlerGauge(t *testing.T) {
	s := newHandler()
	v := 42.5
	upd := models.Metrics{ID: "cpu", MType: models.Gauge, Value: &v}
	updBody, _ := json.Marshal(upd)
	s.JSONUpdateHandler(httptest.NewRecorder(), makeJSONUpdateRequest(string(updBody)))

	get := models.Metrics{ID: "cpu", MType: models.Gauge}
	getBody, _ := json.Marshal(get)
	w := httptest.NewRecorder()
	s.JSONGetHandler(w, makeJSONGetRequest(string(getBody)))

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
	s := newHandler()
	var delta int64 = 7
	upd := models.Metrics{ID: "hits", MType: models.Counter, Delta: &delta}
	updBody, _ := json.Marshal(upd)
	s.JSONUpdateHandler(httptest.NewRecorder(), makeJSONUpdateRequest(string(updBody)))

	get := models.Metrics{ID: "hits", MType: models.Counter}
	getBody, _ := json.Marshal(get)
	w := httptest.NewRecorder()
	s.JSONGetHandler(w, makeJSONGetRequest(string(getBody)))

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
	s := newHandler()
	get := models.Metrics{ID: "nonexistent", MType: models.Gauge}
	getBody, _ := json.Marshal(get)

	w := httptest.NewRecorder()
	s.JSONGetHandler(w, makeJSONGetRequest(string(getBody)))
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing gauge, got %d", w.Code)
	}
}

func TestJsonGetHandlerCounterNotFound(t *testing.T) {
	s := newHandler()
	get := models.Metrics{ID: "nonexistent", MType: models.Counter}
	getBody, _ := json.Marshal(get)

	w := httptest.NewRecorder()
	s.JSONGetHandler(w, makeJSONGetRequest(string(getBody)))
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing counter, got %d", w.Code)
	}
}

func TestJsonGetHandlerInvalidJSON(t *testing.T) {
	s := newHandler()

	w := httptest.NewRecorder()
	s.JSONGetHandler(w, makeJSONGetRequest("not-json"))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for invalid JSON, got %d", w.Code)
	}
}

func TestJsonGetHandlerUnknownType(t *testing.T) {
	s := newHandler()
	get := models.Metrics{ID: "latency", MType: "histogram"}
	getBody, _ := json.Marshal(get)

	w := httptest.NewRecorder()
	s.JSONGetHandler(w, makeJSONGetRequest(string(getBody)))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unknown metric type, got %d", w.Code)
	}
}

func TestJsonGetHandlerGaugeEncodeError(t *testing.T) {
	s := newHandler()
	v := 1.0
	upd := models.Metrics{ID: "cpu", MType: models.Gauge, Value: &v}
	updBody, _ := json.Marshal(upd)
	s.JSONUpdateHandler(httptest.NewRecorder(), makeJSONUpdateRequest(string(updBody)))

	get := models.Metrics{ID: "cpu", MType: models.Gauge}
	getBody, _ := json.Marshal(get)
	s.JSONGetHandler(&errorResponseWriter{}, makeJSONGetRequest(string(getBody)))
}

func TestJsonGetHandlerCounterEncodeError(t *testing.T) {
	s := newHandler()
	var delta int64 = 5
	upd := models.Metrics{ID: "hits", MType: models.Counter, Delta: &delta}
	updBody, _ := json.Marshal(upd)
	s.JSONUpdateHandler(httptest.NewRecorder(), makeJSONUpdateRequest(string(updBody)))

	get := models.Metrics{ID: "hits", MType: models.Counter}
	getBody, _ := json.Marshal(get)
	s.JSONGetHandler(&errorResponseWriter{}, makeJSONGetRequest(string(getBody)))
}

func TestJsonGetHandlerContentType(t *testing.T) {
	s := newHandler()
	v := 1.0
	upd := models.Metrics{ID: "cpu", MType: models.Gauge, Value: &v}
	updBody, _ := json.Marshal(upd)
	s.JSONUpdateHandler(httptest.NewRecorder(), makeJSONUpdateRequest(string(updBody)))

	get := models.Metrics{ID: "cpu", MType: models.Gauge}
	getBody, _ := json.Marshal(get)
	w := httptest.NewRecorder()
	s.JSONGetHandler(w, makeJSONGetRequest(string(getBody)))

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
}

func TestBatchUpdateHandlerSuccess(t *testing.T) {
	h := newHandler()
	v := 42.5
	delta := int64(10)
	batch := []models.Metrics{
		{ID: "cpu", MType: models.Gauge, Value: &v},
		{ID: "hits", MType: models.Counter, Delta: &delta},
	}
	body, _ := json.Marshal(batch)

	r := httptest.NewRequest(http.MethodPost, "/updates", strings.NewReader(string(body)))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.BatchUpdateHandler(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	rw := httptest.NewRecorder()
	h.GetHandler(rw, makeGetRequest("gauge", "cpu"))
	if rw.Body.String() != "42.5" {
		t.Fatalf("expected gauge 42.5, got %q", rw.Body.String())
	}
}

func TestBatchUpdateHandlerEmptyBatch(t *testing.T) {
	h := newHandler()
	r := httptest.NewRequest(http.MethodPost, "/updates", strings.NewReader("[]"))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.BatchUpdateHandler(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestBatchUpdateHandlerBadRequest(t *testing.T) {
	h := newHandler()
	invalid := `[ {"id":"cpu", "type":"gauge"} ]`
	r := httptest.NewRequest(http.MethodPost, "/updates", strings.NewReader(invalid))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.BatchUpdateHandler(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
