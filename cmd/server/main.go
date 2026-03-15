package main

import (
	"github.com/71g3pf4c3/go-musthave-metrics/internal/config"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/server"
)

func main() {
	cfg := config.NewServerConfig()
	srv := server.New(cfg)

	if err := srv.ListenAndServe(); err != nil {
		panic(err)
	}
}
