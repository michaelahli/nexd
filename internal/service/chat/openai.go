package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const DefaultChatModel = "gpt-4"

// OpenAIClient calls an OpenAI-compatible chat completions endpoint.
type OpenAIClient struct {
	host       string
	apiKey     string
	model      string
	httpClient *http.Client
}

// OpenAIConfig configures the OpenAI-compatible chat client.
type OpenAIConfig struct {
	Host       string
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

// NewOpenAIClient creates an OpenAI-compatible chat client.
func NewOpenAIClient(cfg OpenAIConfig) (*OpenAIClient, error) {
	cfg.Host = strings.TrimRight(strings.TrimSpace(cfg.Host), "/")
	if cfg.Host == "" {
		return nil, errors.New("openai host is required")
	}
	if cfg.Model == "" {
		cfg.Model = DefaultChatModel
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 60 * time.Second}
	}

	return &OpenAIClient{host: cfg.Host, apiKey: cfg.APIKey, model: cfg.Model, httpClient: cfg.HTTPClient}, nil
}

// Complete generates a chat completion.
func (c *OpenAIClient) Complete(ctx context.Context, messages []Message, systemPrompt string) (string, error) {
	if c == nil || c.httpClient == nil {
		return "", errors.New("openai client is not configured")
	}

	apiMessages := make([]openAIMessage, 0, len(messages)+1)
	if systemPrompt != "" {
		apiMessages = append(apiMessages, openAIMessage{Role: RoleSystem, Content: systemPrompt})
	}
	for _, msg := range messages {
		apiMessages = append(apiMessages, openAIMessage{Role: msg.Role, Content: msg.Content})
	}

	body, err := json.Marshal(openAIChatRequest{Model: c.model, Messages: apiMessages})
	if err != nil {
		return "", fmt.Errorf("marshal chat request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create chat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send chat request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("chat request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(message)))
	}

	var decoded openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", fmt.Errorf("decode chat response: %w", err)
	}
	if len(decoded.Choices) == 0 {
		return "", fmt.Errorf("chat response contained no choices")
	}

	return decoded.Choices[0].Message.Content, nil
}

type openAIChatRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	Choices []openAIChoice `json:"choices"`
}

type openAIChoice struct {
	Message openAIMessage `json:"message"`
}
