package connector

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Registry maps connector types to factories.
type Registry struct {
	mu        sync.RWMutex
	factories map[string]Factory
}

// NewRegistry creates an empty connector registry.
func NewRegistry() *Registry {
	return &Registry{factories: make(map[string]Factory)}
}

// Register stores a connector factory for a type.
func (r *Registry) Register(connectorType string, factory Factory) error {
	if r == nil {
		return fmt.Errorf("connector registry is not configured")
	}
	connectorType = normalizeType(connectorType)
	if connectorType == "" {
		return fmt.Errorf("connector type is required")
	}
	if factory == nil {
		return fmt.Errorf("connector factory is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.factories[connectorType]; exists {
		return fmt.Errorf("connector type %q is already registered", connectorType)
	}
	r.factories[connectorType] = factory
	return nil
}

// Create builds a connector for a configuration.
func (r *Registry) Create(cfg Config) (Connector, error) {
	if r == nil {
		return nil, fmt.Errorf("connector registry is not configured")
	}
	connectorType := normalizeType(cfg.Type)

	r.mu.RLock()
	factory, ok := r.factories[connectorType]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("connector type %q is not registered", cfg.Type)
	}

	return factory(cfg)
}

// Types returns registered connector types in sorted order.
func (r *Registry) Types() []string {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	types := make([]string, 0, len(r.factories))
	for connectorType := range r.factories {
		types = append(types, connectorType)
	}
	sort.Strings(types)
	return types
}

func normalizeType(connectorType string) string {
	return strings.ToLower(strings.TrimSpace(connectorType))
}
