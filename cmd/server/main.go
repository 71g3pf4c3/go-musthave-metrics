package main

import (
	"net/http"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/handler"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/storage"
)

func main() {
	store := storage.New()
	h := handler.New(store)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /update/{kind}/{name}/{value}", h.Metrics)

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
