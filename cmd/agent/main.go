package main

import (
	"github.com/71g3pf4c3/go-musthave-metrics/internal/agent"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/config"
)

func main() {
	cfg := config.NewAgentConfig()
	a := agent.New(*cfg)
	a.Run()
}
