package config

import (
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Load reads configuration from .env, environment variables, and defaults.
func Load() (*Config, error) {
	_ = godotenv.Load()

	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setDefaults(v)

	cfg := &Config{
		Server: ServerConfig{
			Host: v.GetString("server.host"),
			Port: v.GetInt("server.port"),
		},
		Database: DatabaseConfig{
			Host:     v.GetString("db.host"),
			Port:     v.GetInt("db.port"),
			User:     v.GetString("db.user"),
			Password: v.GetString("db.password"),
			Name:     v.GetString("db.name"),
			SSLMode:  v.GetString("db.sslmode"),
		},
		JWT: JWTConfig{
			Secret:     v.GetString("jwt.secret"),
			Expiration: v.GetDuration("jwt.expiration"),
		},
		AI: AIConfig{
			Provider:       v.GetString("ai.provider"),
			Host:           v.GetString("ai.host"),
			APIKey:         v.GetString("ai.api_key"),
			EmbeddingModel: v.GetString("ai.embedding_model"),
			ChatModel:      v.GetString("ai.chat_model"),
		},
		RateLimit: RateLimitConfig{
			Requests: v.GetInt("rate_limit.requests"),
			Window:   v.GetDuration("rate_limit.window"),
		},
		Indexing: IndexingConfig{
			Workers:   v.GetInt("indexing.workers"),
			BatchSize: v.GetInt("indexing.batch_size"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("db.host", "localhost")
	v.SetDefault("db.port", 5432)
	v.SetDefault("db.user", "nexd")
	v.SetDefault("db.password", "nexd")
	v.SetDefault("db.name", "nexd")
	v.SetDefault("db.sslmode", "disable")
	v.SetDefault("jwt.secret", "change-me-in-production")
	v.SetDefault("jwt.expiration", 24*time.Hour)
	v.SetDefault("ai.provider", "openai")
	v.SetDefault("ai.host", "https://api.openai.com/v1")
	v.SetDefault("ai.embedding_model", "text-embedding-ada-002")
	v.SetDefault("ai.chat_model", "gpt-4")
	v.SetDefault("rate_limit.requests", 100)
	v.SetDefault("rate_limit.window", time.Minute)
	v.SetDefault("indexing.workers", 5)
	v.SetDefault("indexing.batch_size", 10)
}
