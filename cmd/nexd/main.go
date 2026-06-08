package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/michaelahli/nexd/internal/auth"
	"github.com/michaelahli/nexd/internal/config"
	"github.com/michaelahli/nexd/internal/connector"
	"github.com/michaelahli/nexd/internal/connector/smb"
	"github.com/michaelahli/nexd/internal/db"
	apphttp "github.com/michaelahli/nexd/internal/http"
	"github.com/michaelahli/nexd/internal/service/chat"
	"github.com/michaelahli/nexd/internal/service/embedding"
	"github.com/michaelahli/nexd/internal/service/indexing"
	"github.com/michaelahli/nexd/internal/service/permission"
	"github.com/michaelahli/nexd/internal/service/search"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		runMigrations(cfg)
		return
	}

	runServer(cfg)
}

func runServer(cfg *config.Config) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	database, err := db.Connect(ctx, cfg.Database)
	cancel()
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer database.Close()

	// Register connector types
	registry := connector.NewRegistry()
	if err := smb.Register(registry); err != nil {
		log.Fatalf("register smb connector: %v", err)
	}

	// Start connector manager
	connectorManager := connector.NewManager(database.Connectors(), registry)
	if err := connectorManager.Start(context.Background()); err != nil {
		log.Printf("Warning: connector manager start: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := connectorManager.Stop(ctx); err != nil {
			log.Printf("Warning: connector manager stop: %v", err)
		}
	}()

	// Build services
	embeddingClient, err := embedding.NewOpenAIClient(embedding.OpenAIConfig{
		Host:   cfg.AI.Host,
		APIKey: cfg.AI.APIKey,
		Model:  cfg.AI.EmbeddingModel,
	})
	if err != nil {
		log.Fatalf("create embedding client: %v", err)
	}
	embeddingService := embedding.NewService(embeddingClient, embedding.Config{
		BatchSize: cfg.Indexing.BatchSize,
		CacheTTL:  time.Hour,
	})

	permissionCache := permission.NewCache(5 * time.Minute)
	permissionService := permission.NewService(database.Permissions(), permissionCache)

	// Start indexing service
	indexingProcessor := indexing.NewProcessor(indexing.ProcessorOptions{
		Documents: database.Documents(),
		Source:    indexing.NewConnectorSource(connectorManager),
		Embedder:  embeddingService,
	})
	indexingService := indexing.NewService(database.SyncJobs(), indexingProcessor, indexing.Config{
		Workers:      cfg.Indexing.Workers,
		PollInterval: time.Second,
	})
	if err := indexingService.Start(context.Background()); err != nil {
		log.Printf("Warning: indexing service start: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := indexingService.Stop(ctx); err != nil {
			log.Printf("Warning: indexing service stop: %v", err)
		}
	}()

	// Build search and chat services
	searchService := search.NewService(database.Search(), embeddingService, permissionService)
	chatClient, err := chat.NewOpenAIClient(chat.OpenAIConfig{
		Host:   cfg.AI.Host,
		APIKey: cfg.AI.APIKey,
		Model:  cfg.AI.ChatModel,
	})
	if err != nil {
		log.Fatalf("create chat client: %v", err)
	}
	chatService := chat.NewService(chatClient, chat.NewSearchRetriever(searchService))

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr: addr,
		Handler: apphttp.NewRouter(cfg, apphttp.Options{
			Users:      auth.NewUserStore(database.Pool),
			Tokens:     auth.NewTokenManager(cfg.JWT.Secret, cfg.JWT.Expiration),
			AdminUsers: database.Users(),
			Connectors: database.Connectors(),
			AIConfig:   database.AIConfig(),
			Search:     searchService,
			Chat:       chatService,
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("Starting NEXD on %s", addr)
		serverErr <- server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("start http server: %v", err)
		}
	case sig := <-shutdown:
		log.Printf("Received %s, shutting down", sig)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown http server: %v", err)
	}

	log.Println("HTTP server stopped")
}

func runMigrations(cfg *config.Config) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := db.Connect(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer database.Close()

	if err := db.RunMigrations(cfg.Database.DSN(), "migrations"); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	log.Println("Database migrations applied")
}
