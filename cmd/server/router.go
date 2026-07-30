package main

import (
	"crypto/ecdh"
	"net"
	"net/http"
	"time"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/compress"
	servercrypto "github.com/71g3pf4c3/go-musthave-metrics/internal/crypto"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/handlers"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/ipfilter"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/logger"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/sign"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func newRouter(h *handlers.MetricsHandler, key string, privKey *ecdh.PrivateKey, trustedSubnet *net.IPNet) http.Handler {
	r := chi.NewRouter()

	r.Use(logger.RequestLogger)
	r.Use(middleware.RequestID)
	if trustedSubnet != nil {
		r.Use(ipfilter.Middleware(trustedSubnet))
	}
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	if privKey != nil {
		r.Use(servercrypto.DecryptMiddleware(privKey))
	}
	r.Use(compress.CompressMiddleware)
	if key != "" {
		r.Use(sign.Middleware(key))
	}
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(middleware.StripSlashes)

	r.Get("/", h.ListHandler)
	r.Get("/value/{kind}/{name}", h.GetHandler)
	r.Post("/update/{kind}/{name}/{value}", h.UpdateHandler)
	r.Post("/update", h.JSONUpdateHandler)
	r.Post("/updates", h.BatchUpdateHandler)
	r.Get("/readyz", h.ReadyzHandler)
	r.Get("/healthz", h.HealthzHandler)
	r.Get("/ping", h.PingHandler)
	r.Post("/value", h.JSONGetHandler)

	return r
}
