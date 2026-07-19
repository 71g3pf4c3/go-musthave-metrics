package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/agent"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/config"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/logger"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/version"
)

func main() {
	version.Print()
	cfg, err := config.NewAgentConfig()
	if err != nil {
		log.Fatal(err)
	}
	if err := logger.Initialize("debug"); err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	a, err := agent.New(*cfg)
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer stop()

	a.Run(ctx)
}
