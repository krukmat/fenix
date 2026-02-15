// Task 2.3: Unit tests for OllamaProvider.
// Uses httptest.NewServer to mock the Ollama HTTP API — no real Ollama needed.
// Traces: FR-092
package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ============================================================================
// Embed tests
// ============================================================================

func TestOllamaProvider_Embed_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embeddings" || r.Method != http.MethodPost {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ollamaEmbedResponse{Embedding: []float32{0.1, 0.2, 0.3}}) //nolint:errcheck
	}))
	defer srv.Close()

	p := NewOllamaProvider(srv.URL, "nomic-embed-text")
	resp, err := p.Embed(context.Background(), EmbedRequest{Texts: []string{"hello world"}})
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}
	if len(resp.Embeddings) != 1 {
		t.Fatalf("expected 1 embedding, got %d", len(resp.Embeddings))
	}
	if len(resp.Embeddings[0]) != 3 {
		t.Errorf("expected 3 dims, got %d", len(resp.Embeddings[0]))
	}
}

func TestOllamaProvider_Embed_MultiText_CallsOncePerText(t *testing.T) {
	t.Parallel()

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ollamaEmbedResponse{Embedding: []float32{0.5}}) //nolint:errcheck
	}))
	defer srv.Close()

	p := NewOllamaProvider(srv.URL, "nomic-embed-text")
	resp, err := p.Embed(context.Background(), EmbedRequest{Texts: []string{"a", "b", "c"}})
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 HTTP calls (one per text), got %d", callCount)
	}
	if len(resp.Embeddings) != 3 {
		t.Errorf("expected 3 embeddings, got %d", len(resp.Embeddings))
	}
}

func TestOllamaProvider_Embed_ServerError_ReturnsError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := NewOllamaProvider(srv.URL, "nomic-embed-text")
	_, err := p.Embed(context.Background(), EmbedRequest{Texts: []string{"hello"}})
	if err == nil {
		t.Error("expected error for 500 response, got nil")
	}
}

func TestOllamaProvider_Embed_EmptyTexts_ReturnsEmptyEmbeddings(t *testing.T) {
	t.Parallel()

	p := NewOllamaProvider("http://localhost:99999", "nomic-embed-text")
	resp, err := p.Embed(context.Background(), EmbedRequest{Texts: []string{}})
	if err != nil {
		t.Fatalf("expected no error for empty texts, got %v", err)
	}
	if len(resp.Embeddings) != 0 {
		t.Errorf("expected 0 embeddings, got %d", len(resp.Embeddings))
	}
}

// ============================================================================
// ChatCompletion tests
// ============================================================================

func TestOllamaProvider_ChatCompletion_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" || r.Method != http.MethodPost {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ollamaChatResponse{ //nolint:errcheck
			Message:    ollamaChatMessage{Role: "assistant", Content: "Hello from Ollama"},
			DoneReason: "stop",
			Done:       true,
		})
	}))
	defer srv.Close()

	p := NewOllamaProvider(srv.URL, "llama3.2:3b")
	resp, err := p.ChatCompletion(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletion failed: %v", err)
	}
	if resp.Content != "Hello from Ollama" {
		t.Errorf("expected 'Hello from Ollama', got %q", resp.Content)
	}
	if resp.StopReason != "stop" {
		t.Errorf("expected StopReason 'stop', got %q", resp.StopReason)
	}
}

func TestOllamaProvider_ChatCompletion_ServerError_ReturnsError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer srv.Close()

	p := NewOllamaProvider(srv.URL, "llama3.2:3b")
	_, err := p.ChatCompletion(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Error("expected error for 400 response, got nil")
	}
}

// ============================================================================
// HealthCheck tests
// ============================================================================

func TestOllamaProvider_HealthCheck_Healthy(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"models": []interface{}{}}) //nolint:errcheck
	}))
	defer srv.Close()

	p := NewOllamaProvider(srv.URL, "nomic-embed-text")
	if err := p.HealthCheck(context.Background()); err != nil {
		t.Errorf("expected healthy, got error: %v", err)
	}
}

func TestOllamaProvider_HealthCheck_Down_ReturnsError(t *testing.T) {
	t.Parallel()

	// Use a server that immediately closes the connection.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Hijack and close without responding to simulate down server.
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
	}))
	srv.Close() // Closed before the health check call.

	p := NewOllamaProvider(srv.URL, "nomic-embed-text")
	if err := p.HealthCheck(context.Background()); err == nil {
		t.Error("expected error when server is down, got nil")
	}
}

// ============================================================================
// ModelInfo test
// ============================================================================

func TestOllamaProvider_ModelInfo_ReturnsMetadata(t *testing.T) {
	t.Parallel()

	p := NewOllamaProvider("http://localhost:11434", "nomic-embed-text")
	meta := p.ModelInfo()
	if meta.ID != "nomic-embed-text" {
		t.Errorf("expected model ID 'nomic-embed-text', got %q", meta.ID)
	}
	if meta.Provider != "ollama" {
		t.Errorf("expected provider 'ollama', got %q", meta.Provider)
	}
}

// ============================================================================
// buildChatOptions tests (Task 2.3 audit remediation)
// ============================================================================

func TestBuildChatOptions_WithTemperature(t *testing.T) {
	t.Parallel()

	req := ChatRequest{
		Messages:    []Message{{Role: "user", Content: "hi"}},
		Temperature: 0.7,
	}
	opts := buildChatOptions(req)
	if opts == nil {
		t.Fatal("expected non-nil opts map when Temperature is set")
	}
	temp, ok := opts["temperature"]
	if !ok {
		t.Error("expected 'temperature' key in opts")
	}
	if temp != float32(0.7) {
		t.Errorf("expected temperature 0.7, got %v", temp)
	}
}

func TestBuildChatOptions_WithMaxTokens(t *testing.T) {
	t.Parallel()

	req := ChatRequest{
		Messages:  []Message{{Role: "user", Content: "hi"}},
		MaxTokens: 256,
	}
	opts := buildChatOptions(req)
	if opts == nil {
		t.Fatal("expected non-nil opts map when MaxTokens is set")
	}
	predict, ok := opts["num_predict"]
	if !ok {
		t.Error("expected 'num_predict' key in opts")
	}
	if predict != 256 {
		t.Errorf("expected num_predict 256, got %v", predict)
	}
}

func TestBuildChatOptions_BothZero_ReturnsNil(t *testing.T) {
	t.Parallel()

	req := ChatRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
		// Temperature and MaxTokens left at zero values
	}
	opts := buildChatOptions(req)
	if opts != nil {
		t.Errorf("expected nil opts when both Temperature and MaxTokens are zero, got %v", opts)
	}
}
