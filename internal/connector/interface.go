package connector

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Status describes connector health and lifecycle state.
type Status string

const (
	StatusStopped   Status = "stopped"
	StatusStarting  Status = "starting"
	StatusRunning   Status = "running"
	StatusUnhealthy Status = "unhealthy"
)

// Config is a connector configuration loaded from storage.
type Config struct {
	ID           uuid.UUID
	Name         string
	Type         string
	Settings     map[string]any
	SyncInterval time.Duration
	LastSyncAt   *time.Time
	IsActive     bool
}

// PermissionTarget describes a source ACL entry for a document.
type PermissionTarget struct {
	UserID         *uuid.UUID
	GroupID        *uuid.UUID
	PermissionType string
}

// Document is a source document emitted by connectors during sync.
type Document struct {
	SourceType      string
	SourceID        string
	Title           string
	Content         string
	Metadata        map[string]any
	FilePath        string
	FileSize        int64
	MIMEType        string
	SourceUpdatedAt *time.Time
	Permissions     []PermissionTarget
}

// SyncResult summarizes a sync run.
type SyncResult struct {
	ConnectorID        uuid.UUID
	ConnectorName      string
	Full               bool
	DocumentsProcessed int
	StartedAt          time.Time
	CompletedAt        time.Time
}

// Connector defines the lifecycle and sync contract for knowledge sources.
type Connector interface {
	Name() string
	Type() string
	Validate(ctx context.Context, cfg Config) error
	Start(ctx context.Context, cfg Config) error
	Stop(ctx context.Context) error
	Health(ctx context.Context) error
	FullSync(ctx context.Context) (<-chan Document, <-chan error)
	IncrementalSync(ctx context.Context, since time.Time) (<-chan Document, <-chan error)
}

// Factory creates a connector for a stored configuration.
type Factory func(Config) (Connector, error)
