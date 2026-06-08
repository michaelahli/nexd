package chat

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Message is a single chat message.
type Message struct {
	ID        uuid.UUID
	Role      string
	Content   string
	Citations []Citation
	CreatedAt time.Time
}

// Citation links a message to source documents.
type Citation struct {
	DocumentID uuid.UUID
	SourceType string
	SourceID   string
	Title      string
	Snippet    string
}

// Request describes a chat request.
type Request struct {
	UserID  uuid.UUID
	Query   string
	History []Message
}

// Response bundles a chat reply and citations.
type Response struct {
	Message   Message
	Citations []Citation
}

// LLM generates chat completions.
type LLM interface {
	Complete(ctx context.Context, messages []Message, systemPrompt string) (string, error)
}

// Retriever fetches relevant context for a query.
type Retriever interface {
	Retrieve(ctx context.Context, userID uuid.UUID, query string, limit int) ([]RetrievedDocument, error)
}

// RetrievedDocument is a search result used as RAG context.
type RetrievedDocument struct {
	DocumentID uuid.UUID
	SourceType string
	SourceID   string
	Title      string
	Content    string
	Score      float64
}
