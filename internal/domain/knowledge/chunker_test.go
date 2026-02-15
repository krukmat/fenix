// Task 2.2: Unit tests for the text chunker.
// Tests are written BEFORE the implementation (TDD) — they will fail until chunker.go exists.
// No database required — pure unit tests.
// Traces: FR-090
package knowledge

import (
	"strings"
	"testing"
)

func TestChunker_EmptyInput_ReturnsNoChunks(t *testing.T) {
	chunks := Chunk("", 512, 50)
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for empty input, got %d", len(chunks))
	}
}

func TestChunker_WhitespaceOnlyInput_ReturnsNoChunks(t *testing.T) {
	chunks := Chunk("   \t\n  ", 512, 50)
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for whitespace-only input, got %d", len(chunks))
	}
}

func TestChunker_ShortText_ReturnsSingleChunk(t *testing.T) {
	text := "hello world this is a short document"
	chunks := Chunk(text, 512, 50)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for short text, got %d", len(chunks))
	}
	if chunks[0] != text {
		t.Errorf("expected chunk to equal input text, got %q", chunks[0])
	}
}

func TestChunker_ExactChunkSize_ReturnsSingleChunk(t *testing.T) {
	// Build a text with exactly 512 tokens
	words := make([]string, 512)
	for i := range words {
		words[i] = "word"
	}
	text := strings.Join(words, " ")
	chunks := Chunk(text, 512, 50)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for exactly chunkSize tokens, got %d", len(chunks))
	}
}

func TestChunker_LongText_ReturnsMultipleChunks(t *testing.T) {
	// Build a text with 1000 tokens — should produce multiple chunks with size=512, overlap=50
	words := make([]string, 1000)
	for i := range words {
		words[i] = "word"
	}
	text := strings.Join(words, " ")
	chunks := Chunk(text, 512, 50)
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks for 1000-token text, got %d", len(chunks))
	}
}

func TestChunker_OverlapPreservesTokens(t *testing.T) {
	// Build text: first 512 tokens are "alpha", next 100 are "beta"
	// With overlap=50, the second chunk should start 50 tokens before end of first chunk
	alphas := make([]string, 512)
	for i := range alphas {
		alphas[i] = "alpha"
	}
	betas := make([]string, 100)
	for i := range betas {
		betas[i] = "beta"
	}
	text := strings.Join(append(alphas, betas...), " ")
	chunks := Chunk(text, 512, 50)
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}
	// The second chunk must start with "alpha" tokens (from the overlap region)
	if !strings.HasPrefix(chunks[1], "alpha") {
		t.Errorf("expected second chunk to start with overlap 'alpha' tokens, got %q", chunks[1][:20])
	}
}

func TestChunker_AllChunksNonEmpty(t *testing.T) {
	words := make([]string, 2000)
	for i := range words {
		words[i] = "token"
	}
	text := strings.Join(words, " ")
	chunks := Chunk(text, 512, 50)
	for i, c := range chunks {
		if strings.TrimSpace(c) == "" {
			t.Errorf("chunk %d is empty or whitespace-only", i)
		}
	}
}

func TestChunker_TokenCount_PerChunk(t *testing.T) {
	// Each chunk should have at most chunkSize tokens
	words := make([]string, 1500)
	for i := range words {
		words[i] = "tok"
	}
	text := strings.Join(words, " ")
	chunkSize := 512
	chunks := Chunk(text, chunkSize, 50)
	for i, c := range chunks {
		tokenCount := len(strings.Fields(c))
		if tokenCount > chunkSize {
			t.Errorf("chunk %d has %d tokens, exceeds chunkSize %d", i, tokenCount, chunkSize)
		}
	}
}
