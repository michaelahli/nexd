package embedding

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"
)

const DefaultBatchSize = 10

// Vector is a single embedding vector.
type Vector []float32

// Provider generates embeddings for text inputs.
type Provider interface {
	Embed(ctx context.Context, texts []string) ([]Vector, error)
}

// Service chunks text and generates embeddings through a provider.
type Service struct {
	provider Provider
	chunker  *Chunker
	cache    *Cache
	config   Config
	sleep    func(context.Context, time.Duration) error
}

// Config controls embedding generation behavior.
type Config struct {
	BatchSize   int
	MaxRetries  int
	RetryDelay  time.Duration
	CacheTTL    time.Duration
	ChunkConfig ChunkConfig
}

// ChunkEmbedding pairs chunk text with its vector.
type ChunkEmbedding struct {
	Index  int
	Text   string
	Vector Vector
}

// NewService creates an embedding service.
func NewService(provider Provider, cfg Config) *Service {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = DefaultBatchSize
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	if cfg.RetryDelay <= 0 {
		cfg.RetryDelay = 200 * time.Millisecond
	}

	return &Service{
		provider: provider,
		chunker:  NewChunker(cfg.ChunkConfig),
		cache:    NewCache(cfg.CacheTTL),
		config:   cfg,
		sleep:    sleepContext,
	}
}

// EmbedText splits text into chunks and embeds each chunk.
func (s *Service) EmbedText(ctx context.Context, text string) ([]ChunkEmbedding, error) {
	if s == nil || s.provider == nil {
		return nil, errors.New("embedding service is not configured")
	}

	chunks := s.chunker.Chunk(text)
	if len(chunks) == 0 {
		return nil, nil
	}

	vectors := make([]Vector, len(chunks))
	uncachedInputs := make([]string, 0, len(chunks))
	uncachedIndexes := make([]int, 0, len(chunks))
	for i, chunk := range chunks {
		if vector, ok := s.cache.Get(chunk.Text); ok {
			vectors[i] = vector
			continue
		}
		uncachedInputs = append(uncachedInputs, chunk.Text)
		uncachedIndexes = append(uncachedIndexes, i)
	}

	if len(uncachedInputs) > 0 {
		generated, err := s.Embed(ctx, uncachedInputs)
		if err != nil {
			return nil, err
		}
		for i, vector := range generated {
			chunkIndex := uncachedIndexes[i]
			vectors[chunkIndex] = vector
			s.cache.Set(chunks[chunkIndex].Text, vector)
		}
	}

	result := make([]ChunkEmbedding, len(chunks))
	for i, chunk := range chunks {
		result[i] = ChunkEmbedding{Index: chunk.Index, Text: chunk.Text, Vector: vectors[i]}
	}
	return result, nil
}

// Embed generates embeddings for inputs in batches.
func (s *Service) Embed(ctx context.Context, texts []string) ([]Vector, error) {
	if s == nil || s.provider == nil {
		return nil, errors.New("embedding service is not configured")
	}
	if len(texts) == 0 {
		return nil, nil
	}

	vectors := make([]Vector, 0, len(texts))
	for start := 0; start < len(texts); start += s.config.BatchSize {
		end := int(math.Min(float64(start+s.config.BatchSize), float64(len(texts))))
		batchVectors, err := s.embedWithRetry(ctx, texts[start:end])
		if err != nil {
			return nil, err
		}
		if len(batchVectors) != end-start {
			return nil, fmt.Errorf("embedding provider returned %d vectors for %d inputs", len(batchVectors), end-start)
		}
		vectors = append(vectors, batchVectors...)
	}

	return vectors, nil
}

func (s *Service) embedWithRetry(ctx context.Context, texts []string) ([]Vector, error) {
	var lastErr error
	attempts := s.config.MaxRetries + 1
	for attempt := 0; attempt < attempts; attempt++ {
		vectors, err := s.provider.Embed(ctx, texts)
		if err == nil {
			return vectors, nil
		}
		lastErr = err
		if attempt == attempts-1 {
			break
		}
		if err := s.sleep(ctx, s.config.RetryDelay*time.Duration(attempt+1)); err != nil {
			return nil, err
		}
	}
	return nil, fmt.Errorf("generate embeddings: %w", lastErr)
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
