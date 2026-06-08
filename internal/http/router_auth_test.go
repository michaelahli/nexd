package http_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/michaelahli/nexd/internal/auth"
	apphttp "github.com/michaelahli/nexd/internal/http"
	"github.com/michaelahli/nexd/internal/service/chat"
	"github.com/michaelahli/nexd/internal/service/search"
)

type fakeSearchService struct{}

func (f *fakeSearchService) Search(ctx context.Context, query search.Query) (search.Response, error) {
	return search.Response{Query: query.Text, Results: []search.Result{}, TotalCount: 0, Limit: query.Limit, Offset: query.Offset}, nil
}

type fakeChatService struct{}

func (f *fakeChatService) Chat(ctx context.Context, req chat.Request) (chat.Response, error) {
	return chat.Response{Message: chat.Message{ID: uuid.New(), Role: chat.RoleAssistant, Content: "ok", CreatedAt: time.Now()}}, nil
}

func TestSearchAndChatRequireAuth(t *testing.T) {
	tokens := auth.NewTokenManager("test-secret", time.Hour)
	router := apphttp.NewRouter(testConfig(), apphttp.Options{
		Users:  &auth.UserStore{},
		Tokens: tokens,
		Search: &fakeSearchService{},
		Chat:   &fakeChatService{},
	})

	searchReq := httptest.NewRequest(http.MethodPost, "/search", bytes.NewBufferString(`{"query":"hello"}`))
	searchReq.Header.Set("Content-Type", "application/json")
	searchRec := httptest.NewRecorder()
	router.ServeHTTP(searchRec, searchReq)
	if searchRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected /search unauthorized, got %d", searchRec.Code)
	}

	chatReq := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBufferString(`{"query":"hello"}`))
	chatReq.Header.Set("Content-Type", "application/json")
	chatRec := httptest.NewRecorder()
	router.ServeHTTP(chatRec, chatReq)
	if chatRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected /chat unauthorized, got %d", chatRec.Code)
	}
}
