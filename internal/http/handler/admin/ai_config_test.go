package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/michaelahli/nexd/internal/repository"
)

type fakeAIConfigRepo struct{}

func (f *fakeAIConfigRepo) List(ctx context.Context) ([]repository.AIConfigRecord, error) {
	return []repository.AIConfigRecord{{ID: uuid.New(), Provider: "openai", Host: "https://api.openai.com/v1", EmbeddingModel: "text-embedding-3-small", ChatModel: "gpt-4o", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}}, nil
}
func (f *fakeAIConfigRepo) Save(ctx context.Context, cfg repository.AIConfigRecord) error { return nil }

func TestAIConfigList(t *testing.T) {
	h := NewAIConfig(&fakeAIConfigRepo{})
	req := httptest.NewRequest(http.MethodGet, "/admin/ai-config", nil)
	w := httptest.NewRecorder()

	h.List(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAIConfigSave(t *testing.T) {
	h := NewAIConfig(&fakeAIConfigRepo{})
	body := `{"provider":"openai","host":"https://api.openai.com/v1","embedding_model":"text-embedding-3-small","chat_model":"gpt-4o","is_active":true}`
	req := httptest.NewRequest(http.MethodPost, "/admin/ai-config", strings.NewReader(body))
	w := httptest.NewRecorder()

	h.Save(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
