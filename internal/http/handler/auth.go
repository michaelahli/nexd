package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/michaelahli/nexd/internal/auth"
)

type userCreator interface {
	CreateUser(ctx context.Context, email, name, passwordHash string) (auth.User, error)
}

type userFinder interface {
	FindByEmail(ctx context.Context, email string) (auth.User, error)
}

type tokenGenerator interface {
	Generate(user auth.User) (string, time.Time, error)
}

type tokenValidator interface {
	Validate(token string) (*auth.Claims, error)
}

// Auth handles authentication endpoints.
type Auth struct {
	users interface {
		userCreator
		userFinder
	}
	tokens interface {
		tokenGenerator
		tokenValidator
	}
}

// NewAuth creates an auth handler.
func NewAuth(users interface {
	userCreator
	userFinder
}, tokens interface {
	tokenGenerator
	tokenValidator
}) *Auth {
	return &Auth{users: users, tokens: tokens}
}

type authRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type authResponse struct {
	Token     string       `json:"token"`
	TokenType string       `json:"token_type"`
	ExpiresAt string       `json:"expires_at"`
	User      userResponse `json:"user"`
}

type userResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// Register creates a new local user and returns a bearer token.
func (h *Auth) Register(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	user, err := h.users.CreateUser(r.Context(), req.Email, strings.TrimSpace(req.Name), hash)
	if err != nil {
		if errors.Is(err, auth.ErrEmailExists) {
			writeError(w, http.StatusConflict, "email already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	h.writeToken(w, http.StatusCreated, user)
}

// Login verifies credentials and returns a bearer token.
func (h *Auth) Login(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	user, err := h.users.FindByEmail(r.Context(), req.Email)
	if err != nil || !auth.CheckPassword(user.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	h.writeToken(w, http.StatusOK, user)
}

// Refresh validates the current bearer token and returns a new token.
func (h *Auth) Refresh(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "bearer token is required")
		return
	}

	claims, err := h.tokens.Validate(token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	h.writeToken(w, http.StatusOK, auth.User{ID: claims.UserID, Email: claims.Email})
}

func (h *Auth) writeToken(w http.ResponseWriter, status int, user auth.User) {
	token, expiresAt, err := h.tokens.Generate(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	writeJSON(w, status, authResponse{
		Token:     token,
		TokenType: "Bearer",
		ExpiresAt: expiresAt.Format(time.RFC3339),
		User: userResponse{
			ID:    user.ID.String(),
			Email: user.Email,
			Name:  user.Name,
		},
	})
}

func bearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if header == "" {
		return ""
	}

	prefix := "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
