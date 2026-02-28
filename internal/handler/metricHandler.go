package handler

import (
	"fmt"
	"net/http"
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
		if name == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		value := r.PathValue("value")
		if value == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		fmt.Sprintf("%s,%s,%s\r\n", kind, name, value)

		w.WriteHeader(http.StatusOK)
		return
	}
}
