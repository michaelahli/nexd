package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/michaelahli/nexd/internal/config"
	"github.com/michaelahli/nexd/internal/repository"
)

// DB wraps the PostgreSQL connection pool.
type DB struct {
	Pool *pgxpool.Pool
}

// Connect opens a PostgreSQL connection pool and verifies connectivity.
func Connect(ctx context.Context, cfg config.DatabaseConfig) (*DB, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}

	poolCfg.MaxConns = 20
	poolCfg.MinConns = 2
	poolCfg.MaxConnLifetime = time.Hour
	poolCfg.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}

	database := &DB{Pool: pool}
	if err := database.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return database, nil
}

// Ping verifies that PostgreSQL accepts queries.
func (d *DB) Ping(ctx context.Context) error {
	if d == nil || d.Pool == nil {
		return fmt.Errorf("database pool is nil")
	}
	if err := d.Pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	return nil
}

// Permissions returns a repository for document permission checks.
func (d *DB) Permissions() *repository.PermissionRepository {
	if d == nil {
		return repository.NewPermissionRepository(nil)
	}
	return repository.NewPermissionRepository(d.Pool)
}

// Connectors returns a repository for connector configuration.
func (d *DB) Connectors() *repository.ConnectorRepository {
	if d == nil {
		return repository.NewConnectorRepository(nil)
	}
	return repository.NewConnectorRepository(d.Pool)
}

// Close releases the connection pool.
func (d *DB) Close() {
	if d != nil && d.Pool != nil {
		d.Pool.Close()
	}
}
