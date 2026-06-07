package http

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/michael/nextd/internal/config"
	"github.com/michael/nextd/internal/http/handler"
	"github.com/michael/nextd/internal/http/middleware"
)

// NewRouter builds the HTTP router for the application.
func NewRouter(cfg *config.Config) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recovery)
	r.Use(middleware.Logger)
	r.Use(middleware.CORS)

	if cfg != nil {
		r.Use(middleware.NewRateLimiter(cfg.RateLimit.Requests, cfg.RateLimit.Window).Middleware)
	}

	r.Get("/health", handler.Health)
	r.Get("/healthz", handler.Health)

	if _, err := os.Stat("web/static"); err == nil {
		r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	}

	return r
}
