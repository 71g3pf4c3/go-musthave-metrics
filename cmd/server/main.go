package main

import (
	"context"
	"fmt"
	"log"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/config"
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
	cfg, err := config.NewServerConfig()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	srv, err := newServer(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
