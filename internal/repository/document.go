package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/michaelahli/nexd/internal/service/embedding"
	"github.com/michaelahli/nexd/internal/service/indexing"
)

// DocumentRepository persists indexed documents and embeddings.
type DocumentRepository struct {
	pool *pgxpool.Pool
}

// NewDocumentRepository creates a PostgreSQL-backed document repository.
func NewDocumentRepository(pool *pgxpool.Pool) *DocumentRepository {
	return &DocumentRepository{pool: pool}
}

// UpsertDocument inserts or updates a source document and returns its ID.
func (r *DocumentRepository) UpsertDocument(ctx context.Context, doc indexing.Document) (uuid.UUID, error) {
	if r == nil || r.pool == nil {
		return uuid.Nil, fmt.Errorf("document repository is not configured")
	}
	metadata := doc.Metadata
	if metadata == nil {
		metadata = make(map[string]any)
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return uuid.Nil, fmt.Errorf("marshal document metadata: %w", err)
	}

	var documentID uuid.UUID
	if err := r.pool.QueryRow(ctx, `
		INSERT INTO documents (
			connector_id, source_type, source_id, title, content, metadata, file_path,
			file_size, mime_type, source_updated_at, indexed_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (source_type, source_id) DO UPDATE SET
			connector_id = EXCLUDED.connector_id,
			title = EXCLUDED.title,
			content = EXCLUDED.content,
			metadata = EXCLUDED.metadata,
			file_path = EXCLUDED.file_path,
			file_size = EXCLUDED.file_size,
			mime_type = EXCLUDED.mime_type,
			source_updated_at = EXCLUDED.source_updated_at,
			indexed_at = EXCLUDED.indexed_at,
			updated_at = NOW()
		RETURNING id
	`, doc.ConnectorID, doc.SourceType, doc.SourceID, doc.Title, doc.Content, metadataJSON, doc.FilePath, doc.FileSize, doc.MIMEType, doc.SourceUpdatedAt, doc.IndexedAt).Scan(&documentID); err != nil {
		return uuid.Nil, fmt.Errorf("upsert document: %w", err)
	}
	return documentID, nil
}

// ReplaceEmbeddings replaces all stored embedding chunks for a document.
func (r *DocumentRepository) ReplaceEmbeddings(ctx context.Context, documentID uuid.UUID, chunks []indexing.EmbeddingChunk) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("document repository is not configured")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin embedding replacement: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM document_embeddings WHERE document_id = $1`, documentID); err != nil {
		return fmt.Errorf("delete document embeddings: %w", err)
	}
	for _, chunk := range chunks {
		if _, err := tx.Exec(ctx, `
			INSERT INTO document_embeddings (document_id, chunk_index, chunk_text, embedding)
			VALUES ($1, $2, $3, $4::vector)
		`, documentID, chunk.Index, chunk.Text, encodeVector(chunk.Vector)); err != nil {
			return fmt.Errorf("insert document embedding: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit embedding replacement: %w", err)
	}
	return nil
}

func encodeVector(vector embedding.Vector) string {
	parts := make([]string, len(vector))
	for i, value := range vector {
		parts[i] = strconv.FormatFloat(float64(value), 'f', -1, 32)
	}
	return "[" + strings.Join(parts, ",") + "]"
}
