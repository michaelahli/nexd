package admin

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/michaelahli/nexd/internal/repository"
)

type usersRepository interface {
	List(ctx context.Context) ([]repository.UserRecord, error)
	Update(ctx context.Context, user repository.UserRecord) error
	Delete(ctx context.Context, userID uuid.UUID) error
}

// Users handles admin user management endpoints.
type Users struct {
	repo usersRepository
}

// NewUsers creates an admin users handler.
func NewUsers(repo usersRepository) *Users {
	return &Users{repo: repo}
}

// List returns all users.
func (h *Users) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.repo.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, users)
}

// Update modifies an existing user.
func (h *Users) Update(w http.ResponseWriter, r *http.Request) {
	var user repository.UserRecord
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}
	if user.ID == uuid.Nil || user.Email == "" {
		http.Error(w, "user id and email are required", http.StatusBadRequest)
		return
	}
	if err := h.repo.Update(r.Context(), user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// Delete removes a user.
func (h *Users) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	userID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}
	if err := h.repo.Delete(r.Context(), userID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
