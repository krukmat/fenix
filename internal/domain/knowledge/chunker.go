// Package knowledge — Task 2.2: Text chunker for the ingestion pipeline.
// Chunk splits a text into fixed-size token windows with overlap.
// Uses whitespace tokenization (no external dependencies, MVP constraint).
package knowledge

import "strings"

// Chunk splits text into slices of at most chunkSize tokens, advancing by
// (chunkSize - overlap) tokens between chunks so consecutive chunks share
// overlap tokens at their boundary.
//
// Rules:
//   - Empty or whitespace-only input returns nil (no chunks created).
//   - Text shorter than chunkSize returns a single chunk equal to the full text.
//   - Each returned chunk is the joined text of its tokens (single space separator).
//   - overlap must be < chunkSize; if not, overlap is clamped to chunkSize-1.
//
// Token definition: whitespace-separated word (strings.Fields).
func Chunk(text string, chunkSize, overlap int) []string {
	tokens := strings.Fields(text)
	if len(tokens) == 0 {
		return nil
	}

	// Clamp overlap to a safe value
	if overlap >= chunkSize {
		overlap = chunkSize - 1
	}

	// Short text: fits in a single chunk
	if len(tokens) <= chunkSize {
		return []string{strings.Join(tokens, " ")}
	}

	stride := chunkSize - overlap
	var chunks []string

	for start := 0; start < len(tokens); start += stride {
		end := start + chunkSize
		if end > len(tokens) {
			end = len(tokens)
		}
		chunks = append(chunks, strings.Join(tokens[start:end], " "))
		// Stop if this chunk reached the end of the token stream
		if end == len(tokens) {
			break
		}
	}

	return chunks
}
