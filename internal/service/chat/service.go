package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	DefaultContextLimit = 5
	RoleSystem          = "system"
	RoleUser            = "user"
	RoleAssistant       = "assistant"
)

// Service implements RAG-based chat.
type Service struct {
	llm       LLM
	retriever Retriever
}

// NewService creates a chat service.
func NewService(llm LLM, retriever Retriever) *Service {
	return &Service{llm: llm, retriever: retriever}
}

// Chat retrieves context, constructs a prompt, and generates a response.
func (s *Service) Chat(ctx context.Context, req Request) (Response, error) {
	if s == nil || s.llm == nil || s.retriever == nil {
		return Response{}, fmt.Errorf("chat service is not configured")
	}
	if req.Query == "" {
		return Response{}, fmt.Errorf("chat query is required")
	}

	docs, err := s.retriever.Retrieve(ctx, req.UserID, req.Query, DefaultContextLimit)
	if err != nil {
		return Response{}, fmt.Errorf("retrieve context: %w", err)
	}

	systemPrompt := buildSystemPrompt(docs)
	messages := append([]Message(nil), req.History...)
	messages = append(messages, Message{Role: RoleUser, Content: req.Query})

	content, err := s.llm.Complete(ctx, messages, systemPrompt)
	if err != nil {
		return Response{}, fmt.Errorf("generate chat completion: %w", err)
	}

	citations := extractCitations(docs)
	reply := Message{
		ID:        uuid.New(),
		Role:      RoleAssistant,
		Content:   content,
		Citations: citations,
		CreatedAt: time.Now(),
	}

	return Response{Message: reply, Citations: citations}, nil
}

func buildSystemPrompt(docs []RetrievedDocument) string {
	if len(docs) == 0 {
		return "You are a helpful assistant. Answer the user's question based on your knowledge."
	}

	var builder strings.Builder
	builder.WriteString("You are a helpful assistant. Use the following context to answer the user's question. ")
	builder.WriteString("If the context does not contain relevant information, say so.\n\n")
	builder.WriteString("Context:\n")
	for i, doc := range docs {
		builder.WriteString(fmt.Sprintf("\n[%d] %s\n%s\n", i+1, doc.Title, doc.Content))
	}
	return builder.String()
}

func extractCitations(docs []RetrievedDocument) []Citation {
	citations := make([]Citation, 0, len(docs))
	for _, doc := range docs {
		citations = append(citations, Citation{
			DocumentID: doc.DocumentID,
			SourceType: doc.SourceType,
			SourceID:   doc.SourceID,
			Title:      doc.Title,
			Snippet:    truncate(doc.Content, 200),
		})
	}
	return citations
}

func truncate(text string, maxRunes int) string {
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	return string(runes[:maxRunes]) + "..."
}
