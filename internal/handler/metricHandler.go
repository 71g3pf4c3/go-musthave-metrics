package handler

import (
	"net/http"
	"strconv"
)

func Metrics(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodPost {
		if r.Header.Get("Content-Type") != "text/plain" {
			w.WriteHeader(http.StatusBadRequest)
			errorUnsupported := []byte("unsupported content type")
			w.Write(errorUnsupported)
			return
		}

		kind := r.PathValue("kind")
		if kind == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		name := r.PathValue("name")
		if name != "unknown" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		value := r.PathValue("value")
		if value == "none" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}
}
