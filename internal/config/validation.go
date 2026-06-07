package config

import (
	"errors"
	"fmt"
)

// Validate ensures required configuration values are present and usable.
func (c Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("server port must be between 1 and 65535")
	}

	if c.Database.Host == "" {
		return errors.New("database host is required")
	}
	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		return fmt.Errorf("database port must be between 1 and 65535")
	}
	if c.Database.User == "" {
		return errors.New("database user is required")
	}
	if c.Database.Name == "" {
		return errors.New("database name is required")
	}
	if c.Database.SSLMode == "" {
		return errors.New("database sslmode is required")
	}

	if c.JWT.Secret == "" {
		return errors.New("jwt secret is required")
	}
	if c.JWT.Expiration <= 0 {
		return errors.New("jwt expiration must be positive")
	}

	if c.AI.Provider == "" {
		return errors.New("ai provider is required")
	}
	if c.AI.Host == "" {
		return errors.New("ai host is required")
	}
	if c.AI.EmbeddingModel == "" {
		return errors.New("ai embedding model is required")
	}
	if c.AI.ChatModel == "" {
		return errors.New("ai chat model is required")
	}

	if c.RateLimit.Requests <= 0 {
		return errors.New("rate limit requests must be positive")
	}
	if c.RateLimit.Window <= 0 {
		return errors.New("rate limit window must be positive")
	}

	if c.Indexing.Workers <= 0 {
		return errors.New("indexing workers must be positive")
	}
	if c.Indexing.BatchSize <= 0 {
		return errors.New("indexing batch size must be positive")
	}
	if c.Indexing.BatchSize > 100 {
		return errors.New("indexing batch size must not exceed 100")
	}

	return nil
}
