package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/handlers"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/repository"
)

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

// --- ListMetricsHandler ---

func TestListMetricsHandlerEmpty(t *testing.T) {
	w := httptest.NewRecorder()
	newHandler().ListMetricsHandler(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html" {
		t.Errorf("expected Content-Type text/html, got %q", ct)
	}
}

func TestListMetricsHandlerGauge(t *testing.T) {
	h := newHandler()
	h.UpdateHandler(httptest.NewRecorder(), makeRequest("gauge", "cpu", "42.5"))

	w := httptest.NewRecorder()
	h.ListMetricsHandler(w, httptest.NewRequest(http.MethodGet, "/", nil))
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

func TestListMetricsHandlerCounter(t *testing.T) {
	h := newHandler()
	h.UpdateHandler(httptest.NewRecorder(), makeRequest("counter", "hits", "10"))
	h.UpdateHandler(httptest.NewRecorder(), makeRequest("counter", "hits", "5"))

	w := httptest.NewRecorder()
	h.ListMetricsHandler(w, httptest.NewRequest(http.MethodGet, "/", nil))
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

func TestListMetricsHandlerBoth(t *testing.T) {
	h := newHandler()
	h.UpdateHandler(httptest.NewRecorder(), makeRequest("gauge", "mem", "1.5"))
	h.UpdateHandler(httptest.NewRecorder(), makeRequest("counter", "reqs", "7"))

	w := httptest.NewRecorder()
	h.ListMetricsHandler(w, httptest.NewRequest(http.MethodGet, "/", nil))
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

// --- GetMetricHandler: GetValue via HTTP ---

func TestGetMetricHandlerGauge(t *testing.T) {
	h := newHandler()
	h.UpdateHandler(httptest.NewRecorder(), makeRequest("gauge", "cpu", "42.5"))

	w := httptest.NewRecorder()
	h.GetMetricHandler(w, makeGetRequest("gauge", "cpu"))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "42.5" {
		t.Errorf("expected body \"42.5\", got %q", w.Body.String())
	}
}

func TestGetMetricHandlerCounter(t *testing.T) {
	h := newHandler()
	h.UpdateHandler(httptest.NewRecorder(), makeRequest("counter", "hits", "10"))
	h.UpdateHandler(httptest.NewRecorder(), makeRequest("counter", "hits", "5"))

	w := httptest.NewRecorder()
	h.GetMetricHandler(w, makeGetRequest("counter", "hits"))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "15" {
		t.Errorf("expected body \"15\", got %q", w.Body.String())
	}
}

func TestGetMetricHandlerNotFound(t *testing.T) {
	h := newHandler()

	w := httptest.NewRecorder()
	h.GetMetricHandler(w, makeGetRequest("gauge", "nonexistent"))
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetMetricHandlerUnknownKind(t *testing.T) {
	h := newHandler()

	w := httptest.NewRecorder()
	h.GetMetricHandler(w, makeGetRequest("histogram", "cpu"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetMetricHandlerMissingName(t *testing.T) {
	h := newHandler()

	w := httptest.NewRecorder()
	h.GetMetricHandler(w, makeGetRequest("gauge", ""))
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
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
