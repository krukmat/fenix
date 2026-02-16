// Package config provides application-wide configuration loaded from env vars (Task 2.3).
// All fields have safe defaults so the binary runs locally without any env setup.
package config

import "os"

// Config holds runtime configuration for FenixCRM.
type Config struct {
	// LLM
	LLMProvider     string // LLM_PROVIDER — default: "ollama"
	OllamaBaseURL   string // OLLAMA_BASE_URL — default: "http://localhost:11434"
	OllamaModel     string // OLLAMA_MODEL — default: "nomic-embed-text" (embed model, 768 dims)
	OllamaChatModel string // OLLAMA_CHAT_MODEL — default: "llama3.2:3b"
}

const (
	envKeyLLMProvider     = "LLM_PROVIDER"
	envKeyOllamaBaseURL   = "OLLAMA_BASE_URL"
	envKeyOllamaModel     = "OLLAMA_MODEL"
	envKeyOllamaChatModel = "OLLAMA_CHAT_MODEL"
)

// Load reads configuration from environment variables, applying defaults for missing values.
func Load() Config {
	return Config{
		LLMProvider:     envOr(envKeyLLMProvider, "ollama"),
		OllamaBaseURL:   envOr(envKeyOllamaBaseURL, "http://localhost:11434"),
		OllamaModel:     envOr(envKeyOllamaModel, "nomic-embed-text"),
		OllamaChatModel: envOr(envKeyOllamaChatModel, "llama3.2:3b"),
	}
}

// envOr returns the value of the environment variable key, or fallback if not set.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
