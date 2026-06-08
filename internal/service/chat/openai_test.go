package chat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIClientComplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("unexpected auth header: %q", got)
		}

		var req openAIChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "gpt-4o" {
			t.Errorf("unexpected model: %q", req.Model)
		}
		// first message should be system
		if len(req.Messages) == 0 || req.Messages[0].Role != RoleSystem {
			t.Errorf("expected system message first, got: %#v", req.Messages)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(openAIChatResponse{
			Choices: []openAIChoice{{Message: openAIMessage{Role: RoleAssistant, Content: "Hello!"}}},
		})
	}))
	defer server.Close()

	client, err := NewOpenAIClient(OpenAIConfig{Host: server.URL, APIKey: "test-key", Model: "gpt-4o"})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	content, err := client.Complete(context.Background(), []Message{{Role: RoleUser, Content: "Hi"}}, "Be helpful.")
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if content != "Hello!" {
		t.Fatalf("unexpected content: %q", content)
	}
}

func TestOpenAIClientStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer server.Close()

	client, err := NewOpenAIClient(OpenAIConfig{Host: server.URL})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.Complete(context.Background(), []Message{{Role: RoleUser, Content: "Hi"}}, "")
	if err == nil {
		t.Fatal("expected status error")
	}
}
