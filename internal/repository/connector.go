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

// List returns all connector configs.
func (r *ConnectorRepository) List(ctx context.Context) ([]connector.Config, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("connector repository is not configured")
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, name, type, config, sync_interval_seconds, last_sync_at, is_active
		FROM connector_configs
		ORDER BY created_at DESC
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

// Save creates or updates a connector config.
func (r *ConnectorRepository) Save(ctx context.Context, cfg connector.Config) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("connector repository is not configured")
	}
	if cfg.ID == uuid.Nil {
		cfg.ID = uuid.New()
	}
	payload, err := json.Marshal(cfg.Settings)
	if err != nil {
		return fmt.Errorf("marshal connector config: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO connector_configs (id, name, type, config, sync_interval_seconds, last_sync_at, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			type = EXCLUDED.type,
			config = EXCLUDED.config,
			sync_interval_seconds = EXCLUDED.sync_interval_seconds,
			last_sync_at = EXCLUDED.last_sync_at,
			is_active = EXCLUDED.is_active,
			updated_at = NOW()
	`, cfg.ID, cfg.Name, cfg.Type, payload, int(cfg.SyncInterval.Seconds()), cfg.LastSyncAt, cfg.IsActive)
	if err != nil {
		return fmt.Errorf("save connector config: %w", err)
	}
	return nil
}

// Delete removes a connector config.
func (r *ConnectorRepository) Delete(ctx context.Context, connectorID uuid.UUID) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("connector repository is not configured")
	}
	commandTag, err := r.pool.Exec(ctx, `DELETE FROM connector_configs WHERE id = $1`, connectorID)
	if err != nil {
		return fmt.Errorf("delete connector config: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
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
