// Task 2.3 audit remediation: tests for config.Load and envOr.
// No t.Parallel() — env vars are process-global and not thread-safe.
package config

import "testing"

func TestLoad_Defaults(t *testing.T) {
	// Ensure env vars are unset so defaults apply.
	t.Setenv("LLM_PROVIDER", "")
	t.Setenv("OLLAMA_BASE_URL", "")
	t.Setenv("OLLAMA_MODEL", "")
	t.Setenv("OLLAMA_CHAT_MODEL", "")

	cfg := Load()

	if cfg.LLMProvider != "ollama" {
		t.Errorf("expected LLMProvider 'ollama', got %q", cfg.LLMProvider)
	}
	if cfg.OllamaBaseURL != "http://localhost:11434" {
		t.Errorf("expected OllamaBaseURL 'http://localhost:11434', got %q", cfg.OllamaBaseURL)
	}
	if cfg.OllamaModel != "nomic-embed-text" {
		t.Errorf("expected OllamaModel 'nomic-embed-text', got %q", cfg.OllamaModel)
	}
	if cfg.OllamaChatModel != "llama3.2:3b" {
		t.Errorf("expected OllamaChatModel 'llama3.2:3b', got %q", cfg.OllamaChatModel)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "openai")
	t.Setenv("OLLAMA_BASE_URL", "http://ollama.internal:11434")
	t.Setenv("OLLAMA_MODEL", "mxbai-embed-large")
	t.Setenv("OLLAMA_CHAT_MODEL", "llama3.1:8b")

	cfg := Load()

	if cfg.LLMProvider != "openai" {
		t.Errorf("expected LLMProvider 'openai', got %q", cfg.LLMProvider)
	}
	if cfg.OllamaBaseURL != "http://ollama.internal:11434" {
		t.Errorf("expected custom OllamaBaseURL, got %q", cfg.OllamaBaseURL)
	}
	if cfg.OllamaModel != "mxbai-embed-large" {
		t.Errorf("expected OllamaModel 'mxbai-embed-large', got %q", cfg.OllamaModel)
	}
	if cfg.OllamaChatModel != "llama3.1:8b" {
		t.Errorf("expected OllamaChatModel 'llama3.1:8b', got %q", cfg.OllamaChatModel)
	}
}

func TestEnvOr_Present(t *testing.T) {
	t.Setenv("TEST_ENVOR_KEY", "custom-value")
	got := envOr("TEST_ENVOR_KEY", "fallback")
	if got != "custom-value" {
		t.Errorf("expected 'custom-value', got %q", got)
	}
}

func TestEnvOr_Absent(t *testing.T) {
	t.Setenv("TEST_ENVOR_MISSING", "")
	got := envOr("TEST_ENVOR_MISSING", "fallback")
	if got != "fallback" {
		t.Errorf("expected 'fallback', got %q", got)
	}
}
