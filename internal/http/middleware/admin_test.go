package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/michaelahli/nexd/internal/auth"
)

func TestRequireAdminAllowsConfiguredEmail(t *testing.T) {
	handler := RequireAdmin([]string{"admin@example.com"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	ctx := context.WithValue(context.Background(), authUserKey{}, &auth.Claims{UserID: uuid.New(), Email: "admin@example.com"})
	req := httptest.NewRequest(http.MethodGet, "/admin", nil).WithContext(ctx)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.Code)
	}
}

func TestRequireAdminRejectsNonAdmin(t *testing.T) {
	handler := RequireAdmin([]string{"admin@example.com"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	ctx := context.WithValue(context.Background(), authUserKey{}, &auth.Claims{UserID: uuid.New(), Email: "user@example.com"})
	req := httptest.NewRequest(http.MethodGet, "/admin", nil).WithContext(ctx)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.Code)
	}
}
