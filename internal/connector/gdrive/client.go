package gdrive

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Client wraps Google Drive API v3 calls.
type Client struct {
	config Config
	token  string
}

// NewClient creates a Google Drive API client.
func NewClient(config Config) (*Client, error) {
	// Validate service account JSON format
	var sa serviceAccount
	if err := json.Unmarshal([]byte(config.ServiceAccountJSON), &sa); err != nil {
		return nil, fmt.Errorf("invalid service account JSON: %w", err)
	}
	if sa.Type != "service_account" {
		return nil, fmt.Errorf("service account type must be 'service_account', got %q", sa.Type)
	}
	if sa.PrivateKey == "" || sa.ClientEmail == "" {
		return nil, fmt.Errorf("service account missing private_key or client_email")
	}
	return &Client{config: config}, nil
}

// Auth obtains an OAuth2 access token for the service account (stub for now).
func (c *Client) Auth(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("gdrive client is not configured")
	}
	// Real implementation would use JWT signing and call:
	// POST https://oauth2.googleapis.com/token
	// For now, just validate we have credentials.
	var sa serviceAccount
	if err := json.Unmarshal([]byte(c.config.ServiceAccountJSON), &sa); err != nil {
		return fmt.Errorf("invalid service account JSON: %w", err)
	}
	c.token = "stub-token"
	return nil
}

// ListFiles lists Google Drive files (stub for now).
func (c *Client) ListFiles(ctx context.Context) ([]DriveFile, error) {
	if err := c.Auth(ctx); err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}
	// Real implementation would call:
	// GET https://www.googleapis.com/drive/v3/files
	// For now, return empty list.
	return []DriveFile{}, nil
}

// DriveFile represents a Google Drive file reference.
type DriveFile struct {
	ID           string
	Name         string
	MimeType     string
	CreatedTime  time.Time
	ModifiedTime time.Time
}

type serviceAccount struct {
	Type        string `json:"type"`
	ProjectID   string `json:"project_id"`
	PrivateKey  string `json:"private_key"`
	ClientEmail string `json:"client_email"`
	TokenURI    string `json:"token_uri"`
}
