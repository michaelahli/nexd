package chat

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type fakeLLM struct {
	response string
	prompt   string
}

func (l *fakeLLM) Complete(ctx context.Context, messages []Message, systemPrompt string) (string, error) {
	l.prompt = systemPrompt
	return l.response, nil
}

type fakeRetriever struct {
	docs []RetrievedDocument
}

func (r *fakeRetriever) Retrieve(ctx context.Context, userID uuid.UUID, query string, limit int) ([]RetrievedDocument, error) {
	return r.docs, nil
}

func TestServiceBuildsSystemPromptWithContext(t *testing.T) {
	llm := &fakeLLM{response: "The answer is 42."}
	retriever := &fakeRetriever{docs: []RetrievedDocument{
		{DocumentID: uuid.New(), Title: "Doc 1", Content: "Content 1"},
		{DocumentID: uuid.New(), Title: "Doc 2", Content: "Content 2"},
	}}
	service := NewService(llm, retriever)

	response, err := service.Chat(context.Background(), Request{UserID: uuid.New(), Query: "What is the answer?"})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if response.Message.Content != "The answer is 42." {
		t.Fatalf("unexpected message content: %q", response.Message.Content)
	}
	if !strings.Contains(llm.prompt, "Doc 1") || !strings.Contains(llm.prompt, "Content 1") {
		t.Fatalf("system prompt missing context: %q", llm.prompt)
	}
	if len(response.Citations) != 2 {
		t.Fatalf("expected 2 citations, got %d", len(response.Citations))
	}
}

func TestServiceHandlesEmptyContext(t *testing.T) {
	llm := &fakeLLM{response: "I don't have context."}
	retriever := &fakeRetriever{docs: []RetrievedDocument{}}
	service := NewService(llm, retriever)

	response, err := service.Chat(context.Background(), Request{UserID: uuid.New(), Query: "What is the answer?"})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if response.Message.Content != "I don't have context." {
		t.Fatalf("unexpected message content: %q", response.Message.Content)
	}
	if strings.Contains(llm.prompt, "Context:") {
		t.Fatalf("system prompt should not contain context section: %q", llm.prompt)
	}
	if len(response.Citations) != 0 {
		t.Fatalf("expected no citations, got %d", len(response.Citations))
	}
}

func TestServiceExtractsCitations(t *testing.T) {
	docID := uuid.New()
	llm := &fakeLLM{response: "Answer."}
	retriever := &fakeRetriever{docs: []RetrievedDocument{
		{DocumentID: docID, SourceType: "smb", SourceID: "file-1", Title: "File 1", Content: "Content"},
	}}
	service := NewService(llm, retriever)

	response, err := service.Chat(context.Background(), Request{UserID: uuid.New(), Query: "query"})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if len(response.Citations) != 1 {
		t.Fatalf("expected one citation, got %d", len(response.Citations))
	}
	citation := response.Citations[0]
	if citation.DocumentID != docID || citation.SourceType != "smb" || citation.Title != "File 1" {
		t.Fatalf("unexpected citation: %#v", citation)
	}
}
