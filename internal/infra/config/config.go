// Package config provides application-wide configuration loaded from env vars (Task 2.3).
// All fields have safe defaults so the binary runs locally without any env setup.
package config

import (
	"os"
	"strings"
)

// Config holds runtime configuration for FenixCRM.
type Config struct {
	// LLM (legacy — used as fallback when ChatProvider/EmbedProvider are not set)
	LLMProvider     string // LLM_PROVIDER — default: "ollama"
	OllamaBaseURL   string // OLLAMA_BASE_URL — default: "http://localhost:11434"
	OllamaModel     string // OLLAMA_MODEL — default: "nomic-embed-text" (embed model, 768 dims)
	OllamaChatModel string // OLLAMA_CHAT_MODEL — default: "gemma4:e4b"

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
	// BFFOrigin is the primary allowed CORS origin for the BFF (Express gateway).
	// CORSAllowedOrigins is the full browser origin allowlist for Go API CORS.
	// Set via CORS_ALLOWED_ORIGINS as a comma-separated list, or BFF_ORIGIN for legacy single-origin config.
	BFFOrigin          string   // BFF_ORIGIN — default: "http://localhost:3000"
	CORSAllowedOrigins []string // CORS_ALLOWED_ORIGINS — default: BFFOrigin + local dev origins
}

const (
	defaultProviderOllama = "ollama"

	envKeyLLMProvider     = "LLM_PROVIDER"
	envKeyOllamaBaseURL   = "OLLAMA_BASE_URL"
	envKeyOllamaModel     = "OLLAMA_MODEL"
	envKeyOllamaChatModel = "OLLAMA_CHAT_MODEL"
	envKeyBFFOrigin       = "BFF_ORIGIN"
	envKeyCORSOrigins     = "CORS_ALLOWED_ORIGINS"

	envKeyChatProvider        = "CHAT_PROVIDER"
	envKeyEmbedProvider       = "EMBED_PROVIDER"
	envKeyOpenAICompatBaseURL = "OPENAI_COMPAT_BASE_URL"
	//nolint:gosec // env var key name, not a credential value
	envKeyOpenAICompatAPIKey = "OPENAI_COMPAT_API_KEY"
	envKeyOpenAICompatModel  = "OPENAI_COMPAT_MODEL"
)

// Load reads configuration from environment variables, applying defaults for missing values.
func Load() Config {
	llmProvider := envOr(envKeyLLMProvider, defaultProviderOllama)

	// ChatProvider: CHAT_PROVIDER → LLM_PROVIDER → "ollama"
	chatProvider := envOr(envKeyChatProvider, "")
	if chatProvider == "" {
		chatProvider = llmProvider
	}

	bffOrigin := envOr(envKeyBFFOrigin, "http://localhost:3000")
	return Config{
		LLMProvider:         llmProvider,
		OllamaBaseURL:       envOr(envKeyOllamaBaseURL, "http://localhost:11434"),
		OllamaModel:         envOr(envKeyOllamaModel, "nomic-embed-text"),
		OllamaChatModel:     envOr(envKeyOllamaChatModel, "gemma4:e4b"),
		ChatProvider:        chatProvider,
		EmbedProvider:       envOr(envKeyEmbedProvider, defaultProviderOllama),
		OpenAICompatBaseURL: envOr(envKeyOpenAICompatBaseURL, ""),
		OpenAICompatAPIKey:  envOr(envKeyOpenAICompatAPIKey, ""),
		OpenAICompatModel:   envOr(envKeyOpenAICompatModel, ""),
		BFFOrigin:           bffOrigin,
		CORSAllowedOrigins:  corsAllowedOrigins(bffOrigin),
	}
}

func corsAllowedOrigins(bffOrigin string) []string {
	configured := splitCSV(os.Getenv(envKeyCORSOrigins))
	if len(configured) > 0 {
		return configured
	}
	return uniqueStrings([]string{
		bffOrigin,
		"http://localhost:3000",
		"http://localhost:3001",
		"http://localhost:5173",
		"http://127.0.0.1:3000",
		"http://127.0.0.1:3001",
		"http://127.0.0.1:5173",
	})
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return uniqueStrings(out)
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok || value == "" {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

// envOr returns the value of the environment variable key, or fallback if not set.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
