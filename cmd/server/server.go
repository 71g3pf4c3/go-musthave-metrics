package main

import (
	"context"
	"crypto/ecdh"
	"errors"
	"net"
	"net/http"
	"time"

	"google.golang.org/grpc"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/audit"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/config"
	servercrypto "github.com/71g3pf4c3/go-musthave-metrics/internal/crypto"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/grpcserver"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/handlers"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/logger"
	pb "github.com/71g3pf4c3/go-musthave-metrics/internal/proto"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/repository"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/service"
)

// serverApp bundles the running components of the server so main can start
// and gracefully stop them without juggling multiple return values.
type serverApp struct {
	http     *http.Server
	grpc     *grpc.Server
	grpcAddr string
	cleanup  *serverCleanup
}

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

func newServer(ctx context.Context, cfg *config.ServerConfig) (*serverApp, error) {
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
			return nil, err
		}
		logger.Sugar.Infof("X25519 ECDH decryption enabled")
	}

	var trustedSubnet *net.IPNet
	if cfg.TrustedSubnet != "" {
		_, subnet, err := net.ParseCIDR(cfg.TrustedSubnet)
		if err != nil {
			return nil, err
		}
		trustedSubnet = subnet
		logger.Sugar.Infof("trusted subnet filtering enabled: %s", cfg.TrustedSubnet)
	}

	router := newRouter(h, cfg.Key, privKey, trustedSubnet)

	var grpcSrv *grpc.Server
	if cfg.GRPCAddress != "" {
		var opts []grpc.ServerOption
		if trustedSubnet != nil {
			opts = append(opts, grpc.ChainUnaryInterceptor(grpcserver.TrustedSubnetInterceptor(trustedSubnet)))
		}
		grpcSrv = grpc.NewServer(opts...)
		pb.RegisterMetricsServer(grpcSrv, grpcserver.New(svc))
		logger.Sugar.Infof("gRPC server enabled on %s", cfg.GRPCAddress)
	}

	logger.Sugar.Infof("starting server on %s", cfg.Address)
	return &serverApp{
		http: &http.Server{
			Addr:         cfg.Address,
			Handler:      router,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		grpc:     grpcSrv,
		grpcAddr: cfg.GRPCAddress,
		cleanup:  cleanup,
	}, nil
}

// Run starts the HTTP and (optional) gRPC servers. Any fatal serving error is
// sent to errc; the caller is responsible for triggering shutdown.
func (a *serverApp) Run(errc chan<- error) {
	go func() {
		if err := a.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errc <- err
		}
	}()

	if a.grpc != nil {
		go func() {
			lis, err := net.Listen("tcp", a.grpcAddr)
			if err != nil {
				errc <- err
				return
			}
			if err := a.grpc.Serve(lis); err != nil {
				errc <- err
			}
		}()
	}
}

// Shutdown gracefully stops all running components.
func (a *serverApp) Shutdown(ctx context.Context) {
	if err := a.http.Shutdown(ctx); err != nil {
		logger.Sugar.Errorf("server shutdown error: %v", err)
	}
	if a.grpc != nil {
		a.grpc.GracefulStop()
	}
	a.cleanup.Shutdown(ctx)
}
