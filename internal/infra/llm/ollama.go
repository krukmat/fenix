// Package llm — Task 2.3: Ollama HTTP adapter.
// OllamaProvider calls the local Ollama REST API using stdlib net/http.
// Endpoints used:
//   - POST /api/embeddings  — single text embedding
//   - POST /api/chat        — non-streaming chat completion
//   - GET  /api/tags        — health check (lists available models)
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	mimeJSON          = "application/json"
	headerContentType = "Content-Type"
)

// OllamaProvider implements LLMProvider against a running Ollama instance (Task 2.3).
type OllamaProvider struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewOllamaProvider creates an OllamaProvider with a 30s default timeout.
func NewOllamaProvider(baseURL, model string) *OllamaProvider {
	return &OllamaProvider{
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ─── internal Ollama JSON types ──────────────────────────────────────────────

type ollamaEmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type ollamaEmbedResponse struct {
	Embedding []float32 `json:"embedding"`
}

type ollamaChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatRequest struct {
	Model    string              `json:"model"`
	Messages []ollamaChatMessage `json:"messages"`
	Stream   bool                `json:"stream"`
	Options  map[string]any      `json:"options,omitempty"`
}

type ollamaChatResponse struct {
	Message    ollamaChatMessage `json:"message"`
	DoneReason string            `json:"done_reason"`
	Done       bool              `json:"done"`
}

// ─── LLMProvider implementation ─────────────────────────────────────────────

// Embed computes embeddings for each text via POST /api/embeddings (one call per text).
// Ollama does not support batch embeddings in a single call.
func (p *OllamaProvider) Embed(ctx context.Context, req EmbedRequest) (*EmbedResponse, error) {
	if len(req.Texts) == 0 {
		return &EmbedResponse{Embeddings: [][]float32{}}, nil
	}

	model := req.Model
	if model == "" {
		model = p.model
	}

	embeddings := make([][]float32, 0, len(req.Texts))
	for _, text := range req.Texts {
		vec, err := p.embedOne(ctx, model, text)
		if err != nil {
			return nil, fmt.Errorf("ollama embed: %w", err)
		}
		embeddings = append(embeddings, vec)
	}
	return &EmbedResponse{Embeddings: embeddings}, nil
}

// embedOne sends a single /api/embeddings call and returns the vector.
func (p *OllamaProvider) embedOne(ctx context.Context, model, text string) ([]float32, error) {
	body, err := json.Marshal(ollamaEmbedRequest{Model: model, Prompt: text})
	if err != nil {
		return nil, err
	}

	respBody, postErr := p.doPost(ctx, "/api/embeddings", body)
	if postErr != nil {
		return nil, postErr
	}
	defer respBody.Close()

	var ollamaResp ollamaEmbedResponse
	if decodeErr := json.NewDecoder(respBody).Decode(&ollamaResp); decodeErr != nil {
		return nil, fmt.Errorf("decode embed response: %w", decodeErr)
	}
	return ollamaResp.Embedding, nil
}

// ChatCompletion performs a non-streaming chat via POST /api/chat.
func (p *OllamaProvider) ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	msgs := make([]ollamaChatMessage, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = ollamaChatMessage(m)
	}

	opts := buildChatOptions(req)
	body, err := json.Marshal(ollamaChatRequest{
		Model:    model,
		Messages: msgs,
		Stream:   false,
		Options:  opts,
	})
	if err != nil {
		return nil, err
	}

	respBody, postErr := p.doPost(ctx, "/api/chat", body)
	if postErr != nil {
		return nil, postErr
	}
	defer respBody.Close()

	var ollamaResp ollamaChatResponse
	if decodeErr := json.NewDecoder(respBody).Decode(&ollamaResp); decodeErr != nil {
		return nil, fmt.Errorf("decode chat response: %w", decodeErr)
	}
	return &ChatResponse{
		Content:    ollamaResp.Message.Content,
		StopReason: ollamaResp.DoneReason,
	}, nil
}

// buildChatOptions converts ChatRequest fields into Ollama options map.
func buildChatOptions(req ChatRequest) map[string]any {
	opts := map[string]any{}
	if req.Temperature != 0 {
		opts["temperature"] = req.Temperature
	}
	if req.MaxTokens != 0 {
		opts["num_predict"] = req.MaxTokens
	}
	if len(opts) == 0 {
		return nil
	}
	return opts
}

// ModelInfo returns static metadata for this provider/model.
func (p *OllamaProvider) ModelInfo() ModelMeta {
	return ModelMeta{
		ID:        p.model,
		Provider:  "ollama",
		Version:   "v1",
		MaxTokens: 4096,
	}
}

// HealthCheck calls GET /api/tags — returns nil if Ollama is reachable.
func (p *OllamaProvider) HealthCheck(ctx context.Context) error {
	url := p.baseURL + "/api/tags"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("ollama healthcheck: build request: %w", err)
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ollama healthcheck: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama healthcheck: status %d", resp.StatusCode)
	}
	return nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

// doPost sends a POST request to baseURL+path and returns the response body.
// Caller is responsible for closing the returned ReadCloser.
func (p *OllamaProvider) doPost(ctx context.Context, path string, body []byte) (io.ReadCloser, error) {
	url := p.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ollama post %s: build request: %w", path, err)
	}
	req.Header.Set(headerContentType, mimeJSON)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama post %s: %w", path, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close() //nolint:errcheck
		return nil, fmt.Errorf("ollama post %s: status %d", path, resp.StatusCode)
	}
	return resp.Body, nil
}
