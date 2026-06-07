package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/michael/nextd/internal/config"
	apphttp "github.com/michael/nextd/internal/http"
)

func TestHealthEndpoint(t *testing.T) {
	router := apphttp.NewRouter(testConfig())
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if recorder.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected json content type, got %q", recorder.Header().Get("Content-Type"))
	}
	if recorder.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected request ID header")
	}
}

func TestCORSPreflight(t *testing.T) {
	router := apphttp.NewRouter(testConfig())
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/health", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
	if recorder.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("expected wildcard CORS origin")
	}
}

func TestRateLimiter(t *testing.T) {
	cfg := testConfig()
	cfg.RateLimit.Requests = 1
	router := apphttp.NewRouter(cfg)

	first := httptest.NewRecorder()
	firstRequest := httptest.NewRequest(http.MethodGet, "/health", nil)
	firstRequest.RemoteAddr = "192.0.2.10:1234"
	router.ServeHTTP(first, firstRequest)
	if first.Code != http.StatusOK {
		t.Fatalf("expected first request status %d, got %d", http.StatusOK, first.Code)
	}

	second := httptest.NewRecorder()
	secondRequest := httptest.NewRequest(http.MethodGet, "/health", nil)
	secondRequest.RemoteAddr = "192.0.2.10:5678"
	router.ServeHTTP(second, secondRequest)
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request status %d, got %d", http.StatusTooManyRequests, second.Code)
	}
}

func testConfig() *config.Config {
	return &config.Config{
		RateLimit: config.RateLimitConfig{
			Requests: 100,
			Window:   time.Minute,
		},
	}
}
