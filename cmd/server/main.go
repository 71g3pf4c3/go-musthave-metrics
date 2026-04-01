package main

import (
	"github.com/71g3pf4c3/go-musthave-metrics/internal/config"
)

func main() {
	cfg := config.NewServerConfig()
	srv := newServer(cfg)

	if err := srv.ListenAndServe(); err != nil {
		panic(err)
	}
}
