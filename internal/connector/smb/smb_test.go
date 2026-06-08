package smb

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/michaelahli/nexd/internal/connector"
)

func TestParseConfig(t *testing.T) {
	cfg, err := ParseConfig(connector.Config{
		Name: "SMB",
		Type: Type,
		Settings: map[string]any{
			"root_path":          "/tmp/share",
			"include_extensions": []any{"txt", ".md"},
		},
	})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.RootPath != filepath.Clean("/tmp/share") {
		t.Fatalf("unexpected root path: %q", cfg.RootPath)
	}
	if len(cfg.IncludeExtensions) != 2 || cfg.IncludeExtensions[0] != ".txt" || cfg.IncludeExtensions[1] != ".md" {
		t.Fatalf("unexpected include extensions: %#v", cfg.IncludeExtensions)
	}
}

func TestExtractText(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	text, err := ExtractText(path)
	if err != nil {
		t.Fatalf("extract text: %v", err)
	}
	if text != "hello" {
		t.Fatalf("unexpected text: %q", text)
	}
}

func TestConnectorFullAndIncrementalSync(t *testing.T) {
	dir := t.TempDir()
	path1 := filepath.Join(dir, "a.txt")
	path2 := filepath.Join(dir, "b.log")
	if err := os.WriteFile(path1, []byte("alpha"), 0o644); err != nil {
		t.Fatalf("write file 1: %v", err)
	}
	if err := os.WriteFile(path2, []byte("beta"), 0o644); err != nil {
		t.Fatalf("write file 2: %v", err)
	}

	raw := connector.Config{
		ID:   uuid.New(),
		Name: "SMB",
		Type: Type,
		Settings: map[string]any{
			"root_path":          dir,
			"include_extensions": []any{".txt", ".log"},
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
		if doc.SourceType != Type {
			t.Fatalf("unexpected source type: %q", doc.SourceType)
		}
	}
	for err := range errs {
		if err != nil {
			t.Fatalf("full sync error: %v", err)
		}
	}
	if count != 2 {
		t.Fatalf("expected 2 documents, got %d", count)
	}

	since := time.Now()
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(path2, []byte("gamma"), 0o644); err != nil {
		t.Fatalf("rewrite file 2: %v", err)
	}
	docs, errs = conn.IncrementalSync(context.Background(), since)
	count = 0
	for range docs {
		count++
	}
	for err := range errs {
		if err != nil {
			t.Fatalf("incremental sync error: %v", err)
		}
	}
	if count != 1 {
		t.Fatalf("expected 1 incremental document, got %d", count)
	}
}
