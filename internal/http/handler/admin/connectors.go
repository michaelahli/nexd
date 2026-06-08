package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/michaelahli/nexd/internal/connector"
	"github.com/michaelahli/nexd/internal/service/indexing"
)

type connectorsRepository interface {
	List(ctx context.Context) ([]connector.Config, error)
	Save(ctx context.Context, cfg connector.Config) error
	Delete(ctx context.Context, connectorID uuid.UUID) error
}

type syncJobEnqueuer interface {
	Enqueue(ctx context.Context, connectorID uuid.UUID, jobType string, scheduledAt time.Time) (uuid.UUID, error)
}

type connectorOperator interface {
	StartConnector(ctx context.Context, cfg connector.Config) error
	StopConnector(ctx context.Context, connectorID uuid.UUID) error
	Health(ctx context.Context, connectorID uuid.UUID) error
}

// Connectors handles admin connector management endpoints.
type Connectors struct {
	repo connectorsRepository
	jobs syncJobEnqueuer
	ops  connectorOperator
}

// NewConnectors creates an admin connectors handler.
func NewConnectors(repo connectorsRepository, jobs syncJobEnqueuer, ops connectorOperator) *Connectors {
	return &Connectors{repo: repo, jobs: jobs, ops: ops}
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

// Test validates and health-checks a connector from stored config.
func (h *Connectors) Test(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	connectorID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, "invalid connector id", http.StatusBadRequest)
		return
	}
	configs, err := h.repo.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var cfg connector.Config
	found := false
	for _, item := range configs {
		if item.ID == connectorID {
			cfg = item
			found = true
			break
		}
	}
	if !found {
		http.Error(w, "connector not found", http.StatusNotFound)
		return
	}
	if h.ops == nil {
		http.Error(w, "connector operations unavailable", http.StatusServiceUnavailable)
		return
	}

	started := false
	if err := h.ops.StartConnector(r.Context(), cfg); err == nil {
		started = true
	}
	if err := h.ops.Health(r.Context(), connectorID); err != nil {
		if started {
			_ = h.ops.StopConnector(r.Context(), connectorID)
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if started {
		_ = h.ops.StopConnector(r.Context(), connectorID)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// TriggerSync enqueues a connector sync job.
func (h *Connectors) TriggerSync(w http.ResponseWriter, r *http.Request) {
	if h.jobs == nil {
		http.Error(w, "sync queue unavailable", http.StatusServiceUnavailable)
		return
	}
	var payload struct {
		ConnectorID uuid.UUID `json:"connector_id"`
		Full        bool      `json:"full"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}
	if payload.ConnectorID == uuid.Nil {
		http.Error(w, "connector_id is required", http.StatusBadRequest)
		return
	}
	jobType := indexing.JobTypeIncrementalSync
	if payload.Full {
		jobType = indexing.JobTypeFullSync
	}
	jobID, err := h.jobs.Enqueue(r.Context(), payload.ConnectorID, jobType, time.Now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "queued", "job_id": jobID.String()})
}
