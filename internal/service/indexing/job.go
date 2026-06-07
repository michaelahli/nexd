package indexing

import (
	"time"

	"github.com/google/uuid"
)

const (
	JobTypeFullSync        = "full_sync"
	JobTypeIncrementalSync = "incremental_sync"

	JobStatusPending    = "pending"
	JobStatusProcessing = "processing"
	JobStatusCompleted  = "completed"
	JobStatusFailed     = "failed"
)

// Job describes an indexing job stored in the database queue.
type Job struct {
	ID                 uuid.UUID
	ConnectorID        uuid.UUID
	Type               string
	Status             string
	Attempts           int
	Error              string
	DocumentsProcessed int
	ScheduledAt        time.Time
	StartedAt          *time.Time
	CompletedAt        *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
