package main

import (
	"net/http"

	handlers "github.com/71g3pf4c3/go-musthave-metrics/internal/handler"
	_ "github.com/71g3pf4c3/go-musthave-metrics/internal/model"
	repository "github.com/71g3pf4c3/go-musthave-metrics/internal/repository"
)

const (
	ServerPort    = "8080"
	ServerAddress = "localhost"
)

func main() {
	mux := http.NewServeMux()
	storage := repository.NewMemStorage()
	metricsServer := handlers.NewMetricsServer(storage)
	mux.HandleFunc(`POST /update/{kind}/{name}/{value}`, metricsServer.UpdateHandler)
	err := http.ListenAndServe(ServerAddress+":"+ServerPort, mux)
	if err != nil {
		panic(err)
	}
}
