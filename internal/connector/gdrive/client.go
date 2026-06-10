package gdrive

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client wraps Google Drive API v3 calls.
type Client struct {
	config     Config
	httpClient *http.Client
	token      string
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
	return &Client{config: config, httpClient: &http.Client{Timeout: 30 * time.Second}}, nil
}

// Auth obtains an OAuth2 access token for the service account (stub for now).
func (c *Client) Auth(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("gdrive client is not configured")
	}
	if c.config.AccessToken != "" {
		c.token = c.config.AccessToken
		return nil
	}
	// Real service-account JWT signing can be added later. For now, validate credentials exist.
	var sa serviceAccount
	if err := json.Unmarshal([]byte(c.config.ServiceAccountJSON), &sa); err != nil {
		return fmt.Errorf("invalid service account JSON: %w", err)
	}
	return fmt.Errorf("gdrive connector requires access_token for live API calls; service-account JWT exchange is not implemented yet")
}

// ListFiles lists Google Drive files.
func (c *Client) ListFiles(ctx context.Context) ([]DriveFile, error) {
	if err := c.Auth(ctx); err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}

	query := url.Values{}
	query.Set("pageSize", "50")
	query.Set("fields", "files(id,name,mimeType,createdTime,modifiedTime)")
	if c.config.DriveFolderID != "" {
		query.Set("q", fmt.Sprintf("'%s' in parents", c.config.DriveFolderID))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/drive/v3/files?"+query.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("create list files request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send list files request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("list files request failed with status %d: %s", resp.StatusCode, string(message))
	}

	var result struct {
		Files []struct {
			ID           string `json:"id"`
			Name         string `json:"name"`
			MimeType     string `json:"mimeType"`
			CreatedTime  string `json:"createdTime"`
			ModifiedTime string `json:"modifiedTime"`
		} `json:"files"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode list files response: %w", err)
	}

	files := make([]DriveFile, 0, len(result.Files))
	for _, file := range result.Files {
		createdAt, _ := time.Parse(time.RFC3339, file.CreatedTime)
		modifiedAt, _ := time.Parse(time.RFC3339, file.ModifiedTime)
		files = append(files, DriveFile{
			ID:           file.ID,
			Name:         file.Name,
			MimeType:     file.MimeType,
			CreatedTime:  createdAt,
			ModifiedTime: modifiedAt,
		})
	}
	return files, nil
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
