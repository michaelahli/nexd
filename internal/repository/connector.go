package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/michaelahli/nexd/internal/connector"
)

// ConnectorRepository reads connector configuration records.
type ConnectorRepository struct {
	pool *pgxpool.Pool
}

// NewConnectorRepository creates a PostgreSQL-backed connector repository.
func NewConnectorRepository(pool *pgxpool.Pool) *ConnectorRepository {
	return &ConnectorRepository{pool: pool}
}

// ActiveConnectorConfigs returns all active connector configs.
func (r *ConnectorRepository) ActiveConnectorConfigs(ctx context.Context) ([]connector.Config, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("connector repository is not configured")
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, name, type, config, sync_interval_seconds, last_sync_at, is_active
		FROM connector_configs
		WHERE is_active = true
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query connector configs: %w", err)
	}
	defer rows.Close()

	configs, err := pgx.CollectRows(rows, scanConnectorConfig)
	if err != nil {
		return nil, fmt.Errorf("collect connector configs: %w", err)
	}
	return configs, nil
}

// UpdateLastSyncAt records the most recent successful sync time.
func (r *ConnectorRepository) UpdateLastSyncAt(ctx context.Context, connectorID uuid.UUID, syncedAt time.Time) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("connector repository is not configured")
	}

	result, err := r.pool.Exec(ctx, `
		UPDATE connector_configs
		SET last_sync_at = $2, updated_at = NOW()
		WHERE id = $1
	`, connectorID, syncedAt)
	if err != nil {
		return fmt.Errorf("update connector last sync: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("connector %s was not found", connectorID)
	}
	return nil
}

func scanConnectorConfig(row pgx.CollectableRow) (connector.Config, error) {
	var cfg connector.Config
	var rawConfig []byte
	var syncIntervalSeconds int
	if err := row.Scan(&cfg.ID, &cfg.Name, &cfg.Type, &rawConfig, &syncIntervalSeconds, &cfg.LastSyncAt, &cfg.IsActive); err != nil {
		return connector.Config{}, err
	}

	if len(rawConfig) > 0 {
		if err := json.Unmarshal(rawConfig, &cfg.Settings); err != nil {
			return connector.Config{}, fmt.Errorf("decode connector config: %w", err)
		}
	}
	if cfg.Settings == nil {
		cfg.Settings = make(map[string]any)
	}
	cfg.SyncInterval = time.Duration(syncIntervalSeconds) * time.Second
	return cfg, nil
}
