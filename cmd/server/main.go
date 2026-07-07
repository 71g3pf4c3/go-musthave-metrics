package main

import (
	"context"
	"log"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/config"
)

func main() {
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
