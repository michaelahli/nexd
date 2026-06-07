package permission

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/michaelahli/nexd/internal/repository"
)

type fakePermissionRepository struct {
	allowed       bool
	accessCalls   int
	groups        []uuid.UUID
	groupCalls    int
	syncedDocID   uuid.UUID
	syncedTargets []repository.PermissionTarget
}

func (r *fakePermissionRepository) UserCanAccessDocument(ctx context.Context, userID, documentID uuid.UUID, permissionTypes []string) (bool, error) {
	r.accessCalls++
	return r.allowed, nil
}

func (r *fakePermissionRepository) UserGroupIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	r.groupCalls++
	return append([]uuid.UUID(nil), r.groups...), nil
}

func (r *fakePermissionRepository) ReplaceDocumentPermissions(ctx context.Context, documentID uuid.UUID, targets []repository.PermissionTarget) error {
	r.syncedDocID = documentID
	r.syncedTargets = append([]repository.PermissionTarget(nil), targets...)
	return nil
}

func TestServiceCanAccessDocumentUsesCache(t *testing.T) {
	repo := &fakePermissionRepository{allowed: true}
	service := NewService(repo, NewCache(time.Minute))
	userID := uuid.New()
	documentID := uuid.New()

	allowed, err := service.CanAccessDocument(context.Background(), userID, documentID, ReadAction)
	if err != nil {
		t.Fatalf("check access: %v", err)
	}
	if !allowed {
		t.Fatal("expected access allowed")
	}

	allowed, err = service.CanAccessDocument(context.Background(), userID, documentID, ReadAction)
	if err != nil {
		t.Fatalf("check cached access: %v", err)
	}
	if !allowed {
		t.Fatal("expected cached access allowed")
	}
	if repo.accessCalls != 1 {
		t.Fatalf("expected one repository call, got %d", repo.accessCalls)
	}
}

func TestServiceUserGroupIDsUsesCache(t *testing.T) {
	groupID := uuid.New()
	repo := &fakePermissionRepository{groups: []uuid.UUID{groupID}}
	service := NewService(repo, NewCache(time.Minute))
	userID := uuid.New()

	groups, err := service.UserGroupIDs(context.Background(), userID)
	if err != nil {
		t.Fatalf("resolve groups: %v", err)
	}
	if len(groups) != 1 || groups[0] != groupID {
		t.Fatalf("expected group %s, got %v", groupID, groups)
	}

	groups, err = service.UserGroupIDs(context.Background(), userID)
	if err != nil {
		t.Fatalf("resolve cached groups: %v", err)
	}
	if repo.groupCalls != 1 {
		t.Fatalf("expected one group repository call, got %d", repo.groupCalls)
	}
}

func TestServiceSyncDocumentPermissionsInvalidatesCache(t *testing.T) {
	repo := &fakePermissionRepository{allowed: true}
	cache := NewCache(time.Minute)
	service := NewService(repo, cache)
	userID := uuid.New()
	documentID := uuid.New()
	groupID := uuid.New()

	if _, err := service.CanAccessDocument(context.Background(), userID, documentID, ReadAction); err != nil {
		t.Fatalf("check access: %v", err)
	}
	if repo.accessCalls != 1 {
		t.Fatalf("expected one repository call, got %d", repo.accessCalls)
	}

	if err := service.SyncDocumentPermissions(context.Background(), documentID, []repository.PermissionTarget{{GroupID: &groupID}}); err != nil {
		t.Fatalf("sync permissions: %v", err)
	}
	if repo.syncedDocID != documentID {
		t.Fatalf("expected synced document %s, got %s", documentID, repo.syncedDocID)
	}
	if len(repo.syncedTargets) != 1 || repo.syncedTargets[0].GroupID == nil || *repo.syncedTargets[0].GroupID != groupID {
		t.Fatalf("expected synced group target")
	}

	if _, err := service.CanAccessDocument(context.Background(), userID, documentID, ReadAction); err != nil {
		t.Fatalf("check access after sync: %v", err)
	}
	if repo.accessCalls != 2 {
		t.Fatalf("expected cache invalidation to force second repository call, got %d calls", repo.accessCalls)
	}
}

func TestServiceRejectsInvalidSyncTarget(t *testing.T) {
	repo := &fakePermissionRepository{}
	service := NewService(repo, NewCache(time.Minute))

	err := service.SyncDocumentPermissions(context.Background(), uuid.New(), []repository.PermissionTarget{{}})
	if err == nil {
		t.Fatal("expected invalid sync target error")
	}
}
