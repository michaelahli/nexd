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
	return []connector.Config{{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111"), Name: "SMB", Type: "smb", SyncInterval: time.Minute, IsActive: true}}, nil
}
func (f *fakeConnectorsRepo) Save(ctx context.Context, cfg connector.Config) error { return nil }
func (f *fakeConnectorsRepo) Delete(ctx context.Context, connectorID uuid.UUID) error {
	return nil
}

type fakeSyncJobs struct{}

func (f *fakeSyncJobs) Enqueue(ctx context.Context, connectorID uuid.UUID, jobType string, scheduledAt time.Time) (uuid.UUID, error) {
	return uuid.MustParse("22222222-2222-2222-2222-222222222222"), nil
}

type fakeConnectorOps struct{}

func (f *fakeConnectorOps) StartConnector(ctx context.Context, cfg connector.Config) error {
	return nil
}
func (f *fakeConnectorOps) StopConnector(ctx context.Context, connectorID uuid.UUID) error {
	return nil
}
func (f *fakeConnectorOps) Health(ctx context.Context, connectorID uuid.UUID) error { return nil }

func TestConnectorsList(t *testing.T) {
	h := NewConnectors(&fakeConnectorsRepo{}, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/connectors", nil)
	w := httptest.NewRecorder()

	h.List(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestConnectorsSave(t *testing.T) {
	h := NewConnectors(&fakeConnectorsRepo{}, nil, nil)
	body := `{"name":"SMB","type":"smb","settings":{"path":"/share"},"sync_interval_seconds":60,"is_active":true}`
	req := httptest.NewRequest(http.MethodPost, "/admin/connectors", strings.NewReader(body))
	w := httptest.NewRecorder()

	h.Save(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestConnectorsTest(t *testing.T) {
	h := NewConnectors(&fakeConnectorsRepo{}, nil, &fakeConnectorOps{})
	req := httptest.NewRequest(http.MethodPost, "/admin/connectors/test?id=11111111-1111-1111-1111-111111111111", nil)
	w := httptest.NewRecorder()

	h.Test(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestConnectorsTriggerSync(t *testing.T) {
	h := NewConnectors(&fakeConnectorsRepo{}, &fakeSyncJobs{}, nil)
	body := `{"connector_id":"11111111-1111-1111-1111-111111111111","full":true}`
	req := httptest.NewRequest(http.MethodPost, "/admin/connectors/sync", strings.NewReader(body))
	w := httptest.NewRecorder()

	h.TriggerSync(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", w.Code)
	}
}
