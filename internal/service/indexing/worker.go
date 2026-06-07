package indexing

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/michaelahli/nexd/internal/connector"
	"github.com/michaelahli/nexd/internal/service/embedding"
)

// DocumentRepository persists indexed documents and embeddings.
type DocumentRepository interface {
	UpsertDocument(ctx context.Context, doc Document) (uuid.UUID, error)
	ReplaceEmbeddings(ctx context.Context, documentID uuid.UUID, chunks []EmbeddingChunk) error
}

// PermissionTarget describes an ACL entry to sync for an indexed document.
type PermissionTarget struct {
	UserID         *uuid.UUID
	GroupID        *uuid.UUID
	PermissionType string
}

// PermissionSyncer syncs source ACLs for indexed documents.
type PermissionSyncer interface {
	SyncDocumentPermissions(ctx context.Context, documentID uuid.UUID, targets []PermissionTarget) error
}

// DocumentSource returns source documents for an indexing job.
type DocumentSource interface {
	Documents(ctx context.Context, job Job) (<-chan connector.Document, <-chan error)
}

// Embedder generates embeddings for document text.
type Embedder interface {
	EmbedText(ctx context.Context, text string) ([]embedding.ChunkEmbedding, error)
}

// Processor executes one indexing job.
type Processor struct {
	documents   DocumentRepository
	permissions PermissionSyncer
	source      DocumentSource
	embedder    Embedder
}

// ProcessorOptions configures a Processor.
type ProcessorOptions struct {
	Documents   DocumentRepository
	Permissions PermissionSyncer
	Source      DocumentSource
	Embedder    Embedder
}

// NewProcessor creates a job processor.
func NewProcessor(opts ProcessorOptions) *Processor {
	return &Processor{documents: opts.Documents, permissions: opts.Permissions, source: opts.Source, embedder: opts.Embedder}
}

// Document is the database representation of a source document.
type Document struct {
	ConnectorID     uuid.UUID
	SourceType      string
	SourceID        string
	Title           string
	Content         string
	Metadata        map[string]any
	FilePath        string
	FileSize        int64
	MIMEType        string
	SourceUpdatedAt *time.Time
	IndexedAt       time.Time
}

// EmbeddingChunk is a stored embedding chunk.
type EmbeddingChunk struct {
	Index  int
	Text   string
	Vector embedding.Vector
}

// Process runs fetch, persist, permission sync, chunk, embed, and embedding storage.
func (p *Processor) Process(ctx context.Context, job Job) (int, error) {
	if p == nil || p.documents == nil || p.source == nil || p.embedder == nil {
		return 0, fmt.Errorf("indexing processor is not configured")
	}

	docs, errs := p.source.Documents(ctx, job)
	processed := 0
	for docs != nil || errs != nil {
		select {
		case <-ctx.Done():
			return processed, ctx.Err()
		case doc, ok := <-docs:
			if !ok {
				docs = nil
				continue
			}
			if err := p.processDocument(ctx, job, doc); err != nil {
				return processed, err
			}
			processed++
		case err, ok := <-errs:
			if !ok {
				errs = nil
				continue
			}
			if err != nil {
				return processed, err
			}
		}
	}
	return processed, nil
}

func (p *Processor) processDocument(ctx context.Context, job Job, sourceDoc connector.Document) error {
	if sourceDoc.SourceID == "" {
		return fmt.Errorf("source document ID is required")
	}
	if sourceDoc.SourceType == "" {
		sourceDoc.SourceType = "unknown"
	}
	if sourceDoc.Title == "" {
		sourceDoc.Title = sourceDoc.SourceID
	}

	documentID, err := p.documents.UpsertDocument(ctx, Document{
		ConnectorID:     job.ConnectorID,
		SourceType:      sourceDoc.SourceType,
		SourceID:        sourceDoc.SourceID,
		Title:           sourceDoc.Title,
		Content:         sourceDoc.Content,
		Metadata:        sourceDoc.Metadata,
		FilePath:        sourceDoc.FilePath,
		FileSize:        sourceDoc.FileSize,
		MIMEType:        sourceDoc.MIMEType,
		SourceUpdatedAt: sourceDoc.SourceUpdatedAt,
		IndexedAt:       time.Now(),
	})
	if err != nil {
		return fmt.Errorf("upsert document %q: %w", sourceDoc.SourceID, err)
	}

	if p.permissions != nil && len(sourceDoc.Permissions) > 0 {
		if err := p.permissions.SyncDocumentPermissions(ctx, documentID, permissionTargets(sourceDoc.Permissions)); err != nil {
			return fmt.Errorf("sync document permissions %q: %w", sourceDoc.SourceID, err)
		}
	}

	chunks, err := p.embedder.EmbedText(ctx, sourceDoc.Content)
	if err != nil {
		return fmt.Errorf("embed document %q: %w", sourceDoc.SourceID, err)
	}
	storedChunks := make([]EmbeddingChunk, 0, len(chunks))
	for _, chunk := range chunks {
		storedChunks = append(storedChunks, EmbeddingChunk{Index: chunk.Index, Text: chunk.Text, Vector: chunk.Vector})
	}
	if err := p.documents.ReplaceEmbeddings(ctx, documentID, storedChunks); err != nil {
		return fmt.Errorf("store embeddings %q: %w", sourceDoc.SourceID, err)
	}

	return nil
}

func permissionTargets(targets []connector.PermissionTarget) []PermissionTarget {
	converted := make([]PermissionTarget, len(targets))
	for i, target := range targets {
		converted[i] = PermissionTarget{UserID: target.UserID, GroupID: target.GroupID, PermissionType: target.PermissionType}
	}
	return converted
}
