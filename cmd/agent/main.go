package main

import (
	"fmt"
	"log"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/agent"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/config"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/logger"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func printBuildInfo() {
	na := func(s string) string {
		if s == "" {
			return "N/A"
		}
		return s
	}
	fmt.Printf("Build version: %s\n", na(buildVersion))
	fmt.Printf("Build date: %s\n", na(buildDate))
	fmt.Printf("Build commit: %s\n", na(buildCommit))
}

func main() {
	printBuildInfo()
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
