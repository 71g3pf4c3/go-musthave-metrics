package handlers

import (
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

func MainPageHandler(res http.ResponseWriter, req *http.Request) {
	res.Write([]byte("Welcome to go-musthave-metrics!"))
}

func (ms *MetricsServer) UpdateHandler(res http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Content-Type") != "text/plain" {
		res.WriteHeader(http.StatusBadRequest)
		res.Write([]byte("unsupported content type"))
		return
	}

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
