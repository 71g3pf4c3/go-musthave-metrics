// Package handlers contains HTTP handlers for the metrics server.
package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/audit"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/logger"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/models"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/repository"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/service"
	"go.uber.org/zap"
)

// MetricsHandler holds all HTTP handlers for the metrics API.
type MetricsHandler struct {
	service  service.Service
	notifier *audit.Notifier
}

// NewMetricsHandler creates a handler. notifier may be nil — audit is skipped.
func NewMetricsHandler(metricService service.Service, notifier *audit.Notifier) *MetricsHandler {
	return &MetricsHandler{service: metricService, notifier: notifier}
}

func (h *MetricsHandler) emitAudit(r *http.Request, names []string) {
	if h.notifier == nil {
		return
	}
	h.notifier.Notify(audit.Event{
		TS:        time.Now().Unix(),
		Metrics:   names,
		IPAddress: r.RemoteAddr,
	})
}

// MainPageHandler returns a static welcome message.
func MainPageHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Welcome to go-musthave-metrics!"))
}

// ReadyzHandler responds 200 OK — server is ready.
func (h *MetricsHandler) ReadyzHandler(w http.ResponseWriter, r *http.Request) {
	logger.Sugar.Debug("Updating readiness probe")
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// HealthzHandler responds 200 OK — process is alive.
func (h *MetricsHandler) HealthzHandler(w http.ResponseWriter, r *http.Request) {
	logger.Sugar.Debug("Updating healthz probe")
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// PingHandler checks the storage connection. 200 on success, 500 on error.
func (h *MetricsHandler) PingHandler(w http.ResponseWriter, r *http.Request) {
	logger.Sugar.Debug("ping database")
	w.Header().Set("Content-Type", "text/plain")

	if err := h.service.Ping(r.Context()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

// ListHandler returns an HTML page with all stored metrics.
func (h *MetricsHandler) ListHandler(w http.ResponseWriter, r *http.Request) {
	logger.Sugar.Debug("listing all metrics")
	gauges, counters, err := h.service.List(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var body strings.Builder
	body.WriteString("<html><body><ul>")
	for name, value := range gauges {
		body.WriteString("<li>")
		body.WriteString(name)
		body.WriteString(" = ")
		body.WriteString(strconv.FormatFloat(value, 'f', -1, 64))
		body.WriteString("</li>")
	}
	for name, value := range counters {
		body.WriteString("<li>")
		body.WriteString(name)
		body.WriteString(" = ")
		body.WriteString(strconv.FormatInt(value, 10))
		body.WriteString("</li>")
	}
	body.WriteString("</ul></body></html>")

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, body.String())
}

// GetHandler returns a single metric value as plain text.
// Route: GET /value/{kind}/{name}
func (h *MetricsHandler) GetHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	kind := r.PathValue("kind")
	logger.Sugar.Debugf("get metric %s/%s", kind, name)

	if name == "" {
		logger.Sugar.Debug("metric name is empty")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	v, err := h.service.GetValue(r.Context(), kind, name)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			logger.Sugar.Debugf("metric %s/%s not found", kind, name)
			w.WriteHeader(http.StatusNotFound)
		} else {
			logger.Sugar.Debugf("bad request for metric %s/%s: %v", kind, name, err)
			w.WriteHeader(http.StatusBadRequest)
		}
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(v))
}

// UpdateHandler stores a metric from URL path parameters.
// Route: POST /update/{kind}/{name}/{value}
func (h *MetricsHandler) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	kind := r.PathValue("kind")
	name := r.PathValue("name")
	value := r.PathValue("value")
	logger.Sugar.Debugf("update metric %s/%s=%s", kind, name, value)

	if name == "" {
		logger.Sugar.Debug("metric name is empty")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if value == "" {
		logger.Sugar.Debug("metric value is empty")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err := h.service.Update(r.Context(), kind, name, value)
	if err != nil {
		if errors.Is(err, service.ErrBadRequest) {
			logger.Sugar.Debugf("invalid update request for metric %s/%s=%s: %v", kind, name, value, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		logger.Sugar.Debugf("failed to update metric %s/%s=%s: %v", kind, name, value, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.emitAudit(r, []string{name})
}

// JSONUpdateHandler stores a single metric from a JSON request body.
// Route: POST /update
func (h *MetricsHandler) JSONUpdateHandler(w http.ResponseWriter, r *http.Request) {
	logger.Sugar.Debug("decoding request")
	var metric models.Metrics
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&metric); err != nil {
		logger.Sugar.Debug("cannot decode request JSON body", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := h.service.JSONUpdate(r.Context(), metric); err != nil {
		if errors.Is(err, service.ErrBadRequest) {
			logger.Sugar.Debug("unsupported or invalid request type", zap.String("type", metric.MType), zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		logger.Sugar.Debug("failed to update metric", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.emitAudit(r, []string{metric.ID})
	w.WriteHeader(http.StatusOK)
}

// JSONGetHandler returns a single metric as JSON, looked up by ID and type.
// Route: POST /value
func (h *MetricsHandler) JSONGetHandler(w http.ResponseWriter, r *http.Request) {
	logger.Sugar.Debug("decoding request")
	var metric models.Metrics
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&metric); err != nil {
		logger.Sugar.Debug("cannot decode request JSON body", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	respMetric, err := h.service.JSONGet(r.Context(), metric)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			logger.Sugar.Debug("metric not found", zap.Error(err))
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if errors.Is(err, service.ErrBadRequest) {
			logger.Sugar.Debug("unsupported request type", zap.String("type", metric.MType), zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		logger.Sugar.Debug("failed to get metric", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(respMetric); err != nil {
		logger.Sugar.Debug("error encoding response", zap.Error(err))
		return
	}
}

// BatchUpdateHandler stores multiple metrics from a JSON request body.
// Route: POST /updates
func (h *MetricsHandler) BatchUpdateHandler(w http.ResponseWriter, r *http.Request) {
	logger.Sugar.Debug("decoding batch request")

	var metrics []models.Metrics
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&metrics); err != nil {
		logger.Sugar.Debug("cannot decode batch JSON body", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(metrics) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	err := h.service.BatchUpdate(r.Context(), metrics)
	if err != nil {
		if errors.Is(err, service.ErrBadRequest) {
			logger.Sugar.Debug("invalid batch request", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		logger.Sugar.Debug("failed to batch update metrics", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	names := make([]string, 0, len(metrics))
	for _, m := range metrics {
		names = append(names, m.ID)
	}
	h.emitAudit(r, names)
	w.WriteHeader(http.StatusOK)
}
