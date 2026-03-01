package main

import (
	"github.com/71g3pf4c3/go-musthave-metrics/internal/agent"
)

const (
	PollInterval   = 2
	ReportInterval = 10
)

func main() {
	config := agent.AgentConfig{PollInterval: 2, ReportInterval: 10}
	a := agent.New("http://localhost:8080", config)
	a.Run()
}
