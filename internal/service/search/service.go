package search

import (
	"context"
	"fmt"
)

const DefaultLimit = 10

// Repository is the persistence contract for search.
type Repository interface {
	VectorSearch(ctx context.Context, vector []float32, limit, offset int) ([]Result, int, error)
}

// Service performs permission-aware document search.
type Service struct {
	repo        Repository
	embedder    Embedder
	permissions PermissionChecker
}

// NewService creates a search service.
func NewService(repo Repository, embedder Embedder, permissions PermissionChecker) *Service {
	return &Service{repo: repo, embedder: embedder, permissions: permissions}
}

// Search embeds the query and retrieves permission-filtered results.
func (s *Service) Search(ctx context.Context, query Query) (Response, error) {
	if s == nil || s.repo == nil || s.embedder == nil {
		return Response{}, fmt.Errorf("search service is not configured")
	}
	if query.Text == "" {
		return Response{Query: query.Text, Results: []Result{}, TotalCount: 0, Limit: query.Limit, Offset: query.Offset}, nil
	}
	if query.Limit <= 0 {
		query.Limit = DefaultLimit
	}
	if query.Offset < 0 {
		query.Offset = 0
	}

	vectors, err := s.embedder.Embed(ctx, []string{query.Text})
	if err != nil {
		return Response{}, fmt.Errorf("embed search query: %w", err)
	}
	if len(vectors) == 0 {
		return Response{}, fmt.Errorf("embedder returned no vectors")
	}

	candidateLimit := query.Limit * 3
	if s.permissions != nil {
		candidateLimit = query.Limit * 10
	}
	candidates, totalCount, err := s.repo.VectorSearch(ctx, vectors[0], candidateLimit, query.Offset)
	if err != nil {
		return Response{}, fmt.Errorf("vector search: %w", err)
	}

	filtered := make([]Result, 0, len(candidates))
	for _, candidate := range candidates {
		if s.permissions != nil {
			allowed, err := s.permissions.CanAccessDocument(ctx, query.UserID, candidate.DocumentID, "read")
			if err != nil {
				return Response{}, fmt.Errorf("check document permission: %w", err)
			}
			if !allowed {
				continue
			}
		}
		filtered = append(filtered, candidate)
		if len(filtered) >= query.Limit {
			break
		}
	}

	return Response{Results: filtered, TotalCount: totalCount, Query: query.Text, Limit: query.Limit, Offset: query.Offset}, nil
}
