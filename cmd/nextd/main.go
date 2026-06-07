package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/michael/nextd/internal/config"
	"github.com/michael/nextd/internal/db"
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

	log.Printf("Starting NEXTD on %s:%d", cfg.Server.Host, cfg.Server.Port)
	// TODO: Initialize database and HTTP server.
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
