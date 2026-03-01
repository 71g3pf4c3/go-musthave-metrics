package main

import (
	"net/http"

	_ "github.com/71g3pf4c3/go-musthave-metrics/internal/model"
	repository "github.com/71g3pf4c3/go-musthave-metrics/internal/repository"
	metrics "github.com/71g3pf4c3/go-musthave-metrics/internal/service"
)

const (
	ServerPort    = "8080"
	ServerAddress = "localhost"
)

func main() {
	mux := http.NewServeMux()
	storage := repository.NewMemStorage()
	server := metrics.NewMetricsServer(storage)
	mux.HandleFunc(`POST /update/{kind}/{name}/{value}`, server.UpdateHandler)
	err := http.ListenAndServe(ServerAddress+":"+ServerPort, mux)
	if err != nil {
		panic(err)
	}
}
