package embedding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIClientEmbedsInputs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/embeddings" {
			t.Fatalf("expected /embeddings, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("unexpected authorization header: %q", got)
		}

		var req openAIEmbeddingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "model-a" {
			t.Fatalf("unexpected model: %q", req.Model)
		}
		if len(req.Input) != 2 || req.Input[0] != "first" || req.Input[1] != "second" {
			t.Fatalf("unexpected input: %#v", req.Input)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"index":1,"embedding":[2,2]},{"index":0,"embedding":[1,1]}]}`))
	}))
	defer server.Close()

	client, err := NewOpenAIClient(OpenAIConfig{Host: server.URL, APIKey: "secret", Model: "model-a"})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	vectors, err := client.Embed(context.Background(), []string{"first", "second"})
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if len(vectors) != 2 || vectors[0][0] != 1 || vectors[1][0] != 2 {
		t.Fatalf("unexpected vectors: %#v", vectors)
	}
}

func TestOpenAIClientReturnsStatusErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusTooManyRequests)
	}))
	defer server.Close()

	client, err := NewOpenAIClient(OpenAIConfig{Host: server.URL})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.Embed(context.Background(), []string{"hello"})
	if err == nil {
		t.Fatal("expected status error")
	}
}
