package main

import (
	"flag"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/agent"
)

func main() {
	addr := flag.String("a", "http://localhost:8080", "server endpoint address")
	pollInterval := flag.Int("p", 2, "poll interval in seconds")
	reportInterval := flag.Int("r", 2, "report interval in seconds")
	flag.Parse()

	config := agent.AgentConfig{PollInterval: *pollInterval, ReportInterval: *reportInterval}
	a := agent.New(*addr, config)
	a.Run()
}
