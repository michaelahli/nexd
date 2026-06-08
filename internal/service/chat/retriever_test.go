package chat

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/michaelahli/nexd/internal/service/search"
)

type fakeSearchService struct {
	results []search.Result
}

func (s *fakeSearchService) Search(ctx context.Context, query search.Query) (search.Response, error) {
	return search.Response{Results: s.results, TotalCount: len(s.results), Query: query.Text, Limit: query.Limit}, nil
}

func TestSearchRetrieverConvertsResults(t *testing.T) {
	docID := uuid.New()
	fake := &fakeSearchService{results: []search.Result{
		{DocumentID: docID, SourceType: "lark", SourceID: "doc-1", Title: "Doc 1", Snippet: "snippet text", Score: 0.92},
	}}
	retriever := NewSearchRetriever(fake)

	docs, err := retriever.Retrieve(context.Background(), uuid.New(), "hello", 5)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}
	doc := docs[0]
	if doc.DocumentID != docID || doc.SourceType != "lark" || doc.Title != "Doc 1" || doc.Content != "snippet text" || doc.Score != 0.92 {
		t.Fatalf("unexpected doc: %#v", doc)
	}
}
