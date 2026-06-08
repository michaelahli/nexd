package search

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/michaelahli/nexd/internal/service/embedding"
)

// Query describes a search request.
type Query struct {
	Text   string
	UserID uuid.UUID
	Limit  int
	Offset int
}

// Result is a single search result.
type Result struct {
	DocumentID      uuid.UUID
	SourceType      string
	SourceID        string
	Title           string
	Snippet         string
	Score           float64
	FilePath        string
	MIMEType        string
	SourceUpdatedAt *time.Time
	IndexedAt       time.Time
}

// Response bundles search results and metadata.
type Response struct {
	Results    []Result
	TotalCount int
	Query      string
	Limit      int
	Offset     int
}

// Embedder generates query embeddings.
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([]embedding.Vector, error)
}

// PermissionChecker verifies user access to documents.
type PermissionChecker interface {
	CanAccessDocument(ctx context.Context, userID, documentID uuid.UUID, action string) (bool, error)
}
