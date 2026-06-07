package permission

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type accessCacheKey struct {
	UserID     uuid.UUID
	DocumentID uuid.UUID
	Action     string
}

type cacheEntry[T any] struct {
	value     T
	expiresAt time.Time
}

// Cache stores short-lived permission lookups.
type Cache struct {
	ttl       time.Duration
	now       func() time.Time
	mu        sync.RWMutex
	access    map[accessCacheKey]cacheEntry[bool]
	userGroup map[uuid.UUID]cacheEntry[[]uuid.UUID]
}

// NewCache creates a permission cache. A non-positive TTL disables caching.
func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		ttl:       ttl,
		now:       time.Now,
		access:    make(map[accessCacheKey]cacheEntry[bool]),
		userGroup: make(map[uuid.UUID]cacheEntry[[]uuid.UUID]),
	}
}

func (c *Cache) getAccess(userID, documentID uuid.UUID, action string) (bool, bool) {
	if c == nil || c.ttl <= 0 {
		return false, false
	}

	key := accessCacheKey{UserID: userID, DocumentID: documentID, Action: action}
	c.mu.RLock()
	entry, ok := c.access[key]
	c.mu.RUnlock()
	if !ok || !entry.expiresAt.After(c.now()) {
		if ok {
			c.mu.Lock()
			delete(c.access, key)
			c.mu.Unlock()
		}
		return false, false
	}

	return entry.value, true
}

func (c *Cache) setAccess(userID, documentID uuid.UUID, action string, allowed bool) {
	if c == nil || c.ttl <= 0 {
		return
	}

	key := accessCacheKey{UserID: userID, DocumentID: documentID, Action: action}
	c.mu.Lock()
	c.access[key] = cacheEntry[bool]{value: allowed, expiresAt: c.now().Add(c.ttl)}
	c.mu.Unlock()
}

func (c *Cache) getUserGroups(userID uuid.UUID) ([]uuid.UUID, bool) {
	if c == nil || c.ttl <= 0 {
		return nil, false
	}

	c.mu.RLock()
	entry, ok := c.userGroup[userID]
	c.mu.RUnlock()
	if !ok || !entry.expiresAt.After(c.now()) {
		if ok {
			c.mu.Lock()
			delete(c.userGroup, userID)
			c.mu.Unlock()
		}
		return nil, false
	}

	return append([]uuid.UUID(nil), entry.value...), true
}

func (c *Cache) setUserGroups(userID uuid.UUID, groupIDs []uuid.UUID) {
	if c == nil || c.ttl <= 0 {
		return
	}

	c.mu.Lock()
	c.userGroup[userID] = cacheEntry[[]uuid.UUID]{value: append([]uuid.UUID(nil), groupIDs...), expiresAt: c.now().Add(c.ttl)}
	c.mu.Unlock()
}

// InvalidateDocument removes cached access decisions for a document.
func (c *Cache) InvalidateDocument(documentID uuid.UUID) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.access {
		if key.DocumentID == documentID {
			delete(c.access, key)
		}
	}
}

// InvalidateUser removes cached access decisions and group memberships for a user.
func (c *Cache) InvalidateUser(userID uuid.UUID) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.userGroup, userID)
	for key := range c.access {
		if key.UserID == userID {
			delete(c.access, key)
		}
	}
}

// Clear removes all cached permission data.
func (c *Cache) Clear() {
	if c == nil {
		return
	}

	c.mu.Lock()
	c.access = make(map[accessCacheKey]cacheEntry[bool])
	c.userGroup = make(map[uuid.UUID]cacheEntry[[]uuid.UUID])
	c.mu.Unlock()
}
