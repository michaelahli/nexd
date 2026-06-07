package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/michaelahli/nexd/internal/service/indexing"
)

const DefaultMaxJobAttempts = 3

// SyncJobRepository manages PostgreSQL-backed indexing jobs.
type SyncJobRepository struct {
	pool        *pgxpool.Pool
	maxAttempts int
}

// NewSyncJobRepository creates a sync job repository.
func NewSyncJobRepository(pool *pgxpool.Pool) *SyncJobRepository {
	return &SyncJobRepository{pool: pool, maxAttempts: DefaultMaxJobAttempts}
}

// Enqueue creates a pending sync job.
func (r *SyncJobRepository) Enqueue(ctx context.Context, connectorID uuid.UUID, jobType string, scheduledAt time.Time) (uuid.UUID, error) {
	if r == nil || r.pool == nil {
		return uuid.Nil, fmt.Errorf("sync job repository is not configured")
	}
	if jobType == "" {
		jobType = indexing.JobTypeIncrementalSync
	}
	if scheduledAt.IsZero() {
		scheduledAt = time.Now()
	}

	var jobID uuid.UUID
	if err := r.pool.QueryRow(ctx, `
		INSERT INTO sync_jobs (connector_id, type, status, scheduled_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, connectorID, jobType, indexing.JobStatusPending, scheduledAt).Scan(&jobID); err != nil {
		return uuid.Nil, fmt.Errorf("enqueue sync job: %w", err)
	}
	return jobID, nil
}

// Next claims the next pending job, if one is available.
func (r *SyncJobRepository) Next(ctx context.Context) (indexing.Job, bool, error) {
	if r == nil || r.pool == nil {
		return indexing.Job{}, false, fmt.Errorf("sync job repository is not configured")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return indexing.Job{}, false, fmt.Errorf("begin claim sync job: %w", err)
	}
	defer tx.Rollback(ctx)

	var job indexing.Job
	err = tx.QueryRow(ctx, `
		SELECT id, connector_id, type, status, attempts, COALESCE(error, ''), documents_processed,
		       scheduled_at, started_at, completed_at, created_at, updated_at
		FROM sync_jobs
		WHERE status = $1 AND scheduled_at <= NOW()
		ORDER BY scheduled_at ASC, created_at ASC
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`, indexing.JobStatusPending).Scan(
		&job.ID,
		&job.ConnectorID,
		&job.Type,
		&job.Status,
		&job.Attempts,
		&job.Error,
		&job.DocumentsProcessed,
		&job.ScheduledAt,
		&job.StartedAt,
		&job.CompletedAt,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return indexing.Job{}, false, nil
		}
		return indexing.Job{}, false, fmt.Errorf("select sync job: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE sync_jobs
		SET status = $2, attempts = attempts + 1, started_at = COALESCE(started_at, NOW()), updated_at = NOW()
		WHERE id = $1
	`, job.ID, indexing.JobStatusProcessing); err != nil {
		return indexing.Job{}, false, fmt.Errorf("claim sync job: %w", err)
	}
	job.Status = indexing.JobStatusProcessing
	job.Attempts++

	if err := tx.Commit(ctx); err != nil {
		return indexing.Job{}, false, fmt.Errorf("commit sync job claim: %w", err)
	}
	return job, true, nil
}

// Complete marks a sync job as completed.
func (r *SyncJobRepository) Complete(ctx context.Context, jobID uuid.UUID, documentsProcessed int) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("sync job repository is not configured")
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE sync_jobs
		SET status = $2, documents_processed = $3, error = NULL, completed_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, jobID, indexing.JobStatusCompleted, documentsProcessed)
	if err != nil {
		return fmt.Errorf("complete sync job: %w", err)
	}
	return nil
}

// Fail marks a sync job as failed or returns it to pending for retry.
func (r *SyncJobRepository) Fail(ctx context.Context, jobID uuid.UUID, jobErr error) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("sync job repository is not configured")
	}
	message := "unknown error"
	if jobErr != nil {
		message = jobErr.Error()
	}

	_, err := r.pool.Exec(ctx, `
		UPDATE sync_jobs
		SET status = CASE WHEN attempts >= $3 THEN $2 ELSE $4 END,
		    error = $5,
		    updated_at = NOW(),
		    completed_at = CASE WHEN attempts >= $3 THEN NOW() ELSE completed_at END
		WHERE id = $1
	`, jobID, indexing.JobStatusFailed, r.maxAttempts, indexing.JobStatusPending, message)
	if err != nil {
		return fmt.Errorf("fail sync job: %w", err)
	}
	return nil
}
