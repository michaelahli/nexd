package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailExists        = errors.New("email already exists")
)

// User is the authenticated user record.
type User struct {
	ID           uuid.UUID
	Email        string
	Name         string
	PasswordHash string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// UserStore persists and retrieves users.
type UserStore struct {
	pool *pgxpool.Pool
}

// NewUserStore creates a PostgreSQL-backed user store.
func NewUserStore(pool *pgxpool.Pool) *UserStore {
	return &UserStore{pool: pool}
}

// CreateUser inserts a new local user.
func (s *UserStore) CreateUser(ctx context.Context, email, name, passwordHash string) (User, error) {
	if s == nil || s.pool == nil {
		return User{}, fmt.Errorf("user store is not configured")
	}

	email = normalizeEmail(email)
	row := s.pool.QueryRow(ctx, `
		INSERT INTO users (email, name, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, email, COALESCE(name, ''), COALESCE(password_hash, ''), is_active, created_at, updated_at
	`, email, name, passwordHash)

	user, err := scanUser(row)
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrEmailExists
		}
		return User{}, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

// FindByEmail returns an active user by email.
func (s *UserStore) FindByEmail(ctx context.Context, email string) (User, error) {
	if s == nil || s.pool == nil {
		return User{}, fmt.Errorf("user store is not configured")
	}

	row := s.pool.QueryRow(ctx, `
		SELECT id, email, COALESCE(name, ''), COALESCE(password_hash, ''), is_active, created_at, updated_at
		FROM users
		WHERE email = $1 AND is_active = true
	`, normalizeEmail(email))

	user, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrInvalidCredentials
		}
		return User{}, fmt.Errorf("find user by email: %w", err)
	}

	return user, nil
}

func scanUser(row pgx.Row) (User, error) {
	var user User
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.PasswordHash,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	return user, err
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
