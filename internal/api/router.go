package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Handler returns the root HTTP handler (router + middleware chain).
func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(s.RequestTimeout))
	r.Use(s.requestLog())

	r.Get("/product", s.ListProducts)
	r.Get("/product/{productId}", s.GetProduct)
	r.Post("/order", s.PlaceOrder)
	r.Get("/healthz", s.Health)

	return r
}
