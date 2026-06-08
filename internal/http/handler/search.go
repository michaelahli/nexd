package handler

import (
	"context"
	"net/http"

	"github.com/michaelahli/nexd/internal/http/middleware"
	"github.com/michaelahli/nexd/internal/service/search"
)

type searchService interface {
	Search(ctx context.Context, query search.Query) (search.Response, error)
}

// Search handles public search endpoints.
type Search struct {
	service searchService
}

// NewSearch creates a search handler.
func NewSearch(service searchService) *Search {
	return &Search{service: service}
}

type searchRequest struct {
	Query  string `json:"query"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

// Query executes a permission-aware search for the authenticated user.
func (h *Search) Query(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req searchRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}

	response, err := h.service.Search(r.Context(), search.Query{
		Text:   req.Query,
		UserID: claims.UserID,
		Limit:  req.Limit,
		Offset: req.Offset,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, response)
}
