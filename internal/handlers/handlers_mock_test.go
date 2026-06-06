package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/handlers"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/mocks"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/models"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/repository"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/service"
	"github.com/golang/mock/gomock"
)

func TestGetHandlerNotFoundFromService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	h := handlers.NewMetricsHandler(mockSvc, nil)

	r := httptest.NewRequest(http.MethodGet, "/value/gauge/cpu", nil)
	r.SetPathValue("kind", models.Gauge)
	r.SetPathValue("name", "cpu")
	w := httptest.NewRecorder()

	mockSvc.EXPECT().GetValue(gomock.Any(), models.Gauge, "cpu").Return("", repository.ErrNotFound)

	h.GetHandler(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestUpdateHandlerBadRequestFromService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	h := handlers.NewMetricsHandler(mockSvc, nil)

	r := httptest.NewRequest(http.MethodPost, "/update/counter/hits/abc", nil)
	r.SetPathValue("kind", models.Counter)
	r.SetPathValue("name", "hits")
	r.SetPathValue("value", "abc")
	w := httptest.NewRecorder()

	mockSvc.EXPECT().Update(gomock.Any(), models.Counter, "hits", "abc").Return(service.ErrBadRequest)

	h.UpdateHandler(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestPingHandlerErrorFromService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	h := handlers.NewMetricsHandler(mockSvc, nil)

	r := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	mockSvc.EXPECT().Ping(gomock.Any()).Return(errors.New("db down"))

	h.PingHandler(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestJSONGetHandlerSuccessFromService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	h := handlers.NewMetricsHandler(mockSvc, nil)

	body := `{"id":"cpu","type":"gauge"}`
	r := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(body))
	w := httptest.NewRecorder()

	v := 42.5
	mockSvc.EXPECT().JSONGet(gomock.Any(), models.Metrics{ID: "cpu", MType: models.Gauge}).Return(models.Metrics{ID: "cpu", MType: models.Gauge, Value: &v}, nil)

	h.JSONGetHandler(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp models.Metrics
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Value == nil || *resp.Value != 42.5 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}
