package permission

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/michaelahli/nexd/internal/repository"
)

const ReadAction = "read"

// Repository is the persistence contract required by Service.
type Repository interface {
	UserCanAccessDocument(ctx context.Context, userID, documentID uuid.UUID, permissionTypes []string) (bool, error)
	UserGroupIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	ReplaceDocumentPermissions(ctx context.Context, documentID uuid.UUID, targets []repository.PermissionTarget) error
}

// Service checks and synchronizes document permissions.
type Service struct {
	repo  Repository
	cache *Cache
}

// NewService creates a permission service.
func NewService(repo Repository, cache *Cache) *Service {
	return &Service{repo: repo, cache: cache}
}

// CanAccessDocument returns whether a user can perform action on a document.
func (s *Service) CanAccessDocument(ctx context.Context, userID, documentID uuid.UUID, action string) (bool, error) {
	if s == nil || s.repo == nil {
		return false, fmt.Errorf("permission service is not configured")
	}
	if userID == uuid.Nil {
		return false, fmt.Errorf("user ID is required")
	}
	if documentID == uuid.Nil {
		return false, fmt.Errorf("document ID is required")
	}

	action = normalizeAction(action)
	if allowed, ok := s.cache.getAccess(userID, documentID, action); ok {
		return allowed, nil
	}

	allowed, err := s.repo.UserCanAccessDocument(ctx, userID, documentID, permissionTypesForAction(action))
	if err != nil {
		return false, err
	}

	s.cache.setAccess(userID, documentID, action, allowed)
	return allowed, nil
}

// UserGroupIDs returns the groups a user belongs to.
func (s *Service) UserGroupIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("permission service is not configured")
	}
	if userID == uuid.Nil {
		return nil, fmt.Errorf("user ID is required")
	}

	if groupIDs, ok := s.cache.getUserGroups(userID); ok {
		return groupIDs, nil
	}

	groupIDs, err := s.repo.UserGroupIDs(ctx, userID)
	if err != nil {
		return nil, err
	}

	s.cache.setUserGroups(userID, groupIDs)
	return groupIDs, nil
}

// SyncDocumentPermissions replaces the ACL entries for a document.
func (s *Service) SyncDocumentPermissions(ctx context.Context, documentID uuid.UUID, targets []repository.PermissionTarget) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("permission service is not configured")
	}
	if documentID == uuid.Nil {
		return fmt.Errorf("document ID is required")
	}

	for _, target := range targets {
		if target.UserID == nil && target.GroupID == nil {
			return fmt.Errorf("permission target requires user or group")
		}
		if target.UserID != nil && target.GroupID != nil {
			return fmt.Errorf("permission target cannot contain both user and group")
		}
	}

	if err := s.repo.ReplaceDocumentPermissions(ctx, documentID, targets); err != nil {
		return err
	}

	s.cache.InvalidateDocument(documentID)
	return nil
}

func normalizeAction(action string) string {
	action = strings.ToLower(strings.TrimSpace(action))
	if action == "" {
		return ReadAction
	}
	return action
}

func permissionTypesForAction(action string) []string {
	switch action {
	case ReadAction:
		return []string{"read"}
	default:
		return []string{action}
	}
}
