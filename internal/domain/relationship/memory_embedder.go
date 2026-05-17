// Package relationship — Task B.5: MemoryEmbedder service.
// Consumes interaction_signal.created events, calls LLMProvider.Embed() on the
// signal summary, and persists vectors via an idempotent repository contract.
package relationship

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
	"github.com/matiasleandrokruk/fenix/internal/infra/llm"
)

const (
	memoryEmbedMaxRetries = 3
	memoryEmbedBaseDelay  = 100 * time.Millisecond
)

// EmbeddingRepository stores one embedding vector per interaction signal.
// UpsertSignalEmbedding must be idempotent by signal_id.
type EmbeddingRepository interface {
	UpsertSignalEmbedding(ctx context.Context, workspaceID, signalID string, vector []float32) error
}

type signalInput struct {
	workspaceID string
	signalID    string
	summary     string
}

// MemoryEmbedder subscribes to interaction signal creation events and writes embeddings.
type MemoryEmbedder struct {
	bus  eventbus.EventBus
	llm  llm.LLMProvider
	repo EmbeddingRepository
}

// NewMemoryEmbedder constructs a MemoryEmbedder with its required dependencies.
func NewMemoryEmbedder(bus eventbus.EventBus, provider llm.LLMProvider, repo EmbeddingRepository) *MemoryEmbedder {
	return &MemoryEmbedder{bus: bus, llm: provider, repo: repo}
}

// Run processes interaction signal events until ctx is cancelled.
func (m *MemoryEmbedder) Run(ctx context.Context) {
	ch := m.bus.Subscribe(TopicInteractionSignalCreated)
	for {
		select {
		case ev := <-ch:
			m.handle(ctx, ev)
		case <-ctx.Done():
			return
		}
	}
}

// handle parses the signal event, embeds the summary, and persists the vector.
func (m *MemoryEmbedder) handle(ctx context.Context, ev eventbus.Event) {
	input, err := parseSignalPayload(ev)
	if err != nil {
		log.Printf("relationship.MemoryEmbedder: parse payload topic=%s err=%v", ev.Topic, err)
		return
	}

	vector, err := m.callEmbedWithRetry(ctx, input.summary)
	if err != nil {
		log.Printf("relationship.MemoryEmbedder: embed topic=%s signal_id=%s err=%v", ev.Topic, input.signalID, err)
		return
	}

	if repoErr := m.repo.UpsertSignalEmbedding(ctx, input.workspaceID, input.signalID, vector); repoErr != nil {
		log.Printf("relationship.MemoryEmbedder: UpsertSignalEmbedding topic=%s signal_id=%s err=%v", ev.Topic, input.signalID, repoErr)
	}
}

func (m *MemoryEmbedder) callEmbedWithRetry(ctx context.Context, text string) ([]float32, error) {
	var lastErr error
	delay := memoryEmbedBaseDelay

	for attempt := 0; attempt < memoryEmbedMaxRetries; attempt++ {
		nextDelay, waitErr := waitForNextEmbedAttempt(ctx, attempt, delay)
		if waitErr != nil {
			return nil, waitErr
		}
		delay = nextDelay

		resp, err := m.llm.Embed(ctx, llm.EmbedRequest{Texts: []string{text}})
		if err != nil {
			lastErr = err
			continue
		}
		if len(resp.Embeddings) == 0 {
			lastErr = fmt.Errorf("embed response missing embeddings")
			continue
		}
		return resp.Embeddings[0], nil
	}

	return nil, fmt.Errorf("all %d retries failed: %w", memoryEmbedMaxRetries, lastErr)
}

func waitForNextEmbedAttempt(ctx context.Context, attempt int, delay time.Duration) (time.Duration, error) {
	if attempt == 0 {
		return delay, nil
	}

	select {
	case <-ctx.Done():
		return 0, fmt.Errorf("wait for embed retry: %w", ctx.Err())
	case <-time.After(delay):
		return delay * 2, nil
	}
}

func parseSignalPayload(ev eventbus.Event) (signalInput, error) {
	m, ok := ev.Payload.(map[string]any)
	if !ok {
		return signalInput{}, fmt.Errorf("payload is not map[string]any: %T", ev.Payload)
	}

	str := func(key string) string {
		v, _ := m[key].(string)
		return v
	}

	input := signalInput{
		workspaceID: str("workspace_id"),
		signalID:    str("signal_id"),
		summary:     str("summary"),
	}
	if input.workspaceID == "" || input.signalID == "" || input.summary == "" {
		return signalInput{}, fmt.Errorf("payload missing workspace_id, signal_id, or summary")
	}

	return input, nil
}
