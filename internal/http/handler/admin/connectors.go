package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/michaelahli/nexd/internal/connector"
)

type connectorsRepository interface {
	List(ctx context.Context) ([]connector.Config, error)
	Save(ctx context.Context, cfg connector.Config) error
	Delete(ctx context.Context, connectorID uuid.UUID) error
}

// Connectors handles admin connector management endpoints.
type Connectors struct {
	repo connectorsRepository
}

// NewConnectors creates an admin connectors handler.
func NewConnectors(repo connectorsRepository) *Connectors {
	return &Connectors{repo: repo}
}

// List returns all connector configs.
func (h *Connectors) List(w http.ResponseWriter, r *http.Request) {
	configs, err := h.repo.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, configs)
}

// Save creates or updates a connector config.
func (h *Connectors) Save(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ID           uuid.UUID      `json:"id"`
		Name         string         `json:"name"`
		Type         string         `json:"type"`
		Settings     map[string]any `json:"settings"`
		SyncInterval int            `json:"sync_interval_seconds"`
		IsActive     bool           `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}
	if payload.Name == "" || payload.Type == "" {
		http.Error(w, "connector name and type are required", http.StatusBadRequest)
		return
	}
	cfg := connector.Config{
		ID:           payload.ID,
		Name:         payload.Name,
		Type:         payload.Type,
		Settings:     payload.Settings,
		SyncInterval: time.Duration(payload.SyncInterval) * time.Second,
		IsActive:     payload.IsActive,
	}
	if err := h.repo.Save(r.Context(), cfg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

// Delete removes a connector config.
func (h *Connectors) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	connectorID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "invalid connector id", http.StatusBadRequest)
		return
	}
	if err := h.repo.Delete(r.Context(), connectorID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
