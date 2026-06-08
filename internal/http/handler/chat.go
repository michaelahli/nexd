package handler

import (
	"context"
	"net/http"

	"github.com/michaelahli/nexd/internal/http/middleware"
	"github.com/michaelahli/nexd/internal/service/chat"
)

type chatService interface {
	Chat(ctx context.Context, req chat.Request) (chat.Response, error)
}

// Chat handles public chat endpoints.
type Chat struct {
	service chatService
}

// NewChat creates a chat handler.
func NewChat(service chatService) *Chat {
	return &Chat{service: service}
}

type chatRequest struct {
	Query   string         `json:"query"`
	History []chat.Message `json:"history"`
}

// Complete runs a RAG chat request for the authenticated user.
func (h *Chat) Complete(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req chatRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}

	response, err := h.service.Chat(r.Context(), chat.Request{
		UserID:  claims.UserID,
		Query:   req.Query,
		History: req.History,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, response)
}
