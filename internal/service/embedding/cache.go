package embedding

import (
	"sync"
	"time"
)

type cacheEntry struct {
	vector    Vector
	expiresAt time.Time
}

// Cache stores short-lived embeddings keyed by exact input text.
type Cache struct {
	ttl time.Duration
	now func() time.Time
	mu  sync.RWMutex
	m   map[string]cacheEntry
}

// NewCache creates an embedding cache. A non-positive TTL disables caching.
func NewCache(ttl time.Duration) *Cache {
	return &Cache{ttl: ttl, now: time.Now, m: make(map[string]cacheEntry)}
}

// Get returns a copied vector from cache.
func (c *Cache) Get(text string) (Vector, bool) {
	if c == nil || c.ttl <= 0 || text == "" {
		return nil, false
	}

	c.mu.RLock()
	entry, ok := c.m[text]
	c.mu.RUnlock()
	if !ok || !entry.expiresAt.After(c.now()) {
		if ok {
			c.mu.Lock()
			delete(c.m, text)
			c.mu.Unlock()
		}
		return nil, false
	}

	return append(Vector(nil), entry.vector...), true
}

// Set stores a copied vector in cache.
func (c *Cache) Set(text string, vector Vector) {
	if c == nil || c.ttl <= 0 || text == "" || len(vector) == 0 {
		return
	}

	c.mu.Lock()
	c.m[text] = cacheEntry{vector: append(Vector(nil), vector...), expiresAt: c.now().Add(c.ttl)}
	c.mu.Unlock()
}

// Clear removes all cached embeddings.
func (c *Cache) Clear() {
	if c == nil {
		return
	}

	c.mu.Lock()
	c.m = make(map[string]cacheEntry)
	c.mu.Unlock()
}
