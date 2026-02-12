// Package llm — Task 2.3: LLMProvider interface.
// Adapters (Ollama, OpenAI, etc.) implement this interface so the application
// is never coupled to a specific LLM vendor.
package llm

import "context"

// LLMProvider is the model-agnostic interface for LLM operations (Task 2.3).
// MVP methods: ChatCompletion, Embed, ModelInfo, HealthCheck.
// ChatCompletionStream is excluded from MVP (adds goroutine complexity not needed yet).
type LLMProvider interface {
	// ChatCompletion performs a non-streaming chat completion.
	ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error)

	// Embed computes dense vector representations for a batch of texts.
	Embed(ctx context.Context, req EmbedRequest) (*EmbedResponse, error)

	// ModelInfo returns static metadata about the provider/model.
	ModelInfo() ModelMeta

	// HealthCheck returns nil if the provider is reachable and operational.
	HealthCheck(ctx context.Context) error
}
