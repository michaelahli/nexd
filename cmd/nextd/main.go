package main

import (
	"log"

	"github.com/michael/nextd/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	log.Printf("Starting NEXTD on %s:%d", cfg.Server.Host, cfg.Server.Port)
	// TODO: Initialize database and HTTP server.
}
