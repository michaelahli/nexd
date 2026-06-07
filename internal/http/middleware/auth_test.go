package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/michael/nextd/internal/auth"
)

type fakeClaimsValidator struct {
	claims *auth.Claims
	err    error
}

func (v fakeClaimsValidator) Validate(token string) (*auth.Claims, error) {
	if v.err != nil {
		return nil, v.err
	}
	return v.claims, nil
}

func TestAuthMiddlewareStoresClaims(t *testing.T) {
	userID := uuid.New()
	validator := fakeClaimsValidator{claims: &auth.Claims{UserID: userID, Email: "user@example.com"}}

	handler := Auth(validator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			t.Fatal("expected claims in context")
		}
		if claims.UserID != userID {
			t.Fatalf("expected user id %s, got %s", userID, claims.UserID)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer token")

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
}

func TestAuthMiddlewareRejectsMissingToken(t *testing.T) {
	validator := fakeClaimsValidator{claims: &auth.Claims{}}
	handler := Auth(validator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestAuthMiddlewareRejectsInvalidToken(t *testing.T) {
	validator := fakeClaimsValidator{err: auth.ErrInvalidToken}
	handler := Auth(validator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer token")

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}
