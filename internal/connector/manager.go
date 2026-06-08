package connector

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

const DefaultSyncInterval = 15 * time.Minute

// Repository is the storage contract required by Manager.
type Repository interface {
	ActiveConnectorConfigs(ctx context.Context) ([]Config, error)
	UpdateLastSyncAt(ctx context.Context, connectorID uuid.UUID, syncedAt time.Time) error
}

// Manager controls connector lifecycle and scheduled syncs.
type Manager struct {
	repo     Repository
	registry *Registry
	now      func() time.Time

	mu      sync.RWMutex
	running map[uuid.UUID]*runningConnector
}

type runningConnector struct {
	config    Config
	connector Connector
	status    Status
	lastError error
	cancel    context.CancelFunc
	done      chan struct{}
}

// NewManager creates a connector manager.
func NewManager(repo Repository, registry *Registry) *Manager {
	if registry == nil {
		registry = NewRegistry()
	}
	return &Manager{repo: repo, registry: registry, now: time.Now, running: make(map[uuid.UUID]*runningConnector)}
}

// Start loads active connectors and starts their scheduled sync loops.
func (m *Manager) Start(ctx context.Context) error {
	if m == nil || m.repo == nil {
		return fmt.Errorf("connector manager is not configured")
	}

	configs, err := m.repo.ActiveConnectorConfigs(ctx)
	if err != nil {
		return fmt.Errorf("load connector configs: %w", err)
	}

	for _, cfg := range configs {
		if err := m.StartConnector(ctx, cfg); err != nil {
			return err
		}
	}
	return nil
}

// StartConnector starts one connector from config.
func (m *Manager) StartConnector(ctx context.Context, cfg Config) error {
	if m == nil || m.registry == nil {
		return fmt.Errorf("connector manager is not configured")
	}
	if err := validateConfig(cfg); err != nil {
		return err
	}

	m.mu.Lock()
	if _, exists := m.running[cfg.ID]; exists {
		m.mu.Unlock()
		return fmt.Errorf("connector %s is already running", cfg.ID)
	}
	m.mu.Unlock()

	conn, err := m.registry.Create(cfg)
	if err != nil {
		return err
	}
	if err := conn.Validate(ctx, cfg); err != nil {
		return fmt.Errorf("validate connector %q: %w", cfg.Name, err)
	}
	if err := conn.Start(ctx, cfg); err != nil {
		return fmt.Errorf("start connector %q: %w", cfg.Name, err)
	}

	loopCtx, cancel := context.WithCancel(context.Background())
	running := &runningConnector{config: cfg, connector: conn, status: StatusRunning, cancel: cancel, done: make(chan struct{})}

	m.mu.Lock()
	m.running[cfg.ID] = running
	m.mu.Unlock()

	go m.syncLoop(loopCtx, running)
	return nil
}

// Stop stops all running connectors.
func (m *Manager) Stop(ctx context.Context) error {
	if m == nil {
		return nil
	}

	m.mu.RLock()
	ids := make([]uuid.UUID, 0, len(m.running))
	for id := range m.running {
		ids = append(ids, id)
	}
	m.mu.RUnlock()

	var joined error
	for _, id := range ids {
		if err := m.StopConnector(ctx, id); err != nil {
			joined = errors.Join(joined, err)
		}
	}
	return joined
}

// StopConnector stops one running connector.
func (m *Manager) StopConnector(ctx context.Context, connectorID uuid.UUID) error {
	if m == nil {
		return nil
	}

	m.mu.Lock()
	running, ok := m.running[connectorID]
	if ok {
		delete(m.running, connectorID)
	}
	m.mu.Unlock()
	if !ok {
		return nil
	}

	running.cancel()
	select {
	case <-running.done:
	case <-ctx.Done():
		return ctx.Err()
	}

	if err := running.connector.Stop(ctx); err != nil {
		return fmt.Errorf("stop connector %q: %w", running.config.Name, err)
	}
	return nil
}

// Health checks a running connector.
func (m *Manager) Health(ctx context.Context, connectorID uuid.UUID) error {
	running, ok := m.get(connectorID)
	if !ok {
		return fmt.Errorf("connector %s is not running", connectorID)
	}
	if err := running.connector.Health(ctx); err != nil {
		m.setStatus(connectorID, StatusUnhealthy, err)
		return err
	}
	m.setStatus(connectorID, StatusRunning, nil)
	return nil
}

// Status returns the current connector status.
func (m *Manager) Status(connectorID uuid.UUID) (Status, error) {
	running, ok := m.get(connectorID)
	if !ok {
		return StatusStopped, nil
	}
	return running.status, running.lastError
}

// SyncNow runs a sync for a connector immediately.
func (m *Manager) SyncNow(ctx context.Context, connectorID uuid.UUID, full bool) (SyncResult, error) {
	running, ok := m.get(connectorID)
	if !ok {
		return SyncResult{}, fmt.Errorf("connector %s is not running", connectorID)
	}
	return m.sync(ctx, running, full)
}

// Running returns a currently running connector instance.
func (m *Manager) Running(connectorID uuid.UUID) (Connector, bool) {
	running, ok := m.get(connectorID)
	if !ok {
		return nil, false
	}
	return running.connector, true
}

// Config returns the last known connector config.
func (m *Manager) Config(connectorID uuid.UUID) (Config, bool) {
	running, ok := m.get(connectorID)
	if !ok {
		return Config{}, false
	}
	return running.config, true
}

func (m *Manager) syncLoop(ctx context.Context, running *runningConnector) {
	defer close(running.done)

	interval := running.config.SyncInterval
	if interval <= 0 {
		interval = DefaultSyncInterval
	}
	timer := time.NewTimer(interval)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			_, err := m.sync(ctx, running, false)
			if err != nil {
				m.setStatus(running.config.ID, StatusUnhealthy, err)
			} else {
				m.setStatus(running.config.ID, StatusRunning, nil)
			}
			timer.Reset(interval)
		}
	}
}

func (m *Manager) sync(ctx context.Context, running *runningConnector, full bool) (SyncResult, error) {
	startedAt := m.now()
	var docs <-chan Document
	var errs <-chan error
	if full || running.config.LastSyncAt == nil {
		docs, errs = running.connector.FullSync(ctx)
		full = true
	} else {
		docs, errs = running.connector.IncrementalSync(ctx, *running.config.LastSyncAt)
	}

	processed := 0
	for docs != nil || errs != nil {
		select {
		case <-ctx.Done():
			return SyncResult{}, ctx.Err()
		case _, ok := <-docs:
			if !ok {
				docs = nil
				continue
			}
			processed++
		case err, ok := <-errs:
			if !ok {
				errs = nil
				continue
			}
			if err != nil {
				return SyncResult{}, err
			}
		}
	}

	completedAt := m.now()
	if m.repo != nil {
		if err := m.repo.UpdateLastSyncAt(ctx, running.config.ID, completedAt); err != nil {
			return SyncResult{}, fmt.Errorf("update connector sync timestamp: %w", err)
		}
	}
	running.config.LastSyncAt = &completedAt

	return SyncResult{
		ConnectorID:        running.config.ID,
		ConnectorName:      running.config.Name,
		Full:               full,
		DocumentsProcessed: processed,
		StartedAt:          startedAt,
		CompletedAt:        completedAt,
	}, nil
}

func (m *Manager) get(connectorID uuid.UUID) (*runningConnector, bool) {
	if m == nil {
		return nil, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	running, ok := m.running[connectorID]
	return running, ok
}

func (m *Manager) setStatus(connectorID uuid.UUID, status Status, err error) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if running, ok := m.running[connectorID]; ok {
		running.status = status
		running.lastError = err
	}
}

func validateConfig(cfg Config) error {
	if cfg.ID == uuid.Nil {
		return fmt.Errorf("connector ID is required")
	}
	if cfg.Name == "" {
		return fmt.Errorf("connector name is required")
	}
	if cfg.Type == "" {
		return fmt.Errorf("connector type is required")
	}
	if cfg.SyncInterval < 0 {
		return fmt.Errorf("connector sync interval must not be negative")
	}
	return nil
}
