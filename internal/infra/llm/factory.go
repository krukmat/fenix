package llm

import (
	"fmt"

	"github.com/matiasleandrokruk/fenix/internal/infra/config"
)

const (
	providerOllama       = "ollama"
	providerOpenAICompat = "openai-compat"
)

// NewChatProvider creates the chat/completions provider selected in config.
func NewChatProvider(cfg config.Config) (LLMProvider, error) {
	switch cfg.ChatProvider {
	case "", providerOllama:
		return NewOllamaProvider(cfg.OllamaBaseURL, cfg.OllamaModel, cfg.OllamaChatModel), nil
	case providerOpenAICompat:
		return NewOpenAICompatProvider(cfg.OpenAICompatBaseURL, cfg.OpenAICompatAPIKey, cfg.OpenAICompatModel), nil
	default:
		return nil, fmt.Errorf("llm chat provider %q is not supported", cfg.ChatProvider)
	}
}

// NewEmbedProvider creates the embeddings provider selected in config.
func NewEmbedProvider(cfg config.Config) (LLMProvider, error) {
	switch cfg.EmbedProvider {
	case "", providerOllama:
		return NewOllamaProvider(cfg.OllamaBaseURL, cfg.OllamaModel, cfg.OllamaChatModel), nil
	default:
		return nil, fmt.Errorf("llm embed provider %q is not supported", cfg.EmbedProvider)
	}
}
