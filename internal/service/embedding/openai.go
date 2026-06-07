package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

const DefaultOpenAIEmbeddingModel = "text-embedding-ada-002"

// OpenAIClient calls an OpenAI-compatible embeddings endpoint.
type OpenAIClient struct {
	host       string
	apiKey     string
	model      string
	httpClient *http.Client
}

// OpenAIConfig configures the OpenAI-compatible embedding provider.
type OpenAIConfig struct {
	Host       string
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

// NewOpenAIClient creates an OpenAI-compatible embedding client.
func NewOpenAIClient(cfg OpenAIConfig) (*OpenAIClient, error) {
	cfg.Host = strings.TrimRight(strings.TrimSpace(cfg.Host), "/")
	if cfg.Host == "" {
		return nil, errors.New("openai host is required")
	}
	if cfg.Model == "" {
		cfg.Model = DefaultOpenAIEmbeddingModel
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}

	return &OpenAIClient{host: cfg.Host, apiKey: cfg.APIKey, model: cfg.Model, httpClient: cfg.HTTPClient}, nil
}

// Embed generates embeddings using POST /embeddings.
func (c *OpenAIClient) Embed(ctx context.Context, texts []string) ([]Vector, error) {
	if c == nil || c.httpClient == nil {
		return nil, errors.New("openai client is not configured")
	}
	if len(texts) == 0 {
		return nil, nil
	}

	body, err := json.Marshal(openAIEmbeddingRequest{Model: c.model, Input: texts})
	if err != nil {
		return nil, fmt.Errorf("marshal embedding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send embedding request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("embedding request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(message)))
	}

	var decoded openAIEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode embedding response: %w", err)
	}
	if len(decoded.Data) != len(texts) {
		return nil, fmt.Errorf("embedding response contained %d vectors for %d inputs", len(decoded.Data), len(texts))
	}

	sort.Slice(decoded.Data, func(i, j int) bool { return decoded.Data[i].Index < decoded.Data[j].Index })
	vectors := make([]Vector, len(decoded.Data))
	for i, item := range decoded.Data {
		if item.Index != i {
			return nil, fmt.Errorf("embedding response missing index %d", i)
		}
		vectors[i] = item.Embedding
	}

	return vectors, nil
}

type openAIEmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type openAIEmbeddingResponse struct {
	Data []openAIEmbeddingData `json:"data"`
}

type openAIEmbeddingData struct {
	Index     int    `json:"index"`
	Embedding Vector `json:"embedding"`
}
