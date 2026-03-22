package server

import (
	"log"
	"net/http"
	"time"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/compress"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/config"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/handlers"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/logger"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/repository"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func New(cfg *config.ServerConfig) *http.Server {
	storage := repository.NewMemStorage()
	ms := handlers.NewMetricsServer(storage)

	r := chi.NewRouter()

	if err := logger.Initialize(cfg.LogLevel); err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}

	r.Use(logger.RequestLogger)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(compress.CompressMiddleware)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Use(middleware.StripSlashes)

	r.Get("/", ms.ListHandler)
	r.Get("/value/{kind}/{name}", ms.GetHandler)
	r.Post("/update/{kind}/{name}/{value}", ms.UpdateHandler)
	r.Post("/update", ms.JSONUpdateHandler)
	r.Post("/value", ms.JSONGetHandler)

	return &http.Server{
		Addr:         cfg.Address,
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}
