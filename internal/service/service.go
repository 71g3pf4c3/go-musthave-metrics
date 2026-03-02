package service

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	models "github.com/71g3pf4c3/go-musthave-metrics/internal/model"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/repository"
)

type MetricsServer struct {
	repository *repository.MemStorage
}

func NewMetricsServer(ms *repository.MemStorage) *MetricsServer {
	return &MetricsServer{repository: ms}
}

func (ms *MetricsServer) ListMetricsHandler(res http.ResponseWriter, req *http.Request) {
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

func (ms *MetricsServer) GetMetricHandler(res http.ResponseWriter, req *http.Request) {

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
