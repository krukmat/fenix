// Package llm — OpenAI-compatible HTTP adapter.
// OpenAICompatProvider calls any OpenAI-compatible API (Gradient, Groq, Together.ai, vLLM).
// Endpoints used:
//   - POST /v1/chat/completions — non-streaming chat completion
//   - GET  /v1/models           — health check (lists available models)
//
// Embeddings are NOT supported — use OllamaProvider for embeddings.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const headerAuthorization = "Authorization"

// OpenAICompatProvider implements LLMProvider against any OpenAI-compatible API.
type OpenAICompatProvider struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewOpenAICompatProvider creates a provider targeting an OpenAI-compatible endpoint.
func NewOpenAICompatProvider(baseURL, apiKey, model string) *OpenAICompatProvider {
	return &OpenAICompatProvider{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ─── internal OpenAI JSON types ─────────────────────────────────────────────

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiChatRequest struct {
	Model       string          `json:"model"`
	Messages    []openaiMessage `json:"messages"`
	Temperature float32         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
}

type openaiChoice struct {
	Message      openaiMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openaiUsage struct {
	TotalTokens int `json:"total_tokens"`
}

type openaiChatResponse struct {
	Choices []openaiChoice `json:"choices"`
	Usage   openaiUsage    `json:"usage"`
}

// ─── LLMProvider implementation ─────────────────────────────────────────────

// ChatCompletion performs a non-streaming chat via POST /v1/chat/completions.
func (p *OpenAICompatProvider) ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	body, err := json.Marshal(buildOpenAIChatRequest(req, p.model))
	if err != nil {
		return nil, fmt.Errorf("openai-compat: marshal request: %w", err)
	}

	respBody, postErr := p.doRequest(ctx, http.MethodPost, "/v1/chat/completions", body)
	if postErr != nil {
		return nil, postErr
	}
	defer respBody.Close()

	return decodeChatResponse(respBody)
}

func buildOpenAIChatRequest(req ChatRequest, defaultModel string) openaiChatRequest {
	oaiReq := openaiChatRequest{
		Model:    coalesceModel(req.Model, defaultModel),
		Messages: toOpenAIMessages(req.Messages),
	}
	if req.Temperature != 0 {
		oaiReq.Temperature = req.Temperature
	}
	if req.MaxTokens != 0 {
		oaiReq.MaxTokens = req.MaxTokens
	}
	return oaiReq
}

func coalesceModel(model, fallback string) string {
	if model != "" {
		return model
	}
	return fallback
}

func toOpenAIMessages(messages []Message) []openaiMessage {
	msgs := make([]openaiMessage, len(messages))
	for i, m := range messages {
		msgs[i] = openaiMessage(m)
	}
	return msgs
}

func decodeChatResponse(respBody io.Reader) (*ChatResponse, error) {
	var oaiResp openaiChatResponse
	if err := json.NewDecoder(respBody).Decode(&oaiResp); err != nil {
		return nil, fmt.Errorf("openai-compat: decode response: %w", err)
	}
	if len(oaiResp.Choices) == 0 {
		return nil, fmt.Errorf("openai-compat: empty choices in response")
	}
	choice := oaiResp.Choices[0]
	return &ChatResponse{
		Content:    choice.Message.Content,
		StopReason: choice.FinishReason,
		Tokens:     oaiResp.Usage.TotalTokens,
	}, nil
}

// Embed is not supported by the OpenAI-compatible provider.
// Use OllamaProvider for embeddings.
func (p *OpenAICompatProvider) Embed(_ context.Context, _ EmbedRequest) (*EmbedResponse, error) {
	return nil, fmt.Errorf("openai-compat provider does not support embeddings; use ollama for embed")
}

// ModelInfo returns static metadata for this provider/model.
func (p *OpenAICompatProvider) ModelInfo() ModelMeta {
	return ModelMeta{
		ID:        p.model,
		Provider:  "openai-compat",
		Version:   "v1",
		MaxTokens: 8192,
	}
}

// HealthCheck calls GET /v1/models — returns nil if the API is reachable.
func (p *OpenAICompatProvider) HealthCheck(ctx context.Context) error {
	respBody, err := p.doRequest(ctx, http.MethodGet, "/v1/models", nil)
	if err != nil {
		return fmt.Errorf("openai-compat healthcheck: %w", err)
	}
	respBody.Close()
	return nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

// doRequest sends an HTTP request with the Authorization header and returns the response body.
// Caller is responsible for closing the returned ReadCloser.
func (p *OpenAICompatProvider) doRequest(ctx context.Context, method, path string, body []byte) (io.ReadCloser, error) {
	url := p.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, requestBodyReader(body))
	if err != nil {
		return nil, fmt.Errorf("openai-compat %s %s: build request: %w", method, path, err)
	}
	setOpenAICompatHeaders(req, body != nil, p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai-compat %s %s: %w", method, path, err)
	}
	if statusErr := ensureSuccessStatus(resp, method, path); statusErr != nil {
		return nil, statusErr
	}
	return resp.Body, nil
}

func requestBodyReader(body []byte) io.Reader {
	if body == nil {
		return nil
	}
	return bytes.NewReader(body)
}

func setOpenAICompatHeaders(req *http.Request, hasBody bool, apiKey string) {
	if hasBody {
		req.Header.Set(headerContentType, mimeJSON)
	}
	if apiKey != "" {
		req.Header.Set(headerAuthorization, "Bearer "+apiKey)
	}
}

func ensureSuccessStatus(resp *http.Response, method, path string) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	errBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close() //nolint:errcheck
	log.Printf("[openai-compat] %s %s: status %d body=%s", method, path, resp.StatusCode, string(errBody))
	return fmt.Errorf("openai-compat %s %s: status %d", method, path, resp.StatusCode)
}
