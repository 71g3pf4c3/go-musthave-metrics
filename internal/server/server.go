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
	if err := logger.Initialize(cfg.LogLevel); err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}

	storage := repository.NewMemStorage()
	ms := handlers.NewMetricsServer(storage)

	r := chi.NewRouter()

	if cfg.RestoreFlag {
		if err := ms.Restore(cfg.FileStoragePath); err != nil {
			logger.Sugar.Infof("restore from %s: %v", cfg.FileStoragePath, err)
		}
	}

	if cfg.StoreInterval > 0 {
		logger.Sugar.Infof("periodic dump enabled every %ds to %s", cfg.StoreInterval, cfg.FileStoragePath)
		dumpTicker := time.NewTicker(time.Duration(cfg.StoreInterval) * time.Second)
		go func() {
			for range dumpTicker.C {
				if err := ms.Dump(cfg.FileStoragePath); err != nil {
					logger.Sugar.Errorf("failed to dump data: %v", err)
				}
			}
		}()
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

	logger.Sugar.Infof("starting server on %s", cfg.Address)
	return &http.Server{
		Addr:         cfg.Address,
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}
