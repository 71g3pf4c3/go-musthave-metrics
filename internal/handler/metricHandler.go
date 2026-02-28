package handler

import (
	"net/http"
	"strconv"

	models "github.com/71g3pf4c3/go-musthave-metrics/internal/model"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/storage"
)

type Handler struct {
	store *storage.MemStorage
}

func New(s *storage.MemStorage) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Metrics(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "text/plain" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("unsupported content type"))
		return
	}

	kind := r.PathValue("kind")
	name := r.PathValue("name")
	value := r.PathValue("value")

	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if value == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch kind {
	case models.Gauge:
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		h.store.SetGauge(name, v)
	case models.Counter:
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		h.store.AddCounter(name, v)
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}
