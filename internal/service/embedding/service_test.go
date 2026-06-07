package embedding

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type fakeProvider struct {
	calls  [][]string
	fails  int
	vector Vector
}

func (p *fakeProvider) Embed(ctx context.Context, texts []string) ([]Vector, error) {
	p.calls = append(p.calls, append([]string(nil), texts...))
	if p.fails > 0 {
		p.fails--
		return nil, errors.New("temporary failure")
	}
	vectors := make([]Vector, len(texts))
	for i := range texts {
		if p.vector != nil {
			vectors[i] = append(Vector(nil), p.vector...)
			continue
		}
		vectors[i] = Vector{float32(len(texts[i]))}
	}
	return vectors, nil
}

func TestServiceEmbedsInBatches(t *testing.T) {
	provider := &fakeProvider{}
	service := NewService(provider, Config{BatchSize: 2})

	vectors, err := service.Embed(context.Background(), []string{"a", "bb", "ccc"})
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if len(vectors) != 3 {
		t.Fatalf("expected 3 vectors, got %d", len(vectors))
	}
	if len(provider.calls) != 2 {
		t.Fatalf("expected 2 provider calls, got %d", len(provider.calls))
	}
	if !reflect.DeepEqual(provider.calls[0], []string{"a", "bb"}) || !reflect.DeepEqual(provider.calls[1], []string{"ccc"}) {
		t.Fatalf("unexpected batches: %#v", provider.calls)
	}
}

func TestServiceRetriesFailures(t *testing.T) {
	provider := &fakeProvider{fails: 1}
	service := NewService(provider, Config{BatchSize: 2, MaxRetries: 1})
	service.sleep = func(ctx context.Context, d time.Duration) error { return nil }

	if _, err := service.Embed(context.Background(), []string{"a"}); err != nil {
		t.Fatalf("embed with retry: %v", err)
	}
	if len(provider.calls) != 2 {
		t.Fatalf("expected retry call, got %d calls", len(provider.calls))
	}
}

func TestServiceEmbedTextUsesCache(t *testing.T) {
	provider := &fakeProvider{vector: Vector{1, 2}}
	service := NewService(provider, Config{
		BatchSize: 1,
		CacheTTL:  time.Minute,
		ChunkConfig: ChunkConfig{
			MaxRunes: 100,
		},
	})

	first, err := service.EmbedText(context.Background(), "hello")
	if err != nil {
		t.Fatalf("first embed: %v", err)
	}
	second, err := service.EmbedText(context.Background(), "hello")
	if err != nil {
		t.Fatalf("second embed: %v", err)
	}
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("expected one chunk each, got %d and %d", len(first), len(second))
	}
	if len(provider.calls) != 1 {
		t.Fatalf("expected cached second call, got %d provider calls", len(provider.calls))
	}
}
