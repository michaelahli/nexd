package connector

import (
	"testing"

	"github.com/google/uuid"
)

func TestRegistryRegistersAndCreatesConnectors(t *testing.T) {
	registry := NewRegistry()
	created := false

	if err := registry.Register(" Test ", func(cfg Config) (Connector, error) {
		created = true
		return &fakeConnector{name: cfg.Name, connectorType: cfg.Type}, nil
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	conn, err := registry.Create(Config{ID: uuid.New(), Name: "alpha", Type: "test"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if conn.Name() != "alpha" || !created {
		t.Fatalf("unexpected connector: %#v created=%v", conn, created)
	}
	if types := registry.Types(); len(types) != 1 || types[0] != "test" {
		t.Fatalf("unexpected types: %#v", types)
	}
}

func TestRegistryRejectsDuplicates(t *testing.T) {
	registry := NewRegistry()
	factory := func(cfg Config) (Connector, error) { return &fakeConnector{}, nil }
	if err := registry.Register("test", factory); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := registry.Register("TEST", factory); err == nil {
		t.Fatal("expected duplicate registration error")
	}
}
