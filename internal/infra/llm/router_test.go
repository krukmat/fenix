// Task 2.3: Unit tests for Router.
// Uses stub LLMProvider implementations (function struct) — no HTTP needed.
// Traces: FR-092
package llm

import (
	"context"
	"testing"
)

// stubProvider is a minimal LLMProvider stub for router testing.
type stubProvider struct{ id string }

func (s *stubProvider) ChatCompletion(_ context.Context, _ ChatRequest) (*ChatResponse, error) {
	return &ChatResponse{Content: "stub"}, nil
}
func (s *stubProvider) Embed(_ context.Context, _ EmbedRequest) (*EmbedResponse, error) {
	return &EmbedResponse{Embeddings: [][]float32{}}, nil
}
func (s *stubProvider) ModelInfo() ModelMeta { return ModelMeta{ID: s.id, Provider: "stub"} }
func (s *stubProvider) HealthCheck(_ context.Context) error { return nil }

// ============================================================================
// Router tests
// ============================================================================

func TestRouter_Route_ReturnsDefaultProvider(t *testing.T) {
	t.Parallel()

	ollama := &stubProvider{id: "nomic-embed-text"}
	r := NewRouter(map[string]LLMProvider{"ollama": ollama}, "ollama")

	p, err := r.Route(context.Background())
	if err != nil {
		t.Fatalf("Route failed: %v", err)
	}
	if p.ModelInfo().Provider != "stub" || p.ModelInfo().ID != "nomic-embed-text" {
		t.Errorf("unexpected provider returned: %v", p.ModelInfo())
	}
}

func TestRouter_Route_UnknownDefaultProvider_ReturnsError(t *testing.T) {
	t.Parallel()

	ollama := &stubProvider{id: "nomic-embed-text"}
	// defaultProvider key "openai" is not in the map — should return error.
	r := NewRouter(map[string]LLMProvider{"ollama": ollama}, "openai")

	_, err := r.Route(context.Background())
	if err == nil {
		t.Error("expected error for unknown defaultProvider, got nil")
	}
}

func TestRouter_Route_EmptyProviders_ReturnsError(t *testing.T) {
	t.Parallel()

	r := NewRouter(map[string]LLMProvider{}, "ollama")
	_, err := r.Route(context.Background())
	if err == nil {
		t.Error("expected error for empty providers map, got nil")
	}
}

func TestRouter_RegisterAndRoute_NewProvider(t *testing.T) {
	t.Parallel()

	r := NewRouter(map[string]LLMProvider{}, "ollama")
	ollama := &stubProvider{id: "llama3.2:3b"}
	r.Register("ollama", ollama)

	p, err := r.Route(context.Background())
	if err != nil {
		t.Fatalf("Route after Register failed: %v", err)
	}
	if p.ModelInfo().ID != "llama3.2:3b" {
		t.Errorf("expected llama3.2:3b, got %q", p.ModelInfo().ID)
	}
}
