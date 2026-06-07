package embedding

import (
	"strings"
	"testing"
)

func TestChunkerUsesParagraphBoundaries(t *testing.T) {
	chunker := NewChunker(ChunkConfig{MaxRunes: 24})

	chunks := chunker.Chunk("alpha beta\n\ngamma delta\n\nepsilon")
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d: %#v", len(chunks), chunks)
	}
	if chunks[0].Text != "alpha beta\n\ngamma delta" {
		t.Fatalf("unexpected first chunk: %q", chunks[0].Text)
	}
	if chunks[1].Text != "epsilon" {
		t.Fatalf("unexpected second chunk: %q", chunks[1].Text)
	}
}

func TestChunkerWindowsLongParagraphWithOverlap(t *testing.T) {
	chunker := NewChunker(ChunkConfig{MaxRunes: 5, OverlapRunes: 2})

	chunks := chunker.Chunk("abcdefghij")
	got := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		got = append(got, chunk.Text)
	}

	want := []string{"abcde", "defgh", "ghij"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("expected %v, got %v", want, got)
	}
}
