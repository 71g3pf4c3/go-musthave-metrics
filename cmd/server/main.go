package main

import (
	"net/http"
	"time"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/repository"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	ServerPort    = "8080"
	ServerAddress = "localhost"
)

func main() {

	r := chi.NewRouter()
	storage := repository.NewMemStorage()
	ms := service.NewMetricsServer(storage)

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/", ms.ListMetricsHandler)
	r.Get("/value/{kind}/{name}", ms.GetMetricHandler)
	r.Post("/update/{kind}/{name}/{value}", ms.UpdateHandler)

	err := http.ListenAndServe(ServerAddress+":"+ServerPort, r)
	if err != nil {
		panic(err)
	}
}
