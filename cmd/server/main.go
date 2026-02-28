package main

import (
	"net/http"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/handler"
)

func main() {

	mux := http.NewServeMux()
	mux.HandleFunc(http.MethodPost+" /update/{kind}/{name}/{value}", handler.Metrics)

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
