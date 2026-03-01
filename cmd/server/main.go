package main

import (
	"net/http"
	"time"

	_ "github.com/71g3pf4c3/go-musthave-metrics/internal/model"
	repository "github.com/71g3pf4c3/go-musthave-metrics/internal/repository"
	metrics "github.com/71g3pf4c3/go-musthave-metrics/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	ServerPort    = "8080"
	ServerAddress = "localhost"
)

func main() {

	r := chi.NewRouter()
	storage := repository.NewMemStorage()
	server := metrics.NewMetricsServer(storage)

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(middleware.Timeout(60 * time.Second))

	r.Route("/update", func(r chi.Router) {
		r.Route("/{kind}/{name}", func(r chi.Router) {
			// r.Get("/", getMetric)
			r.Route("/{value}", func(r chi.Router) {
				r.Post("/", server.UpdateHandler)
			})
		})
	})

	err := http.ListenAndServe(ServerAddress+":"+ServerPort, r)
	if err != nil {
		panic(err)
	}
}
