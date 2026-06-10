package config

import (
	"fmt"
	"time"
)

// Config holds all application configuration
type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	JWT       JWTConfig
	AI        AIConfig
	Admin     AdminConfig
	RateLimit RateLimitConfig
	Indexing  IndexingConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host string
	Port int
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

// DSN returns PostgreSQL connection string
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode)
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret     string
	Expiration time.Duration
}

// AIConfig holds AI provider configuration
type AIConfig struct {
	Provider       string
	Host           string
	APIKey         string
	EmbeddingModel string
	ChatModel      string
}

// AdminConfig holds admin authorization configuration.
type AdminConfig struct {
	Emails         []string
	BootstrapEmail string
	BootstrapName  string
	BootstrapPass  string
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Requests int
	Window   time.Duration
}

// IndexingConfig holds indexing service configuration
type IndexingConfig struct {
	Workers   int
	BatchSize int
}
