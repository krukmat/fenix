// Unit tests for OpenAICompatProvider.
// Uses httptest.NewServer to mock the OpenAI-compatible API — no real API needed.
// Traces: FR-092
package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ============================================================================
// ChatCompletion tests
// ============================================================================

func TestOpenAICompatProvider_ChatCompletion_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Method != http.MethodPost {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openaiChatResponse{ //nolint:errcheck
			Choices: []openaiChoice{
				{
					Message:      openaiMessage{Role: "assistant", Content: "Hello from Groq"},
					FinishReason: "stop",
				},
			},
			Usage: openaiUsage{TotalTokens: 42},
		})
	}))
	defer srv.Close()

	p := NewOpenAICompatProvider(srv.URL, "test-key", "llama3-8b-8192")
	resp, err := p.ChatCompletion(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletion failed: %v", err)
	}
	if resp.Content != "Hello from Groq" {
		t.Errorf("expected 'Hello from Groq', got %q", resp.Content)
	}
	if resp.StopReason != "stop" {
		t.Errorf("expected StopReason 'stop', got %q", resp.StopReason)
	}
	if resp.Tokens != 42 {
		t.Errorf("expected 42 tokens, got %d", resp.Tokens)
	}
}

func TestOpenAICompatProvider_ChatCompletion_ServerError_ReturnsError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := NewOpenAICompatProvider(srv.URL, "test-key", "llama3-8b-8192")
	_, err := p.ChatCompletion(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Error("expected error for 500 response, got nil")
	}
}

func TestOpenAICompatProvider_ChatCompletion_EmptyChoices_ReturnsError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openaiChatResponse{Choices: []openaiChoice{}}) //nolint:errcheck
	}))
	defer srv.Close()

	p := NewOpenAICompatProvider(srv.URL, "test-key", "llama3-8b-8192")
	_, err := p.ChatCompletion(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Error("expected error for empty choices, got nil")
	}
}

func TestOpenAICompatProvider_ChatCompletion_AuthHeader(t *testing.T) {
	t.Parallel()

	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openaiChatResponse{ //nolint:errcheck
			Choices: []openaiChoice{{Message: openaiMessage{Role: "assistant", Content: "ok"}, FinishReason: "stop"}},
		})
	}))
	defer srv.Close()

	p := NewOpenAICompatProvider(srv.URL, "my-secret-key", "test-model")
	_, err := p.ChatCompletion(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletion failed: %v", err)
	}
	if gotAuth != "Bearer my-secret-key" {
		t.Errorf("expected Authorization 'Bearer my-secret-key', got %q", gotAuth)
	}
}

func TestOpenAICompatProvider_ChatCompletion_TemperatureAndMaxTokens(t *testing.T) {
	t.Parallel()

	var gotReq openaiChatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &gotReq) //nolint:errcheck
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openaiChatResponse{ //nolint:errcheck
			Choices: []openaiChoice{{Message: openaiMessage{Role: "assistant", Content: "ok"}, FinishReason: "stop"}},
		})
	}))
	defer srv.Close()

	p := NewOpenAICompatProvider(srv.URL, "", "test-model")
	_, err := p.ChatCompletion(context.Background(), ChatRequest{
		Messages:    []Message{{Role: "user", Content: "hi"}},
		Temperature: 0.7,
		MaxTokens:   256,
	})
	if err != nil {
		t.Fatalf("ChatCompletion failed: %v", err)
	}
	if gotReq.Temperature != 0.7 {
		t.Errorf("expected temperature 0.7, got %v", gotReq.Temperature)
	}
	if gotReq.MaxTokens != 256 {
		t.Errorf("expected max_tokens 256, got %d", gotReq.MaxTokens)
	}
}

func TestOpenAICompatProvider_ChatCompletion_ModelOverride(t *testing.T) {
	t.Parallel()

	var gotModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openaiChatRequest
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &req) //nolint:errcheck
		gotModel = req.Model
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openaiChatResponse{ //nolint:errcheck
			Choices: []openaiChoice{{Message: openaiMessage{Role: "assistant", Content: "ok"}, FinishReason: "stop"}},
		})
	}))
	defer srv.Close()

	p := NewOpenAICompatProvider(srv.URL, "", "default-model")
	_, err := p.ChatCompletion(context.Background(), ChatRequest{
		Model:    "override-model",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletion failed: %v", err)
	}
	if gotModel != "override-model" {
		t.Errorf("expected model 'override-model', got %q", gotModel)
	}
}

// ============================================================================
// Embed tests
// ============================================================================

func TestOpenAICompatProvider_Embed_ReturnsError(t *testing.T) {
	t.Parallel()

	p := NewOpenAICompatProvider("http://localhost:99999", "", "test-model")
	_, err := p.Embed(context.Background(), EmbedRequest{Texts: []string{"hello"}})
	if err == nil {
		t.Error("expected error from Embed, got nil")
	}
}

// ============================================================================
// HealthCheck tests
// ============================================================================

func TestOpenAICompatProvider_HealthCheck_Healthy(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"data": []any{}}) //nolint:errcheck
	}))
	defer srv.Close()

	p := NewOpenAICompatProvider(srv.URL, "test-key", "test-model")
	if err := p.HealthCheck(context.Background()); err != nil {
		t.Errorf("expected healthy, got error: %v", err)
	}
}

func TestOpenAICompatProvider_HealthCheck_Down_ReturnsError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
	}))
	srv.Close()

	p := NewOpenAICompatProvider(srv.URL, "test-key", "test-model")
	if err := p.HealthCheck(context.Background()); err == nil {
		t.Error("expected error when server is down, got nil")
	}
}

func TestOpenAICompatProvider_HealthCheck_AuthHeader(t *testing.T) {
	t.Parallel()

	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := NewOpenAICompatProvider(srv.URL, "health-key", "test-model")
	if err := p.HealthCheck(context.Background()); err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}
	if gotAuth != "Bearer health-key" {
		t.Errorf("expected 'Bearer health-key', got %q", gotAuth)
	}
}

// ============================================================================
// ModelInfo test
// ============================================================================

func TestOpenAICompatProvider_ModelInfo_ReturnsMetadata(t *testing.T) {
	t.Parallel()

	p := NewOpenAICompatProvider("http://localhost", "", "llama3-8b-8192")
	meta := p.ModelInfo()
	if meta.ID != "llama3-8b-8192" {
		t.Errorf("expected model ID 'llama3-8b-8192', got %q", meta.ID)
	}
	if meta.Provider != "openai-compat" {
		t.Errorf("expected provider 'openai-compat', got %q", meta.Provider)
	}
	if meta.MaxTokens != 8192 {
		t.Errorf("expected MaxTokens 8192, got %d", meta.MaxTokens)
	}
}
