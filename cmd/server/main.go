package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/config"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/logger"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/version"
)

func main() {
	version.Print()
	cfg, err := config.NewServerConfig()
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer stop()

	srv, cleanup, err := newServer(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}

	serverErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Wait for signal or server error.
	select {
	case <-ctx.Done():
		logger.Sugar.Infof("shutting down server...")
	case err := <-serverErr:
		log.Fatal(err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Sugar.Errorf("server shutdown error: %v", err)
	}

	cleanup.Shutdown(shutdownCtx)

	logger.Sugar.Infof("server stopped")
}
