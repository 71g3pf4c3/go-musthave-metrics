package main

import (
	"context"
	"crypto/ecdh"
	"net"
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

// serverCleanup holds resources that need cleanup on shutdown.
type serverCleanup struct {
	svc             *service.MetricsService
	notifier        *audit.Notifier
	dumpTicker      *time.Ticker
	fileStoragePath string
	needsDump       bool
}

// Shutdown performs graceful cleanup: stops ticker, dumps data, closes notifier.
func (c *serverCleanup) Shutdown(ctx context.Context) {
	if c.dumpTicker != nil {
		c.dumpTicker.Stop()
	}
	if c.needsDump {
		logger.Sugar.Infof("final dump to %s", c.fileStoragePath)
		if err := c.svc.Dump(ctx, c.fileStoragePath); err != nil {
			logger.Sugar.Errorf("final dump failed: %v", err)
		}
	}
	c.notifier.Close()
}

func newServer(ctx context.Context, cfg *config.ServerConfig) (*http.Server, *serverCleanup, error) {
	if err := logger.Initialize(cfg.LogLevel); err != nil {
		return nil, nil, err
	}

	var repo repository.Repository
	useFileStorage := cfg.FileStoragePath != ""
	useDBStorage := cfg.DatabaseDSN != ""

	if cfg.DatabaseDSN != "" {
		pgStore, err := repository.NewPGStorage(cfg.DatabaseDSN)
		if err != nil {
			return nil, nil, err
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
			return nil, nil, err
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

	cleanup := &serverCleanup{
		svc:             svc,
		notifier:        notifier,
		fileStoragePath: cfg.FileStoragePath,
		needsDump:       !useDBStorage && useFileStorage,
	}

	if !useDBStorage && useFileStorage && cfg.StoreInterval > 0 {
		logger.Sugar.Infof("periodic dump enabled every %ds to %s", cfg.StoreInterval, cfg.FileStoragePath)
		cleanup.dumpTicker = time.NewTicker(time.Duration(cfg.StoreInterval) * time.Second)
		go func() {
			for range cleanup.dumpTicker.C {
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
			return nil, nil, err
		}
		logger.Sugar.Infof("X25519 ECDH decryption enabled")
	}

	var trustedSubnet *net.IPNet
	if cfg.TrustedSubnet != "" {
		_, subnet, err := net.ParseCIDR(cfg.TrustedSubnet)
		if err != nil {
			return nil, nil, err
		}
		trustedSubnet = subnet
		logger.Sugar.Infof("trusted subnet filtering enabled: %s", cfg.TrustedSubnet)
	}

	router := newRouter(h, cfg.Key, privKey, trustedSubnet)

	logger.Sugar.Infof("starting server on %s", cfg.Address)
	return &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}, cleanup, nil
}
