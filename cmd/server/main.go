package main

import (
	"net/http"
	"time"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/config"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/repository"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	cfg := config.NewServerConfig()

	r := chi.NewRouter()
	storage := repository.NewMemStorage()
	ms := service.NewMetricsServer(storage)

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/", ms.ListMetricsHandler)
	r.Get("/value/{kind}/{name}", ms.GetMetricHandler)
	r.Post("/update/{kind}/{name}/{value}", ms.UpdateHandler)

	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		panic(err)
	}
}
