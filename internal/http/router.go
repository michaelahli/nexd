package http

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/michael/nextd/internal/auth"
	"github.com/michael/nextd/internal/config"
	"github.com/michael/nextd/internal/http/handler"
	"github.com/michael/nextd/internal/http/middleware"
)

// Options contains optional router dependencies.
type Options struct {
	Users  *auth.UserStore
	Tokens *auth.TokenManager
}

// NewRouter builds the HTTP router for the application.
func NewRouter(cfg *config.Config, opts Options) http.Handler {
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

	if opts.Users != nil && opts.Tokens != nil {
		authHandler := handler.NewAuth(opts.Users, opts.Tokens)
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)
		})
	}

	if _, err := os.Stat("web/static"); err == nil {
		r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	}

	return r
}
