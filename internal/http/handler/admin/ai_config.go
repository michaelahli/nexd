package admin

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/michaelahli/nexd/internal/repository"
)

type aiConfigRepository interface {
	List(ctx context.Context) ([]repository.AIConfigRecord, error)
	Save(ctx context.Context, cfg repository.AIConfigRecord) error
}

// AIConfig handles admin AI configuration endpoints.
type AIConfig struct {
	repo aiConfigRepository
}

// NewAIConfig creates an admin AI config handler.
func NewAIConfig(repo aiConfigRepository) *AIConfig {
	return &AIConfig{repo: repo}
}

// List returns all AI configs.
func (h *AIConfig) List(w http.ResponseWriter, r *http.Request) {
	configs, err := h.repo.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, configs)
}

// Save creates or updates an AI config.
func (h *AIConfig) Save(w http.ResponseWriter, r *http.Request) {
	var cfg repository.AIConfigRecord
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}
	if cfg.Provider == "" || cfg.Host == "" || cfg.EmbeddingModel == "" || cfg.ChatModel == "" {
		http.Error(w, "provider, host, embedding_model, and chat_model are required", http.StatusBadRequest)
		return
	}
	if cfg.ID == uuid.Nil {
		cfg.ID = uuid.New()
	}
	if err := h.repo.Save(r.Context(), cfg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}
