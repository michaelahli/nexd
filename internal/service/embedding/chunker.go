package embedding

import "strings"

const (
	DefaultMaxChunkRunes = 1200
	DefaultOverlapRunes  = 120
)

// ChunkConfig controls text splitting.
type ChunkConfig struct {
	MaxRunes     int
	OverlapRunes int
}

// Chunk is a piece of text ready for embedding.
type Chunk struct {
	Index int
	Text  string
}

// Chunker splits text into embedding-sized chunks.
type Chunker struct {
	config ChunkConfig
}

// NewChunker creates a chunker with sensible defaults.
func NewChunker(cfg ChunkConfig) *Chunker {
	if cfg.MaxRunes <= 0 {
		cfg.MaxRunes = DefaultMaxChunkRunes
	}
	if cfg.OverlapRunes < 0 {
		cfg.OverlapRunes = 0
	}
	if cfg.OverlapRunes >= cfg.MaxRunes {
		cfg.OverlapRunes = cfg.MaxRunes / 5
	}
	return &Chunker{config: cfg}
}

// Chunk splits text by paragraphs first, then falls back to rune windows for long paragraphs.
func (c *Chunker) Chunk(text string) []Chunk {
	if c == nil {
		c = NewChunker(ChunkConfig{})
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	paragraphs := splitParagraphs(text)
	chunks := make([]Chunk, 0, len(paragraphs))
	var current strings.Builder

	flush := func() {
		chunkText := strings.TrimSpace(current.String())
		if chunkText == "" {
			current.Reset()
			return
		}
		chunks = append(chunks, Chunk{Index: len(chunks), Text: chunkText})
		current.Reset()
	}

	for _, paragraph := range paragraphs {
		if runeLen(paragraph) > c.config.MaxRunes {
			flush()
			for _, window := range c.window(paragraph) {
				chunks = append(chunks, Chunk{Index: len(chunks), Text: window})
			}
			continue
		}

		candidateLen := runeLen(paragraph)
		if current.Len() > 0 {
			candidateLen += runeLen(current.String()) + 2
		}
		if candidateLen > c.config.MaxRunes {
			flush()
		}
		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		current.WriteString(paragraph)
	}
	flush()

	return chunks
}

func (c *Chunker) window(text string) []string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) == 0 {
		return nil
	}

	step := c.config.MaxRunes - c.config.OverlapRunes
	if step <= 0 {
		step = c.config.MaxRunes
	}

	windows := make([]string, 0, len(runes)/step+1)
	for start := 0; start < len(runes); start += step {
		end := start + c.config.MaxRunes
		if end > len(runes) {
			end = len(runes)
		}
		windows = append(windows, strings.TrimSpace(string(runes[start:end])))
		if end == len(runes) {
			break
		}
	}
	return windows
}

func splitParagraphs(text string) []string {
	parts := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n\n")
	paragraphs := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			paragraphs = append(paragraphs, part)
		}
	}
	return paragraphs
}

func runeLen(s string) int {
	return len([]rune(s))
}
