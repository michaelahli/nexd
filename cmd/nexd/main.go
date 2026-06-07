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
	"github.com/michaelahli/nexd/internal/db"
	apphttp "github.com/michaelahli/nexd/internal/http"
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

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr: addr,
		Handler: apphttp.NewRouter(cfg, apphttp.Options{
			Users:  auth.NewUserStore(database.Pool),
			Tokens: auth.NewTokenManager(cfg.JWT.Secret, cfg.JWT.Expiration),
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
