package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRecord is the admin-facing user model.
type UserRecord struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UsersRepository manages admin user CRUD operations.
type UsersRepository struct {
	pool *pgxpool.Pool
}

// NewUsersRepository creates a PostgreSQL-backed users repository.
func NewUsersRepository(pool *pgxpool.Pool) *UsersRepository {
	return &UsersRepository{pool: pool}
}

// List returns all users ordered by creation date.
func (r *UsersRepository) List(ctx context.Context) ([]UserRecord, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("users repository is not configured")
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, email, COALESCE(name, ''), is_active, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	users, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (UserRecord, error) {
		var user UserRecord
		err := row.Scan(&user.ID, &user.Email, &user.Name, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
		return user, err
	})
	if err != nil {
		return nil, fmt.Errorf("collect users: %w", err)
	}
	return users, nil
}

// Update modifies admin-editable user fields.
func (r *UsersRepository) Update(ctx context.Context, user UserRecord) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("users repository is not configured")
	}
	commandTag, err := r.pool.Exec(ctx, `
		UPDATE users
		SET email = $2, name = NULLIF($3, ''), is_active = $4, updated_at = NOW()
		WHERE id = $1
	`, user.ID, user.Email, user.Name, user.IsActive)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("user %s was not found", user.ID)
	}
	return nil
}

// Delete removes a user.
func (r *UsersRepository) Delete(ctx context.Context, userID uuid.UUID) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("users repository is not configured")
	}
	commandTag, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("user %s was not found", userID)
	}
	return nil
}
