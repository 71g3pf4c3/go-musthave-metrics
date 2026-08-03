package main

import (
	"context"
	"log"
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

	app, err := newServer(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}

	serverErr := make(chan error, 1)
	app.Run(serverErr)

	// Wait for signal or server error.
	var startupErr error
	select {
	case <-ctx.Done():
		logger.Sugar.Infof("shutting down server...")
	case startupErr = <-serverErr:
		logger.Sugar.Errorf("server error, shutting down: %v", startupErr)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	app.Shutdown(shutdownCtx)

	logger.Sugar.Infof("server stopped")

	if startupErr != nil {
		log.Fatal(startupErr)
	}
}
