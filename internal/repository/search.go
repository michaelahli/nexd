package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/michaelahli/nexd/internal/service/search"
)

// SearchRepository performs vector search over indexed documents.
type SearchRepository struct {
	pool *pgxpool.Pool
}

// NewSearchRepository creates a PostgreSQL-backed search repository.
func NewSearchRepository(pool *pgxpool.Pool) *SearchRepository {
	return &SearchRepository{pool: pool}
}

// VectorSearch performs cosine similarity search using pgvector.
func (r *SearchRepository) VectorSearch(ctx context.Context, vector []float32, limit, offset int) ([]search.Result, int, error) {
	if r == nil || r.pool == nil {
		return nil, 0, fmt.Errorf("search repository is not configured")
	}
	if len(vector) == 0 {
		return nil, 0, fmt.Errorf("search vector is required")
	}
	if limit <= 0 {
		limit = search.DefaultLimit
	}
	if offset < 0 {
		offset = 0
	}

	vectorText := EncodeVector(vector)
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT ON (d.id)
			d.id,
			d.source_type,
			d.source_id,
			d.title,
			COALESCE(LEFT(de.chunk_text, 200), '') AS snippet,
			1 - (de.embedding <=> $1::vector) AS score,
			d.file_path,
			d.mime_type,
			d.source_updated_at,
			d.indexed_at
		FROM document_embeddings de
		INNER JOIN documents d ON d.id = de.document_id
		WHERE de.embedding IS NOT NULL
		ORDER BY d.id, de.embedding <=> $1::vector ASC
		LIMIT $2 OFFSET $3
	`, vectorText, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("vector search query: %w", err)
	}
	defer rows.Close()

	results := make([]search.Result, 0, limit)
	for rows.Next() {
		var result search.Result
		if err := rows.Scan(
			&result.DocumentID,
			&result.SourceType,
			&result.SourceID,
			&result.Title,
			&result.Snippet,
			&result.Score,
			&result.FilePath,
			&result.MIMEType,
			&result.SourceUpdatedAt,
			&result.IndexedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan search result: %w", err)
		}
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate search results: %w", err)
	}

	var totalCount int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(DISTINCT d.id) FROM documents d INNER JOIN document_embeddings de ON de.document_id = d.id WHERE de.embedding IS NOT NULL`).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("count total documents: %w", err)
	}

	return results, totalCount, nil
}
