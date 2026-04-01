package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/config"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/handlers"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/logger"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/repository"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/service"
)

func newServer(cfg *config.ServerConfig) *http.Server {
	if err := logger.Initialize(cfg.LogLevel); err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}

	var repo repository.Repository
	useFileStorage := cfg.FileStoragePath != ""
	useDBStorage := cfg.DatabaseDSN != ""

	if cfg.DatabaseDSN != "" {
		pgStore, err := repository.NewPGStorage(cfg.DatabaseDSN)
		if err != nil {
			log.Fatalf("failed to connect to database: %v", err)
		}
		repo = pgStore
	} else {
		repo = repository.NewMemStorage()
	}

	svc := service.New(repo)
	h := handlers.NewMetricsHandler(svc)

	if !useDBStorage && useFileStorage && cfg.RestoreFlag {
		if err := svc.Restore(context.Background(), cfg.FileStoragePath); err != nil {
			logger.Sugar.Infof("restore from %s: %v", cfg.FileStoragePath, err)
		}
	}

	if !useDBStorage && useFileStorage && cfg.StoreInterval > 0 {
		logger.Sugar.Infof("periodic dump enabled every %ds to %s", cfg.StoreInterval, cfg.FileStoragePath)
		dumpTicker := time.NewTicker(time.Duration(cfg.StoreInterval) * time.Second)
		go func() {
			for range dumpTicker.C {
				if err := svc.Dump(context.Background(), cfg.FileStoragePath); err != nil {
					logger.Sugar.Errorf("failed to dump data: %v", err)
				}
			}
		}()
	}

	router := newRouter(h)

	logger.Sugar.Infof("starting server on %s", cfg.Address)
	return &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}
