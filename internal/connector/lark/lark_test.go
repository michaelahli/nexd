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
			"app_id":       "cli_test123",
			"app_secret":   "secret456",
			"folder_token": "fld_test_123",
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
	if cfg.FolderToken != "fld_test_123" {
		t.Fatalf("unexpected folder token: %q", cfg.FolderToken)
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

func TestClientListDocs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open-apis/auth/v3/tenant_access_token/internal":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":                0,
				"msg":                 "success",
				"tenant_access_token": "t-test-token",
				"expire":              7200,
			})
		case "/open-apis/drive/v1/files":
			if got := r.Header.Get("Authorization"); got != "Bearer t-test-token" {
				t.Fatalf("unexpected auth header: %q", got)
			}
			if got := r.URL.Query().Get("folder_token"); got != "fld_test_123" {
				t.Fatalf("unexpected folder token query: %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"msg":  "success",
				"data": map[string]any{
					"files": []map[string]any{{
						"token":         "doc_123",
						"name":          "Doc 123",
						"type":          "docx",
						"created_time":  "1700000000000",
						"modified_time": "1700000100000",
					}},
				},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(Config{AppID: "cli_test", AppSecret: "secret", BaseURL: server.URL, FolderToken: "fld_test_123"})
	docs, err := client.ListDocs(context.Background())
	if err != nil {
		t.Fatalf("list docs: %v", err)
	}
	if len(docs) != 1 || docs[0].Token != "doc_123" || docs[0].Title != "Doc 123" {
		t.Fatalf("unexpected docs: %#v", docs)
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
			"app_id":       "cli_test",
			"app_secret":   "secret",
			"base_url":     server.URL,
			"folder_token": "fld_test_123",
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
		switch r.URL.Path {
		case "/open-apis/auth/v3/tenant_access_token/internal":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":                0,
				"msg":                 "success",
				"tenant_access_token": "t-test-token",
				"expire":              7200,
			})
		case "/open-apis/drive/v1/files":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"msg":  "success",
				"data": map[string]any{
					"files": []map[string]any{{
						"token":         "doc_123",
						"name":          "Doc 123",
						"type":          "docx",
						"created_time":  "1700000000000",
						"modified_time": "1700000100000",
					}},
				},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	raw := connector.Config{
		ID:   uuid.New(),
		Name: "Lark",
		Type: Type,
		Settings: map[string]any{
			"app_id":       "cli_test",
			"app_secret":   "secret",
			"base_url":     server.URL,
			"folder_token": "fld_test_123",
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
	for doc := range docs {
		count++
		if doc.SourceID != "doc_123" || doc.Title != "Doc 123" {
			t.Fatalf("unexpected doc: %#v", doc)
		}
	}
	for err := range errs {
		if err != nil {
			t.Fatalf("sync error: %v", err)
		}
	}
	if count != 1 {
		t.Fatalf("expected 1 document, got %d", count)
	}
}
