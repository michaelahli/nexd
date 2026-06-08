package search

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/michaelahli/nexd/internal/service/embedding"
)

type fakeSearchRepository struct {
	results    []Result
	totalCount int
}

func (r *fakeSearchRepository) VectorSearch(ctx context.Context, vector []float32, limit, offset int) ([]Result, int, error) {
	return r.results, r.totalCount, nil
}

type fakeEmbedder struct {
	vector embedding.Vector
}

func (e *fakeEmbedder) Embed(ctx context.Context, texts []string) ([]embedding.Vector, error) {
	return []embedding.Vector{e.vector}, nil
}

type fakePermissionChecker struct {
	allowed map[uuid.UUID]bool
}

func (c *fakePermissionChecker) CanAccessDocument(ctx context.Context, userID, documentID uuid.UUID, action string) (bool, error) {
	if c.allowed == nil {
		return true, nil
	}
	return c.allowed[documentID], nil
}

func TestServiceSearchFiltersUnauthorizedDocuments(t *testing.T) {
	userID := uuid.New()
	doc1 := uuid.New()
	doc2 := uuid.New()
	doc3 := uuid.New()
	repo := &fakeSearchRepository{
		results: []Result{
			{DocumentID: doc1, Title: "Document 1"},
			{DocumentID: doc2, Title: "Document 2"},
			{DocumentID: doc3, Title: "Document 3"},
		},
		totalCount: 3,
	}
	embedder := &fakeEmbedder{vector: embedding.Vector{1, 2, 3}}
	permissions := &fakePermissionChecker{allowed: map[uuid.UUID]bool{doc1: true, doc3: true}}
	service := NewService(repo, embedder, permissions)

	response, err := service.Search(context.Background(), Query{Text: "hello", UserID: userID, Limit: 10})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(response.Results) != 2 {
		t.Fatalf("expected 2 filtered results, got %d: %#v", len(response.Results), response.Results)
	}
	if response.Results[0].DocumentID != doc1 || response.Results[1].DocumentID != doc3 {
		t.Fatalf("unexpected filtered results: %#v", response.Results)
	}
}

func TestServiceSearchRespectsLimit(t *testing.T) {
	repo := &fakeSearchRepository{
		results: []Result{
			{DocumentID: uuid.New(), Title: "Document 1"},
			{DocumentID: uuid.New(), Title: "Document 2"},
			{DocumentID: uuid.New(), Title: "Document 3"},
		},
		totalCount: 3,
	}
	embedder := &fakeEmbedder{vector: embedding.Vector{1, 2, 3}}
	service := NewService(repo, embedder, nil)

	response, err := service.Search(context.Background(), Query{Text: "hello", Limit: 2})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(response.Results) != 2 {
		t.Fatalf("expected 2 results respecting limit, got %d", len(response.Results))
	}
}

func TestServiceSearchReturnsEmptyForBlankQuery(t *testing.T) {
	repo := &fakeSearchRepository{results: []Result{{DocumentID: uuid.New()}}, totalCount: 1}
	embedder := &fakeEmbedder{vector: embedding.Vector{1, 2, 3}}
	service := NewService(repo, embedder, nil)

	response, err := service.Search(context.Background(), Query{Text: "", Limit: 10})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(response.Results) != 0 {
		t.Fatalf("expected empty results for blank query, got %d", len(response.Results))
	}
}
