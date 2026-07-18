package main

import (
	"context"
	"crypto/ecdh"
	"net/http"
	"time"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/audit"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/config"
	servercrypto "github.com/71g3pf4c3/go-musthave-metrics/internal/crypto"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/handlers"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/logger"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/repository"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/service"
)

func newServer(ctx context.Context, cfg *config.ServerConfig) (*http.Server, error) {
	if err := logger.Initialize(cfg.LogLevel); err != nil {
		return nil, err
	}

	var repo repository.Repository
	useFileStorage := cfg.FileStoragePath != ""
	useDBStorage := cfg.DatabaseDSN != ""

	if cfg.DatabaseDSN != "" {
		pgStore, err := repository.NewPGStorage(cfg.DatabaseDSN)
		if err != nil {
			return nil, err
		}
		repo = pgStore
	} else {
		repo = repository.NewMemStorage()
	}

	svc := service.New(repo)

	var auditObservers []audit.Observer
	if cfg.AuditFile != "" {
		fo, err := audit.NewFileObserver(cfg.AuditFile)
		if err != nil {
			return nil, err
		}
		auditObservers = append(auditObservers, fo)
		logger.Sugar.Infof("audit file sink enabled: %s", cfg.AuditFile)
	}
	if cfg.AuditURL != "" {
		auditObservers = append(auditObservers, audit.NewHTTPObserver(cfg.AuditURL))
		logger.Sugar.Infof("audit http sink enabled: %s", cfg.AuditURL)
	}
	notifier := audit.NewNotifier(auditObservers...)

	h := handlers.NewMetricsHandler(svc, notifier)

	if !useDBStorage && useFileStorage && cfg.RestoreFlag {
		if err := svc.Restore(ctx, cfg.FileStoragePath); err != nil {
			logger.Sugar.Infof("restore from %s: %v", cfg.FileStoragePath, err)
		}
	}

	if !useDBStorage && useFileStorage && cfg.StoreInterval > 0 {
		logger.Sugar.Infof("periodic dump enabled every %ds to %s", cfg.StoreInterval, cfg.FileStoragePath)
		dumpTicker := time.NewTicker(time.Duration(cfg.StoreInterval) * time.Second)
		go func() {
			for range dumpTicker.C {
				if err := svc.Dump(ctx, cfg.FileStoragePath); err != nil {
					logger.Sugar.Errorf("failed to dump data: %v", err)
				}
			}
		}()
	}

	var privKey *ecdh.PrivateKey
	if cfg.CryptoKey != "" {
		var err error
		privKey, err = servercrypto.LoadPrivateKey(cfg.CryptoKey)
		if err != nil {
			return nil, err
		}
		logger.Sugar.Infof("X25519 ECDH decryption enabled")
	}

	router := newRouter(h, cfg.Key, privKey)

	logger.Sugar.Infof("starting server on %s", cfg.Address)
	return &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}, nil
}
