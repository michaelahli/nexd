package indexing

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/michaelahli/nexd/internal/connector"
)

// ConnectorProvider resolves connector config and instances for indexing jobs.
type ConnectorProvider interface {
	Config(connectorID uuid.UUID) (connector.Config, bool)
	Running(connectorID uuid.UUID) (connector.Connector, bool)
}

// ConnectorSource adapts running connectors to the indexing DocumentSource contract.
type ConnectorSource struct {
	provider ConnectorProvider
}

// NewConnectorSource creates a connector-backed indexing source.
func NewConnectorSource(provider ConnectorProvider) *ConnectorSource {
	return &ConnectorSource{provider: provider}
}

// Documents loads job documents from the currently running connector instance.
func (s *ConnectorSource) Documents(ctx context.Context, job Job) (<-chan connector.Document, <-chan error) {
	docs := make(chan connector.Document)
	errs := make(chan error, 1)

	go func() {
		defer close(docs)
		defer close(errs)

		if s == nil || s.provider == nil {
			errs <- fmt.Errorf("connector source is not configured")
			return
		}
		cfg, ok := s.provider.Config(job.ConnectorID)
		if !ok {
			errs <- fmt.Errorf("connector config %s is not available", job.ConnectorID)
			return
		}
		running, ok := s.provider.Running(job.ConnectorID)
		if !ok {
			errs <- fmt.Errorf("connector %s is not running", job.ConnectorID)
			return
		}

		var srcDocs <-chan connector.Document
		var srcErrs <-chan error
		if job.Type == JobTypeFullSync || cfg.LastSyncAt == nil {
			srcDocs, srcErrs = running.FullSync(ctx)
		} else {
			srcDocs, srcErrs = running.IncrementalSync(ctx, *cfg.LastSyncAt)
		}

		for srcDocs != nil || srcErrs != nil {
			select {
			case <-ctx.Done():
				errs <- ctx.Err()
				return
			case doc, ok := <-srcDocs:
				if !ok {
					srcDocs = nil
					continue
				}
				select {
				case <-ctx.Done():
					errs <- ctx.Err()
					return
				case docs <- doc:
				}
			case err, ok := <-srcErrs:
				if !ok {
					srcErrs = nil
					continue
				}
				if err != nil {
					errs <- err
					return
				}
			}
		}
	}()

	return docs, errs
}

// StaticConnectorProvider is a simple in-memory provider used by application composition.
type StaticConnectorProvider struct {
	configs map[uuid.UUID]connector.Config
	running map[uuid.UUID]connector.Connector
}

// NewStaticConnectorProvider creates an empty in-memory connector provider.
func NewStaticConnectorProvider() *StaticConnectorProvider {
	return &StaticConnectorProvider{configs: make(map[uuid.UUID]connector.Config), running: make(map[uuid.UUID]connector.Connector)}
}

// Set stores connector config and instance.
func (p *StaticConnectorProvider) Set(cfg connector.Config, instance connector.Connector) {
	if p == nil {
		return
	}
	p.configs[cfg.ID] = cfg
	p.running[cfg.ID] = instance
}

// Config returns stored connector config.
func (p *StaticConnectorProvider) Config(connectorID uuid.UUID) (connector.Config, bool) {
	cfg, ok := p.configs[connectorID]
	return cfg, ok
}

// Running returns the stored running connector instance.
func (p *StaticConnectorProvider) Running(connectorID uuid.UUID) (connector.Connector, bool) {
	instance, ok := p.running[connectorID]
	return instance, ok
}

// UpdateLastSync stores the new last-sync timestamp in memory.
func (p *StaticConnectorProvider) UpdateLastSync(connectorID uuid.UUID, syncedAt time.Time) {
	if p == nil {
		return
	}
	cfg, ok := p.configs[connectorID]
	if !ok {
		return
	}
	cfg.LastSyncAt = &syncedAt
	p.configs[connectorID] = cfg
}
