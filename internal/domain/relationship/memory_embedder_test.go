package relationship

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
	"github.com/matiasleandrokruk/fenix/internal/infra/llm"
)

type fakeEmbedLLM struct {
	embeds    [][]float32
	errs      []error
	callCount int
}

func (f *fakeEmbedLLM) ChatCompletion(_ context.Context, _ llm.ChatRequest) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{}, nil
}

func (f *fakeEmbedLLM) Embed(_ context.Context, req llm.EmbedRequest) (*llm.EmbedResponse, error) {
	f.callCount++
	if len(req.Texts) != 1 {
		return nil, errors.New("expected single text")
	}

	index := f.callCount - 1
	if index < len(f.errs) && f.errs[index] != nil {
		return nil, f.errs[index]
	}
	if index < len(f.embeds) {
		return &llm.EmbedResponse{Embeddings: [][]float32{f.embeds[index]}}, nil
	}
	return &llm.EmbedResponse{Embeddings: [][]float32{{0.1, 0.2}}}, nil
}

func (f *fakeEmbedLLM) ModelInfo() llm.ModelMeta { return llm.ModelMeta{} }

func (f *fakeEmbedLLM) HealthCheck(_ context.Context) error { return nil }

type upsertSignalEmbeddingArgs struct {
	workspaceID string
	signalID    string
	vector      []float32
}

type fakeEmbeddingRepo struct {
	calls []upsertSignalEmbeddingArgs
	err   error
}

func (f *fakeEmbeddingRepo) UpsertSignalEmbedding(_ context.Context, workspaceID, signalID string, vector []float32) error {
	f.calls = append(f.calls, upsertSignalEmbeddingArgs{
		workspaceID: workspaceID,
		signalID:    signalID,
		vector:      vector,
	})
	return f.err
}

func makeSignalCreatedEvent(payload any) eventbus.Event {
	return eventbus.Event{
		Topic:   TopicInteractionSignalCreated,
		Payload: payload,
	}
}

func TestMemoryEmbedder_HandleSignalCreated(t *testing.T) {
	repo := &fakeEmbeddingRepo{}
	provider := &fakeEmbedLLM{embeds: [][]float32{{0.25, 0.5, 0.75}}}
	svc := NewMemoryEmbedder(eventbus.New(), provider, repo)

	svc.handle(context.Background(), makeSignalCreatedEvent(map[string]any{
		"workspace_id": "ws-1",
		"signal_id":    "sig-1",
		"summary":      "customer was highly engaged",
	}))

	if len(repo.calls) != 1 {
		t.Fatalf("expected 1 UpsertSignalEmbedding call, got %d", len(repo.calls))
	}
	got := repo.calls[0]
	if got.workspaceID != "ws-1" {
		t.Errorf("workspaceID: want %q, got %q", "ws-1", got.workspaceID)
	}
	if got.signalID != "sig-1" {
		t.Errorf("signalID: want %q, got %q", "sig-1", got.signalID)
	}
	if len(got.vector) != 3 || got.vector[0] != 0.25 || got.vector[2] != 0.75 {
		t.Errorf("vector: unexpected contents %#v", got.vector)
	}
}

func TestMemoryEmbedder_LLMErrorSkips(t *testing.T) {
	repo := &fakeEmbeddingRepo{}
	provider := &fakeEmbedLLM{errs: []error{
		errors.New("embed down"),
		errors.New("embed down"),
		errors.New("embed down"),
	}}
	svc := NewMemoryEmbedder(eventbus.New(), provider, repo)

	svc.handle(context.Background(), makeSignalCreatedEvent(map[string]any{
		"workspace_id": "ws-1",
		"signal_id":    "sig-1",
		"summary":      "customer was highly engaged",
	}))

	if len(repo.calls) != 0 {
		t.Errorf("expected 0 UpsertSignalEmbedding calls on LLM error, got %d", len(repo.calls))
	}
	if provider.callCount != memoryEmbedMaxRetries {
		t.Errorf("callCount: want %d, got %d", memoryEmbedMaxRetries, provider.callCount)
	}
}

func TestMemoryEmbedder_RepoErrorLogsAndContinues(t *testing.T) {
	repo := &fakeEmbeddingRepo{err: errors.New("db down")}
	provider := &fakeEmbedLLM{embeds: [][]float32{{0.1, 0.2}}}
	bus := eventbus.New()
	svc := NewMemoryEmbedder(bus, provider, repo)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		svc.Run(ctx)
		close(done)
	}()
	defer func() {
		cancel()
		<-done
	}()

	time.Sleep(10 * time.Millisecond)
	bus.Publish(TopicInteractionSignalCreated, map[string]any{
		"workspace_id": "ws-1",
		"signal_id":    "sig-1",
		"summary":      "first signal",
	})
	bus.Publish(TopicInteractionSignalCreated, map[string]any{
		"workspace_id": "ws-1",
		"signal_id":    "sig-2",
		"summary":      "second signal",
	})

	deadline := time.Now().Add(200 * time.Millisecond)
	for len(repo.calls) < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	if len(repo.calls) != 2 {
		t.Fatalf("expected 2 UpsertSignalEmbedding attempts, got %d", len(repo.calls))
	}
}

func TestMemoryEmbedder_MissingPayloadSkips(t *testing.T) {
	repo := &fakeEmbeddingRepo{}
	provider := &fakeEmbedLLM{}
	svc := NewMemoryEmbedder(eventbus.New(), provider, repo)

	svc.handle(context.Background(), makeSignalCreatedEvent(nil))
	svc.handle(context.Background(), makeSignalCreatedEvent(map[string]any{"workspace_id": "ws-1"}))

	if len(repo.calls) != 0 {
		t.Errorf("expected 0 UpsertSignalEmbedding calls on malformed payload, got %d", len(repo.calls))
	}
	if provider.callCount != 0 {
		t.Errorf("expected 0 embed calls on malformed payload, got %d", provider.callCount)
	}
}

func TestMemoryEmbedder_RetrySucceeds(t *testing.T) {
	repo := &fakeEmbeddingRepo{}
	provider := &fakeEmbedLLM{
		errs:   []error{errors.New("temporary failure"), nil},
		embeds: [][]float32{{0.3, 0.4}},
	}
	svc := NewMemoryEmbedder(eventbus.New(), provider, repo)

	svc.handle(context.Background(), makeSignalCreatedEvent(map[string]any{
		"workspace_id": "ws-1",
		"signal_id":    "sig-1",
		"summary":      "customer opened the proposal twice",
	}))

	if provider.callCount != 2 {
		t.Fatalf("expected 2 embed calls, got %d", provider.callCount)
	}
	if len(repo.calls) != 1 {
		t.Fatalf("expected 1 UpsertSignalEmbedding call, got %d", len(repo.calls))
	}
}

func TestMemoryEmbedder_StopsOnContextCancel(t *testing.T) {
	repo := &fakeEmbeddingRepo{}
	provider := &fakeEmbedLLM{}
	svc := NewMemoryEmbedder(eventbus.New(), provider, repo)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		svc.Run(ctx)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Run did not stop after context cancel within 200ms")
	}
}
