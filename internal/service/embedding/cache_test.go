package embedding

import (
	"testing"
	"time"
)

func TestCacheCopiesVectors(t *testing.T) {
	cache := NewCache(time.Minute)
	vector := Vector{1, 2, 3}

	cache.Set("hello", vector)
	vector[0] = 9

	cached, ok := cache.Get("hello")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if cached[0] != 1 {
		t.Fatalf("expected stored copy, got %v", cached)
	}

	cached[0] = 8
	again, ok := cache.Get("hello")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if again[0] != 1 {
		t.Fatalf("expected returned copy, got %v", again)
	}
}

func TestCacheExpires(t *testing.T) {
	cache := NewCache(time.Minute)
	now := time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC)
	cache.now = func() time.Time { return now }

	cache.Set("hello", Vector{1})
	now = now.Add(2 * time.Minute)

	if _, ok := cache.Get("hello"); ok {
		t.Fatal("expected expired cache miss")
	}
}
