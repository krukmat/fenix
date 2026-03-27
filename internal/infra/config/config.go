// Package config provides application-wide configuration loaded from env vars (Task 2.3).
// All fields have safe defaults so the binary runs locally without any env setup.
package config

import "os"

// Config holds runtime configuration for FenixCRM.
type Config struct {
	// LLM (legacy — used as fallback when ChatProvider/EmbedProvider are not set)
	LLMProvider     string // LLM_PROVIDER — default: "ollama"
	OllamaBaseURL   string // OLLAMA_BASE_URL — default: "http://localhost:11434"
	OllamaModel     string // OLLAMA_MODEL — default: "nomic-embed-text" (embed model, 768 dims)
	OllamaChatModel string // OLLAMA_CHAT_MODEL — default: "llama3.2:3b"

	// Split provider config — POC deployment readiness.
	// ChatProvider selects the provider for chat/completions ("ollama"|"openai-compat").
	// Falls back to LLMProvider if unset, then to "ollama".
	ChatProvider string // CHAT_PROVIDER
	// EmbedProvider selects the provider for embeddings. Only "ollama" supported today.
	EmbedProvider string // EMBED_PROVIDER — default: "ollama"
	// OpenAI-compatible provider settings (used when ChatProvider == "openai-compat").
	OpenAICompatBaseURL string // OPENAI_COMPAT_BASE_URL
	OpenAICompatAPIKey  string // OPENAI_COMPAT_API_KEY
	OpenAICompatModel   string // OPENAI_COMPAT_MODEL

	// Security
	// BFFOrigin is the single allowed CORS origin for the BFF (Express gateway).
	// Set via BFF_ORIGIN env var. Default: "http://localhost:3000".
	BFFOrigin string // BFF_ORIGIN — default: "http://localhost:3000"
}

const (
	envKeyLLMProvider     = "LLM_PROVIDER"
	envKeyOllamaBaseURL   = "OLLAMA_BASE_URL"
	envKeyOllamaModel     = "OLLAMA_MODEL"
	envKeyOllamaChatModel = "OLLAMA_CHAT_MODEL"
	envKeyBFFOrigin       = "BFF_ORIGIN"

	envKeyChatProvider        = "CHAT_PROVIDER"
	envKeyEmbedProvider       = "EMBED_PROVIDER"
	envKeyOpenAICompatBaseURL = "OPENAI_COMPAT_BASE_URL"
	envKeyOpenAICompatAPIKey  = "OPENAI_COMPAT_API_KEY"
	envKeyOpenAICompatModel   = "OPENAI_COMPAT_MODEL"
)

// Load reads configuration from environment variables, applying defaults for missing values.
func Load() Config {
	llmProvider := envOr(envKeyLLMProvider, "ollama")

	// ChatProvider: CHAT_PROVIDER → LLM_PROVIDER → "ollama"
	chatProvider := envOr(envKeyChatProvider, "")
	if chatProvider == "" {
		chatProvider = llmProvider
	}

	return Config{
		LLMProvider:         llmProvider,
		OllamaBaseURL:       envOr(envKeyOllamaBaseURL, "http://localhost:11434"),
		OllamaModel:         envOr(envKeyOllamaModel, "nomic-embed-text"),
		OllamaChatModel:     envOr(envKeyOllamaChatModel, "llama3.2:3b"),
		ChatProvider:        chatProvider,
		EmbedProvider:       envOr(envKeyEmbedProvider, "ollama"),
		OpenAICompatBaseURL: envOr(envKeyOpenAICompatBaseURL, ""),
		OpenAICompatAPIKey:  envOr(envKeyOpenAICompatAPIKey, ""),
		OpenAICompatModel:   envOr(envKeyOpenAICompatModel, ""),
		BFFOrigin:           envOr(envKeyBFFOrigin, "http://localhost:3000"),
	}
}

// envOr returns the value of the environment variable key, or fallback if not set.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
