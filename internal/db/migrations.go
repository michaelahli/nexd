package db

import (
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations applies all pending SQL migrations from migrationPath.
func RunMigrations(databaseURL, migrationPath string) error {
	if databaseURL == "" {
		return fmt.Errorf("database URL is required")
	}
	if migrationPath == "" {
		return fmt.Errorf("migration path is required")
	}

	m, err := migrate.New("file://"+migrationPath, migrationURL(databaseURL))
	if err != nil {
		return fmt.Errorf("create migration instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}

func migrationURL(databaseURL string) string {
	return strings.Replace(databaseURL, "postgres://", "pgx5://", 1)
}
