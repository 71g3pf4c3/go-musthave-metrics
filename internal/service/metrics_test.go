package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	repository "github.com/71g3pf4c3/go-musthave-metrics/internal/repository"
	metrics "github.com/71g3pf4c3/go-musthave-metrics/internal/service"
)

func newHandler() *metrics.MetricsServer {
	return metrics.NewMetricsServer(repository.NewMemStorage())
}

func makeRequest(kind, name, value string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/update/"+kind+"/"+name+"/"+value, nil)
	req.Header.Set("Content-Type", "text/plain")
	req.SetPathValue("kind", kind)
	req.SetPathValue("name", name)
	req.SetPathValue("value", value)
	return req
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

// --- gauge: invalid values ---

func TestUpdateHandlerAddGaugeString(t *testing.T) {
	w := httptest.NewRecorder()
	newHandler().UpdateHandler(w, makeRequest("gauge", "cpu", "abc"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for string gauge value, got %d", w.Code)
	}
}
