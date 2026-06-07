package permission

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCacheAccess(t *testing.T) {
	cache := NewCache(time.Minute)
	userID := uuid.New()
	documentID := uuid.New()

	if _, ok := cache.getAccess(userID, documentID, ReadAction); ok {
		t.Fatal("expected cache miss")
	}

	cache.setAccess(userID, documentID, ReadAction, true)
	allowed, ok := cache.getAccess(userID, documentID, ReadAction)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if !allowed {
		t.Fatal("expected cached allow")
	}

	cache.InvalidateDocument(documentID)
	if _, ok := cache.getAccess(userID, documentID, ReadAction); ok {
		t.Fatal("expected cache miss after document invalidation")
	}
}

func TestCacheUserGroupsCopiesValues(t *testing.T) {
	cache := NewCache(time.Minute)
	userID := uuid.New()
	groupID := uuid.New()
	groups := []uuid.UUID{groupID}

	cache.setUserGroups(userID, groups)
	groups[0] = uuid.New()

	cached, ok := cache.getUserGroups(userID)
	if !ok {
		t.Fatal("expected cached user groups")
	}
	if cached[0] != groupID {
		t.Fatalf("expected copied group ID %s, got %s", groupID, cached[0])
	}

	cached[0] = uuid.New()
	again, ok := cache.getUserGroups(userID)
	if !ok {
		t.Fatal("expected cached user groups")
	}
	if again[0] != groupID {
		t.Fatalf("expected immutable cached group ID %s, got %s", groupID, again[0])
	}
}

func TestCacheExpires(t *testing.T) {
	cache := NewCache(time.Minute)
	now := time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC)
	cache.now = func() time.Time { return now }

	userID := uuid.New()
	documentID := uuid.New()
	cache.setAccess(userID, documentID, ReadAction, true)

	now = now.Add(2 * time.Minute)
	if _, ok := cache.getAccess(userID, documentID, ReadAction); ok {
		t.Fatal("expected expired cache entry")
	}
}
