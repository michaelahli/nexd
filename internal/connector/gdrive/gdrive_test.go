package gdrive

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/michaelahli/nexd/internal/connector"
)

const testServiceAccountJSON = `{
  "type": "service_account",
  "project_id": "test-project",
  "private_key": "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----\n",
  "client_email": "test@test-project.iam.gserviceaccount.com",
  "token_uri": "https://oauth2.googleapis.com/token"
}`

func TestParseConfig(t *testing.T) {
	cfg, err := ParseConfig(connector.Config{
		Name: "Google Drive",
		Type: Type,
		Settings: map[string]any{
			"service_account_json": testServiceAccountJSON,
			"drive_folder_id":      "1234567890",
			"access_token":         "ya29.test-token",
		},
	})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.ServiceAccountJSON == "" {
		t.Fatal("expected service account JSON")
	}
	if cfg.DriveFolderID != "1234567890" {
		t.Fatalf("unexpected drive folder ID: %q", cfg.DriveFolderID)
	}
	if cfg.AccessToken != "ya29.test-token" {
		t.Fatalf("unexpected access token: %q", cfg.AccessToken)
	}
}

func TestNewClient(t *testing.T) {
	client, err := NewClient(Config{ServiceAccountJSON: testServiceAccountJSON})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestClientAuthWithAccessToken(t *testing.T) {
	client, err := NewClient(Config{ServiceAccountJSON: testServiceAccountJSON, AccessToken: "ya29.test-token"})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if err := client.Auth(context.Background()); err != nil {
		t.Fatalf("auth: %v", err)
	}
	if client.token != "ya29.test-token" {
		t.Fatalf("unexpected token: %q", client.token)
	}
}

func TestConnectorValidateAndHealth(t *testing.T) {
	raw := connector.Config{
		ID:   uuid.New(),
		Name: "Google Drive",
		Type: Type,
		Settings: map[string]any{
			"service_account_json": testServiceAccountJSON,
			"access_token":         "ya29.test-token",
		},
	}
	connAny, err := New(raw)
	if err != nil {
		t.Fatalf("new connector: %v", err)
	}
	conn := connAny.(*Connector)
	if err := conn.Start(context.Background(), raw); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := conn.Health(context.Background()); err != nil {
		t.Fatalf("health: %v", err)
	}
}

func TestConnectorFullSync(t *testing.T) {
	raw := connector.Config{
		ID:   uuid.New(),
		Name: "Google Drive",
		Type: Type,
		Settings: map[string]any{
			"service_account_json": testServiceAccountJSON,
			"access_token":         "ya29.test-token",
		},
	}
	connAny, err := New(raw)
	if err != nil {
		t.Fatalf("new connector: %v", err)
	}
	conn := connAny.(*Connector)
	if err := conn.Start(context.Background(), raw); err != nil {
		t.Fatalf("start: %v", err)
	}

	docs, errs := conn.FullSync(context.Background())
	count := 0
	for range docs {
		count++
	}
	for err := range errs {
		if err == nil {
			continue
		}
		// Without a live Google API in tests, the connector may fail when listing.
		// That is acceptable here because auth path and lifecycle are what we verify.
		return
	}
	if count != 0 {
		t.Fatalf("expected 0 documents without a mocked Google API, got %d", count)
	}
}
