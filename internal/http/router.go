package http

import (
	"context"
	"html/template"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/michaelahli/nexd/internal/auth"
	"github.com/michaelahli/nexd/internal/config"
	"github.com/michaelahli/nexd/internal/connector"
	"github.com/michaelahli/nexd/internal/http/handler"
	adminhandler "github.com/michaelahli/nexd/internal/http/handler/admin"
	"github.com/michaelahli/nexd/internal/http/middleware"
	"github.com/michaelahli/nexd/internal/repository"
	"github.com/michaelahli/nexd/internal/service/chat"
	"github.com/michaelahli/nexd/internal/service/search"
)

type searchService interface {
	Search(ctx context.Context, query search.Query) (search.Response, error)
}

type chatService interface {
	Chat(ctx context.Context, req chat.Request) (chat.Response, error)
}

// Options contains optional router dependencies.
type Options struct {
	Users      *auth.UserStore
	Tokens     *auth.TokenManager
	AdminUsers *repository.UsersRepository
	Connectors *repository.ConnectorRepository
	AIConfig   *repository.AIConfigRepository
	SyncJobs   *repository.SyncJobRepository
	ConnMgr    *connector.Manager
	Search     searchService
	Chat       chatService
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

	// Load templates
	templates, err := template.ParseGlob("web/templates/*.html")
	if err != nil {
		// Templates are optional; if missing, admin UI won't work but API still functions
		templates = nil
	}

	if opts.Users != nil && opts.Tokens != nil {
		authHandler := handler.NewAuth(opts.Users, opts.Tokens)
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)
		})

		if opts.Search != nil || opts.Chat != nil {
			r.Group(func(r chi.Router) {
				r.Use(middleware.Auth(opts.Tokens))
				if opts.Search != nil {
					searchHandler := handler.NewSearch(opts.Search)
					r.Post("/search", searchHandler.Query)
				}
				if opts.Chat != nil {
					chatHandler := handler.NewChat(opts.Chat)
					r.Post("/chat", chatHandler.Complete)
				}
			})
		}

		if opts.AdminUsers != nil || opts.Connectors != nil || opts.AIConfig != nil {
			r.Route("/admin", func(r chi.Router) {
				r.Use(middleware.Auth(opts.Tokens))
				if cfg != nil {
					r.Use(middleware.RequireAdmin(cfg.Admin.Emails))
				}
				if templates != nil {
					dashboardHandler := adminhandler.NewDashboard(templates)
					r.Get("/", dashboardHandler.Index)
					usersPageHandler := adminhandler.NewUsersPage(templates)
					r.Get("/users-page", usersPageHandler.Index)
					connectorsPageHandler := adminhandler.NewConnectorsPage(templates)
					r.Get("/connectors-page", connectorsPageHandler.Index)
					aiconfigPageHandler := adminhandler.NewAIConfigPage(templates)
					r.Get("/ai-config-page", aiconfigPageHandler.Index)
				}
				if opts.AdminUsers != nil {
					usersHandler := adminhandler.NewUsers(opts.AdminUsers)
					r.Get("/users", usersHandler.List)
					r.Put("/users", usersHandler.Update)
					r.Delete("/users", usersHandler.Delete)
				}
				if opts.Connectors != nil {
					connectorsHandler := adminhandler.NewConnectors(opts.Connectors, opts.SyncJobs, opts.ConnMgr)
					r.Get("/connectors", connectorsHandler.List)
					r.Post("/connectors", connectorsHandler.Save)
					r.Delete("/connectors", connectorsHandler.Delete)
					r.Post("/connectors/test", connectorsHandler.Test)
					r.Post("/connectors/sync", connectorsHandler.TriggerSync)
				}
				if opts.AIConfig != nil {
					aiConfigHandler := adminhandler.NewAIConfig(opts.AIConfig)
					r.Get("/ai-config", aiConfigHandler.List)
					r.Post("/ai-config", aiConfigHandler.Save)
				}
			})
		}
	}

	if _, err := os.Stat("web/static"); err == nil {
		r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	}

	return r
}
