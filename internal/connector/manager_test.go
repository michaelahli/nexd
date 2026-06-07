package connector

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeConnectorRepository struct {
	configs       []Config
	updatedID     uuid.UUID
	updatedAt     time.Time
	updateCalls   int
	loadCallCount int
}

func (r *fakeConnectorRepository) ActiveConnectorConfigs(ctx context.Context) ([]Config, error) {
	r.loadCallCount++
	return append([]Config(nil), r.configs...), nil
}

func (r *fakeConnectorRepository) UpdateLastSyncAt(ctx context.Context, connectorID uuid.UUID, syncedAt time.Time) error {
	r.updatedID = connectorID
	r.updatedAt = syncedAt
	r.updateCalls++
	return nil
}

type fakeConnector struct {
	name           string
	connectorType  string
	validateCalls  int
	startCalls     int
	stopCalls      int
	healthErr      error
	fullDocs       []Document
	incrementDocs  []Document
	fullCalls      int
	incrementCalls int
}

func (c *fakeConnector) Name() string { return c.name }
func (c *fakeConnector) Type() string { return c.connectorType }
func (c *fakeConnector) Validate(ctx context.Context, cfg Config) error {
	c.validateCalls++
	return nil
}
func (c *fakeConnector) Start(ctx context.Context, cfg Config) error {
	c.startCalls++
	c.name = cfg.Name
	c.connectorType = cfg.Type
	return nil
}
func (c *fakeConnector) Stop(ctx context.Context) error {
	c.stopCalls++
	return nil
}
func (c *fakeConnector) Health(ctx context.Context) error { return c.healthErr }
func (c *fakeConnector) FullSync(ctx context.Context) (<-chan Document, <-chan error) {
	c.fullCalls++
	return docsChannel(c.fullDocs), errorChannel(nil)
}
func (c *fakeConnector) IncrementalSync(ctx context.Context, since time.Time) (<-chan Document, <-chan error) {
	c.incrementCalls++
	return docsChannel(c.incrementDocs), errorChannel(nil)
}

func docsChannel(docs []Document) <-chan Document {
	ch := make(chan Document, len(docs))
	for _, doc := range docs {
		ch <- doc
	}
	close(ch)
	return ch
}

func errorChannel(err error) <-chan error {
	ch := make(chan error, 1)
	if err != nil {
		ch <- err
	}
	close(ch)
	return ch
}

func TestManagerStartsAndStopsConnectors(t *testing.T) {
	id := uuid.New()
	fake := &fakeConnector{}
	repo := &fakeConnectorRepository{configs: []Config{{ID: id, Name: "drive", Type: "test", IsActive: true, SyncInterval: time.Hour}}}
	registry := NewRegistry()
	if err := registry.Register("test", func(cfg Config) (Connector, error) { return fake, nil }); err != nil {
		t.Fatalf("register: %v", err)
	}
	manager := NewManager(repo, registry)

	if err := manager.Start(context.Background()); err != nil {
		t.Fatalf("start manager: %v", err)
	}
	if fake.validateCalls != 1 || fake.startCalls != 1 {
		t.Fatalf("expected validate/start calls, got %d/%d", fake.validateCalls, fake.startCalls)
	}
	status, err := manager.Status(id)
	if err != nil || status != StatusRunning {
		t.Fatalf("expected running status, got %q err=%v", status, err)
	}

	if err := manager.Stop(context.Background()); err != nil {
		t.Fatalf("stop manager: %v", err)
	}
	if fake.stopCalls != 1 {
		t.Fatalf("expected one stop call, got %d", fake.stopCalls)
	}
}

func TestManagerSyncNowRunsFullThenIncremental(t *testing.T) {
	id := uuid.New()
	now := time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC)
	fake := &fakeConnector{fullDocs: []Document{{SourceID: "1"}}, incrementDocs: []Document{{SourceID: "2"}, {SourceID: "3"}}}
	repo := &fakeConnectorRepository{}
	registry := NewRegistry()
	if err := registry.Register("test", func(cfg Config) (Connector, error) { return fake, nil }); err != nil {
		t.Fatalf("register: %v", err)
	}
	manager := NewManager(repo, registry)
	manager.now = func() time.Time { return now }

	cfg := Config{ID: id, Name: "drive", Type: "test", IsActive: true, SyncInterval: time.Hour}
	if err := manager.StartConnector(context.Background(), cfg); err != nil {
		t.Fatalf("start connector: %v", err)
	}
	defer manager.Stop(context.Background())

	first, err := manager.SyncNow(context.Background(), id, false)
	if err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if !first.Full || first.DocumentsProcessed != 1 || fake.fullCalls != 1 {
		t.Fatalf("unexpected first sync: %#v fullCalls=%d", first, fake.fullCalls)
	}

	second, err := manager.SyncNow(context.Background(), id, false)
	if err != nil {
		t.Fatalf("second sync: %v", err)
	}
	if second.Full || second.DocumentsProcessed != 2 || fake.incrementCalls != 1 {
		t.Fatalf("unexpected second sync: %#v incrementCalls=%d", second, fake.incrementCalls)
	}
	if repo.updateCalls != 2 || repo.updatedID != id {
		t.Fatalf("expected sync timestamp updates, got calls=%d id=%s", repo.updateCalls, repo.updatedID)
	}
}

func TestManagerHealthMarksUnhealthy(t *testing.T) {
	id := uuid.New()
	fake := &fakeConnector{healthErr: errors.New("down")}
	registry := NewRegistry()
	if err := registry.Register("test", func(cfg Config) (Connector, error) { return fake, nil }); err != nil {
		t.Fatalf("register: %v", err)
	}
	manager := NewManager(nil, registry)
	if err := manager.StartConnector(context.Background(), Config{ID: id, Name: "drive", Type: "test"}); err != nil {
		t.Fatalf("start connector: %v", err)
	}
	defer manager.Stop(context.Background())

	if err := manager.Health(context.Background(), id); err == nil {
		t.Fatal("expected health error")
	}
	status, err := manager.Status(id)
	if status != StatusUnhealthy || err == nil {
		t.Fatalf("expected unhealthy status with error, got %q err=%v", status, err)
	}
}
