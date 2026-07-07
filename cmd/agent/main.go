package main

import (
	"log"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/agent"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/config"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/logger"
)

func main() {
	cfg, err := config.NewAgentConfig()
	if err != nil {
		log.Fatal(err)
	}
	if err := logger.Initialize("debug"); err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	a := agent.New(*cfg)
	a.Run()
}
