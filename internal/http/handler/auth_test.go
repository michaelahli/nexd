package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/michaelahli/nexd/internal/auth"
)

type fakeUsers struct {
	user auth.User
	err  error
}

func (f fakeUsers) CreateUser(ctx context.Context, email, name, passwordHash string) (auth.User, error) {
	if f.err != nil {
		return auth.User{}, f.err
	}
	user := f.user
	user.Email = email
	user.Name = name
	user.PasswordHash = passwordHash
	return user, nil
}

func (f fakeUsers) FindByEmail(ctx context.Context, email string) (auth.User, error) {
	if f.err != nil {
		return auth.User{}, f.err
	}
	return f.user, nil
}

type fakeTokens struct{}

func (fakeTokens) Generate(user auth.User) (string, time.Time, error) {
	return "signed-token", time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC), nil
}

func (fakeTokens) Validate(token string) (*auth.Claims, error) {
	if token != "signed-token" {
		return nil, auth.ErrInvalidToken
	}
	return &auth.Claims{UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Email: "user@example.com"}, nil
}

func TestAuthRegister(t *testing.T) {
	h := NewAuth(fakeUsers{user: auth.User{ID: uuid.New()}}, fakeTokens{})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{"email":"user@example.com","name":"User","password":"password123"}`))

	h.Register(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}
	assertTokenResponse(t, recorder.Body.String())
}

func TestAuthLoginRejectsBadPassword(t *testing.T) {
	hash, err := auth.HashPassword("password123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	h := NewAuth(fakeUsers{user: auth.User{ID: uuid.New(), Email: "user@example.com", PasswordHash: hash}}, fakeTokens{})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"user@example.com","password":"wrong"}`))

	h.Login(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestAuthRegisterDuplicateEmail(t *testing.T) {
	h := NewAuth(fakeUsers{err: auth.ErrEmailExists}, fakeTokens{})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{"email":"user@example.com","password":"password123"}`))

	h.Register(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, recorder.Code)
	}
}

func TestAuthRefresh(t *testing.T) {
	h := NewAuth(fakeUsers{}, fakeTokens{})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	request.Header.Set("Authorization", "Bearer signed-token")

	h.Refresh(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	assertTokenResponse(t, recorder.Body.String())
}

func TestAuthRefreshRejectsMissingToken(t *testing.T) {
	h := NewAuth(fakeUsers{}, fakeTokens{})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)

	h.Refresh(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestAuthLoginRejectsMissingUser(t *testing.T) {
	h := NewAuth(fakeUsers{err: errors.New("missing")}, fakeTokens{})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"user@example.com","password":"password123"}`))

	h.Login(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func assertTokenResponse(t *testing.T, body string) {
	t.Helper()

	var response authResponse
	if err := json.Unmarshal([]byte(body), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Token != "signed-token" {
		t.Fatalf("expected signed-token, got %q", response.Token)
	}
	if response.TokenType != "Bearer" {
		t.Fatalf("expected bearer token type, got %q", response.TokenType)
	}
}
