package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type visitor struct {
	windowStart time.Time
	requests    int
}

// RateLimiter limits requests per client IP within a fixed time window.
type RateLimiter struct {
	limit    int
	window   time.Duration
	now      func() time.Time
	mu       sync.Mutex
	visitors map[string]visitor
}

// NewRateLimiter creates a fixed-window in-memory rate limiter.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		limit:    limit,
		window:   window,
		now:      time.Now,
		visitors: make(map[string]visitor),
	}
}

// Middleware applies rate limiting to an HTTP handler.
func (l *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.allow(clientIP(r)) {
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (l *RateLimiter) allow(key string) bool {
	if l == nil || l.limit <= 0 || l.window <= 0 {
		return true
	}

	now := l.now()

	l.mu.Lock()
	defer l.mu.Unlock()

	entry := l.visitors[key]
	if entry.windowStart.IsZero() || now.Sub(entry.windowStart) >= l.window {
		l.visitors[key] = visitor{windowStart: now, requests: 1}
		l.cleanup(now)
		return true
	}

	if entry.requests >= l.limit {
		return false
	}

	entry.requests++
	l.visitors[key] = entry
	return true
}

func (l *RateLimiter) cleanup(now time.Time) {
	for key, entry := range l.visitors {
		if now.Sub(entry.windowStart) >= l.window {
			delete(l.visitors, key)
		}
	}
}

func clientIP(r *http.Request) string {
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		return forwardedFor
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}

	return r.RemoteAddr
}
