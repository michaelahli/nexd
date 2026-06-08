package chat

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/michaelahli/nexd/internal/service/search"
)

// SearchRetriever adapts the search service for RAG context retrieval.
type SearchRetriever struct {
	search interface {
		Search(ctx context.Context, query search.Query) (search.Response, error)
	}
}

// NewSearchRetriever creates a retriever backed by the search service.
func NewSearchRetriever(searchService interface {
	Search(ctx context.Context, query search.Query) (search.Response, error)
}) *SearchRetriever {
	return &SearchRetriever{search: searchService}
}

// Retrieve performs a search and converts results to RAG context documents.
func (r *SearchRetriever) Retrieve(ctx context.Context, userID uuid.UUID, query string, limit int) ([]RetrievedDocument, error) {
	if r == nil || r.search == nil {
		return nil, fmt.Errorf("search retriever is not configured")
	}
	if limit <= 0 {
		limit = DefaultContextLimit
	}

	response, err := r.search.Search(ctx, search.Query{Text: query, UserID: userID, Limit: limit})
	if err != nil {
		return nil, fmt.Errorf("search for context: %w", err)
	}

	docs := make([]RetrievedDocument, 0, len(response.Results))
	for _, result := range response.Results {
		docs = append(docs, RetrievedDocument{
			DocumentID: result.DocumentID,
			SourceType: result.SourceType,
			SourceID:   result.SourceID,
			Title:      result.Title,
			Content:    result.Snippet,
			Score:      result.Score,
		})
	}
	return docs, nil
}
