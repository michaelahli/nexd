package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/michaelahli/nexd/internal/connector"
)

type fakeConnectorsRepo struct{}

func (f *fakeConnectorsRepo) List(ctx context.Context) ([]connector.Config, error) {
	return []connector.Config{{ID: uuid.New(), Name: "SMB", Type: "smb", SyncInterval: time.Minute, IsActive: true}}, nil
}
func (f *fakeConnectorsRepo) Save(ctx context.Context, cfg connector.Config) error { return nil }
func (f *fakeConnectorsRepo) Delete(ctx context.Context, connectorID uuid.UUID) error {
	return nil
}

func TestConnectorsList(t *testing.T) {
	h := NewConnectors(&fakeConnectorsRepo{})
	req := httptest.NewRequest(http.MethodGet, "/admin/connectors", nil)
	w := httptest.NewRecorder()

	h.List(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestConnectorsSave(t *testing.T) {
	h := NewConnectors(&fakeConnectorsRepo{})
	body := `{"name":"SMB","type":"smb","settings":{"path":"/share"},"sync_interval_seconds":60,"is_active":true}`
	req := httptest.NewRequest(http.MethodPost, "/admin/connectors", strings.NewReader(body))
	w := httptest.NewRecorder()

	h.Save(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
