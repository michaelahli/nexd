package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AIConfigRecord is the admin-facing AI configuration model.
type AIConfigRecord struct {
	ID             uuid.UUID `json:"id"`
	Provider       string    `json:"provider"`
	Host           string    `json:"host"`
	APIKeyMasked   string    `json:"api_key_masked"`
	APIKey         string    `json:"-"`
	EmbeddingModel string    `json:"embedding_model"`
	ChatModel      string    `json:"chat_model"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// AIConfigRepository manages admin AI config CRUD operations.
type AIConfigRepository struct {
	pool *pgxpool.Pool
}

// NewAIConfigRepository creates a PostgreSQL-backed AI config repository.
func NewAIConfigRepository(pool *pgxpool.Pool) *AIConfigRepository {
	return &AIConfigRepository{pool: pool}
}

// List returns all AI configs.
func (r *AIConfigRepository) List(ctx context.Context) ([]AIConfigRecord, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("ai config repository is not configured")
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, provider, host, COALESCE(api_key_encrypted, ''), embedding_model, chat_model, is_active, created_at, updated_at
		FROM ai_config
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query ai configs: %w", err)
	}
	defer rows.Close()

	configs, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (AIConfigRecord, error) {
		var record AIConfigRecord
		var rawKey string
		err := row.Scan(&record.ID, &record.Provider, &record.Host, &rawKey, &record.EmbeddingModel, &record.ChatModel, &record.IsActive, &record.CreatedAt, &record.UpdatedAt)
		record.APIKeyMasked = maskKey(rawKey)
		return record, err
	})
	if err != nil {
		return nil, fmt.Errorf("collect ai configs: %w", err)
	}
	return configs, nil
}

// Save creates or updates an AI config.
func (r *AIConfigRepository) Save(ctx context.Context, cfg AIConfigRecord) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("ai config repository is not configured")
	}
	if cfg.ID == uuid.Nil {
		cfg.ID = uuid.New()
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO ai_config (id, provider, host, api_key_encrypted, embedding_model, chat_model, is_active)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			provider = EXCLUDED.provider,
			host = EXCLUDED.host,
			api_key_encrypted = CASE WHEN EXCLUDED.api_key_encrypted IS NULL THEN ai_config.api_key_encrypted ELSE EXCLUDED.api_key_encrypted END,
			embedding_model = EXCLUDED.embedding_model,
			chat_model = EXCLUDED.chat_model,
			is_active = EXCLUDED.is_active,
			updated_at = NOW()
	`, cfg.ID, cfg.Provider, cfg.Host, cfg.APIKey, cfg.EmbeddingModel, cfg.ChatModel, cfg.IsActive)
	if err != nil {
		return fmt.Errorf("save ai config: %w", err)
	}
	return nil
}

func maskKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 4 {
		return "****"
	}
	return key[:2] + "****" + key[len(key)-2:]
}
