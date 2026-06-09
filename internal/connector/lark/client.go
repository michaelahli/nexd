package lark

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client wraps Lark Open API calls.
type Client struct {
	config     Config
	httpClient *http.Client
	token      string
	tokenExp   time.Time
}

// NewClient creates a Lark API client.
func NewClient(config Config) *Client {
	return &Client{
		config:     config,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Auth obtains a tenant_access_token from Lark.
func (c *Client) Auth(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("lark client is not configured")
	}
	if time.Now().Before(c.tokenExp) {
		return nil
	}

	body, err := json.Marshal(map[string]string{
		"app_id":     c.config.AppID,
		"app_secret": c.config.AppSecret,
	})
	if err != nil {
		return fmt.Errorf("marshal auth request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.BaseURL+"/open-apis/auth/v3/tenant_access_token/internal", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send auth request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("auth request failed with status %d: %s", resp.StatusCode, string(message))
	}

	var result struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int    `json:"expire"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode auth response: %w", err)
	}
	if result.Code != 0 {
		return fmt.Errorf("lark auth error %d: %s", result.Code, result.Msg)
	}

	c.token = result.TenantAccessToken
	c.tokenExp = time.Now().Add(time.Duration(result.Expire-60) * time.Second)
	return nil
}

// ListDocs lists Lark Docs accessible by the application (stub for now).
func (c *Client) ListDocs(ctx context.Context) ([]LarkDoc, error) {
	if err := c.Auth(ctx); err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}
	// Real implementation would call /open-apis/drive/v1/files or similar.
	// For now, return empty list to satisfy the connector contract.
	return []LarkDoc{}, nil
}

// LarkDoc represents a Lark document reference.
type LarkDoc struct {
	Token     string
	Title     string
	Type      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
