package lark

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/michaelahli/nexd/internal/connector"
)

func TestParseConfig(t *testing.T) {
	cfg, err := ParseConfig(connector.Config{
		Name: "Lark",
		Type: Type,
		Settings: map[string]any{
			"app_id":     "cli_test123",
			"app_secret": "secret456",
		},
	})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.AppID != "cli_test123" || cfg.AppSecret != "secret456" {
		t.Fatalf("unexpected config: %#v", cfg)
	}
	if cfg.BaseURL != "https://open.larksuite.com" {
		t.Fatalf("unexpected base URL: %q", cfg.BaseURL)
	}
}

func TestClientAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/open-apis/auth/v3/tenant_access_token/internal" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":                0,
			"msg":                 "success",
			"tenant_access_token": "t-test-token",
			"expire":              7200,
		})
	}))
	defer server.Close()

	client := NewClient(Config{AppID: "cli_test", AppSecret: "secret", BaseURL: server.URL})
	if err := client.Auth(context.Background()); err != nil {
		t.Fatalf("auth: %v", err)
	}
	if client.token != "t-test-token" {
		t.Fatalf("unexpected token: %q", client.token)
	}
}

func TestConnectorValidateAndHealth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":                0,
			"msg":                 "success",
			"tenant_access_token": "t-test-token",
			"expire":              7200,
		})
	}))
	defer server.Close()

	raw := connector.Config{
		ID:   uuid.New(),
		Name: "Lark",
		Type: Type,
		Settings: map[string]any{
			"app_id":     "cli_test",
			"app_secret": "secret",
			"base_url":   server.URL,
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":                0,
			"msg":                 "success",
			"tenant_access_token": "t-test-token",
			"expire":              7200,
		})
	}))
	defer server.Close()

	raw := connector.Config{
		ID:   uuid.New(),
		Name: "Lark",
		Type: Type,
		Settings: map[string]any{
			"app_id":     "cli_test",
			"app_secret": "secret",
			"base_url":   server.URL,
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
		if err != nil {
			t.Fatalf("sync error: %v", err)
		}
	}
	// Currently returns empty list because ListDocs is stubbed.
	if count != 0 {
		t.Fatalf("expected 0 documents from stub, got %d", count)
	}
}
