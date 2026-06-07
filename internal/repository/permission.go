package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PermissionTarget describes a user or group ACL entry for a document.
type PermissionTarget struct {
	UserID         *uuid.UUID
	GroupID        *uuid.UUID
	PermissionType string
}

// PermissionRepository reads and writes document ACLs.
type PermissionRepository struct {
	pool *pgxpool.Pool
}

// NewPermissionRepository creates a PostgreSQL-backed permission repository.
func NewPermissionRepository(pool *pgxpool.Pool) *PermissionRepository {
	return &PermissionRepository{pool: pool}
}

// UserGroupIDs returns all group IDs for a user.
func (r *PermissionRepository) UserGroupIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("permission repository is not configured")
	}

	rows, err := r.pool.Query(ctx, `
		SELECT group_id
		FROM user_groups
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query user groups: %w", err)
	}
	defer rows.Close()

	groupIDs, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return nil, fmt.Errorf("collect user groups: %w", err)
	}

	return groupIDs, nil
}

// UserCanAccessDocument returns true when a user has direct or group document access.
func (r *PermissionRepository) UserCanAccessDocument(ctx context.Context, userID, documentID uuid.UUID, permissionTypes []string) (bool, error) {
	if r == nil || r.pool == nil {
		return false, fmt.Errorf("permission repository is not configured")
	}
	if len(permissionTypes) == 0 {
		permissionTypes = []string{"read"}
	}

	var allowed bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM document_permissions dp
			WHERE dp.document_id = $1
			  AND dp.permission_type = ANY($3)
			  AND (
				dp.user_id = $2
				OR dp.group_id IN (
					SELECT ug.group_id
					FROM user_groups ug
					WHERE ug.user_id = $2
				)
			  )
		)
	`, documentID, userID, permissionTypes).Scan(&allowed)
	if err != nil {
		return false, fmt.Errorf("check document permission: %w", err)
	}

	return allowed, nil
}

// ReplaceDocumentPermissions replaces all ACL entries for a document.
func (r *PermissionRepository) ReplaceDocumentPermissions(ctx context.Context, documentID uuid.UUID, targets []PermissionTarget) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("permission repository is not configured")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin permission sync: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM document_permissions WHERE document_id = $1`, documentID); err != nil {
		return fmt.Errorf("delete document permissions: %w", err)
	}

	for _, target := range targets {
		permissionType := target.PermissionType
		if permissionType == "" {
			permissionType = "read"
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO document_permissions (document_id, user_id, group_id, permission_type)
			VALUES ($1, $2, $3, $4)
		`, documentID, target.UserID, target.GroupID, permissionType); err != nil {
			return fmt.Errorf("insert document permission: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit permission sync: %w", err)
	}

	return nil
}
