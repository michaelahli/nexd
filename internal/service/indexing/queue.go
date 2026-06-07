package indexing

import (
	"context"

	"github.com/google/uuid"
)

// Queue is the job queue contract required by Service.
type Queue interface {
	Next(ctx context.Context) (Job, bool, error)
	Complete(ctx context.Context, jobID uuid.UUID, documentsProcessed int) error
	Fail(ctx context.Context, jobID uuid.UUID, jobErr error) error
}
