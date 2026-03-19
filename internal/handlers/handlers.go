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
	repository *repository.MemStorage
}

func NewMetricsServer(ms *repository.MemStorage) *MetricsServer {
	return &MetricsServer{repository: ms}
}

func MainPageHandler(res http.ResponseWriter, req *http.Request) {
	res.Write([]byte("Welcome to go-musthave-metrics!"))
}

func (ms *MetricsServer) ListHandler(res http.ResponseWriter, req *http.Request) {
	gauges := ms.repository.GetAllGauge()
	counters := ms.repository.GetAllCounter()

	body := "<html><body><ul>"
	for name, value := range gauges {
		body += fmt.Sprintf("<li>%s = %s</li>", name, strconv.FormatFloat(value, 'f', -1, 64))
	}
	for name, value := range counters {
		body += fmt.Sprintf("<li>%s = %s</li>", name, strconv.FormatInt(value, 10))
	}
	body += "</ul></body></html>"

	res.Header().Set("Content-Type", "text/html")
	res.WriteHeader(http.StatusOK)
	res.Write([]byte(body))
}

func (ms *MetricsServer) GetHandler(res http.ResponseWriter, req *http.Request) {

	name := req.PathValue("name")
	kind := req.PathValue("kind")

	if name == "" {
		res.WriteHeader(http.StatusNotFound)
		return
	}
	v, err := ms.repository.GetValue(name, kind)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			res.WriteHeader(http.StatusNotFound)
		} else {
			res.WriteHeader(http.StatusBadRequest)
		}
		return
	}
	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusOK)
	res.Write([]byte(v))
}

func (ms *MetricsServer) UpdateHandler(res http.ResponseWriter, req *http.Request) {
	kind := req.PathValue("kind")
	name := req.PathValue("name")
	value := req.PathValue("value")

	if name == "" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	if value == "" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	switch kind {
	case models.Gauge:
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		ms.repository.SetGauge(name, v)
	case models.Counter:
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		ms.repository.AddCounter(name, v)
	default:
		res.WriteHeader(http.StatusBadRequest)
		return
	}

}

func (ms *MetricsServer) JsonUpdateHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		logger.Sugar.Debug("got request with bad method", zap.String("method", req.Method))
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	logger.Sugar.Debug("decoding request")
	var metric models.Metrics
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&metric); err != nil {
		logger.Sugar.Debug("cannot decode request JSON body", zap.Error(err))
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	switch metric.MType {
	case models.Gauge:
		ms.repository.SetGauge(metric.ID, *metric.Value)
	case models.Counter:
		ms.repository.AddCounter(metric.ID, *metric.Delta)
	default:
		res.WriteHeader(http.StatusBadRequest)
		logger.Sugar.Debug("unsupported request type", zap.String("type", metric.MType))
		res.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

}

func (ms *MetricsServer) JsonGetHandler(res http.ResponseWriter, req *http.Request) {

	if req.Method != http.MethodGet {
		logger.Sugar.Debug("got request with bad method", zap.String("method", req.Method))
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	logger.Sugar.Debug("decoding request")
	var metric models.Metrics
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&metric); err != nil {
		logger.Sugar.Debug("cannot decode request JSON body", zap.Error(err))
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")

	enc := json.NewEncoder(res)
	switch metric.MType {
	case models.Gauge:
		gaugeValue, err := ms.repository.GetGauge(metric.ID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				logger.Sugar.Debug("metric not found", zap.Error(err))
				res.WriteHeader(http.StatusNotFound)
			} else {
				logger.Sugar.Debug("bad request", zap.Error(err))
				res.WriteHeader(http.StatusBadRequest)
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
				res.WriteHeader(http.StatusNotFound)
			} else {
				logger.Sugar.Debug("bad request", zap.Error(err))
				res.WriteHeader(http.StatusBadRequest)
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
		res.WriteHeader(http.StatusBadRequest)
		logger.Sugar.Debug("unsupported request type", zap.String("type", metric.MType))
		res.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	res.WriteHeader(http.StatusOK)
}
