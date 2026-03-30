package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/logger"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/models"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/repository"
	"go.uber.org/zap"
)

type MetricsServer struct {
	repository repository.Repository
}

func NewMetricsServer(repo repository.Repository) *MetricsServer {
	return &MetricsServer{repository: repo}
}

func (ms *MetricsServer) Dump(path string) error {
	err := ms.repository.Dump(path)
	return err
}

func (ms *MetricsServer) Restore(path string) error {
	err := ms.repository.Restore(path)
	return err
}

func MainPageHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Welcome to go-musthave-metrics!"))
}

func (ms *MetricsServer) ReadyzHandler(w http.ResponseWriter, r *http.Request) {
	logger.Sugar.Debug("Updating readiness probe")

	body := "ok"

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(body))
}

func (ms *MetricsServer) HealthzHandler(w http.ResponseWriter, r *http.Request) {
	logger.Sugar.Debug("Updating healthz probe")

	body := "ok"

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(body))
}

func (ms *MetricsServer) PingHandler(w http.ResponseWriter, r *http.Request) {
	logger.Sugar.Debug("Updating healthz probe")

	w.Header().Set("Content-Type", "text/plain")

	err := ms.repository.Ping()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}

}

func (ms *MetricsServer) ListHandler(w http.ResponseWriter, r *http.Request) {
	logger.Sugar.Debug("listing all metrics")
	gauges, err := ms.repository.GetAllGauge()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	counters, err := ms.repository.GetAllCounter()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	body := "<html><body><ul>"
	for name, value := range gauges {
		body += fmt.Sprintf("<li>%s = %s</li>", name, strconv.FormatFloat(value, 'f', -1, 64))
	}
	for name, value := range counters {
		body += fmt.Sprintf("<li>%s = %s</li>", name, strconv.FormatInt(value, 10))
	}
	body += "</ul></body></html>"

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(body))
}

func (ms *MetricsServer) GetHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	kind := r.PathValue("kind")
	logger.Sugar.Debugf("get metric %s/%s", kind, name)

	if name == "" {
		logger.Sugar.Debug("metric name is empty")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	v, err := ms.repository.GetValue(name, kind)
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

func (ms *MetricsServer) UpdateHandler(w http.ResponseWriter, r *http.Request) {
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

	switch kind {
	case models.Gauge:
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			logger.Sugar.Debugf("invalid gauge value %q: %v", value, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		ms.repository.SetGauge(name, v)
		logger.Sugar.Debugf("gauge %s updated to %v", name, v)
	case models.Counter:
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			logger.Sugar.Debugf("invalid counter value %q: %v", value, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		ms.repository.AddCounter(name, v)
		logger.Sugar.Debugf("counter %s incremented by %d", name, v)
	default:
		logger.Sugar.Debugf("unsupported metric type %q", kind)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

func (ms *MetricsServer) JSONUpdateHandler(w http.ResponseWriter, r *http.Request) {
	logger.Sugar.Debug("decoding request")
	var metric models.Metrics
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&metric); err != nil {
		logger.Sugar.Debug("cannot decode request JSON body", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	switch metric.MType {
	case models.Gauge:
		ms.repository.SetGauge(metric.ID, *metric.Value)
		logger.Sugar.Debugf("json update: gauge %s set to %v", metric.ID, *metric.Value)
	case models.Counter:
		ms.repository.AddCounter(metric.ID, *metric.Delta)
		logger.Sugar.Debugf("json update: counter %s incremented by %d", metric.ID, *metric.Delta)
	default:
		logger.Sugar.Debug("unsupported request type", zap.String("type", metric.MType))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (ms *MetricsServer) JSONGetHandler(w http.ResponseWriter, r *http.Request) {

	logger.Sugar.Debug("decoding request")
	var metric models.Metrics
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&metric); err != nil {
		logger.Sugar.Debug("cannot decode request JSON body", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	enc := json.NewEncoder(w)
	switch metric.MType {
	case models.Gauge:
		gaugeValue, err := ms.repository.GetGauge(metric.ID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				logger.Sugar.Debug("metric not found", zap.Error(err))
				w.WriteHeader(http.StatusNotFound)
			} else {
				logger.Sugar.Debug("bad request", zap.Error(err))
				w.WriteHeader(http.StatusBadRequest)
			}
			return
		}
		respMetric := models.Metrics{
			ID:    metric.ID,
			MType: metric.MType,
			Value: &gaugeValue,
		}
		if err := enc.Encode(respMetric); err != nil {
			logger.Sugar.Debug("error encoding response", zap.Error(err))
			return
		}
	case models.Counter:
		counterValue, err := ms.repository.GetCounter(metric.ID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				logger.Sugar.Debug("metric not found", zap.Error(err))
				w.WriteHeader(http.StatusNotFound)
			} else {
				logger.Sugar.Debug("bad request", zap.Error(err))
				w.WriteHeader(http.StatusBadRequest)
			}
			return
		}
		respMetric := models.Metrics{
			ID:    metric.ID,
			MType: metric.MType,
			Delta: &counterValue,
		}
		if err := enc.Encode(respMetric); err != nil {
			logger.Sugar.Debug("error encoding response", zap.Error(err))
			return
		}
	default:
		logger.Sugar.Debug("unsupported request type", zap.String("type", metric.MType))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}
