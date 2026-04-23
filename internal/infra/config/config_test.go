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
	t.Setenv("CHAT_PROVIDER", "")
	t.Setenv("EMBED_PROVIDER", "")
	t.Setenv("OPENAI_COMPAT_BASE_URL", "")
	t.Setenv("OPENAI_COMPAT_API_KEY", "")
	t.Setenv("OPENAI_COMPAT_MODEL", "")
	t.Setenv("BFF_ORIGIN", "")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")

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
	if cfg.OllamaChatModel != "gemma4:e4b" {
		t.Errorf("expected OllamaChatModel 'gemma4:e4b', got %q", cfg.OllamaChatModel)
	}
	// Split provider defaults: ChatProvider falls back to LLMProvider ("ollama").
	if cfg.ChatProvider != "ollama" {
		t.Errorf("expected ChatProvider 'ollama', got %q", cfg.ChatProvider)
	}
	if cfg.EmbedProvider != "ollama" {
		t.Errorf("expected EmbedProvider 'ollama', got %q", cfg.EmbedProvider)
	}
	if cfg.OpenAICompatBaseURL != "" {
		t.Errorf("expected empty OpenAICompatBaseURL, got %q", cfg.OpenAICompatBaseURL)
	}
	if !containsString(cfg.CORSAllowedOrigins, "http://localhost:3000") || !containsString(cfg.CORSAllowedOrigins, "http://localhost:5173") {
		t.Errorf("expected default CORSAllowedOrigins to include BFF and local dev origins, got %#v", cfg.CORSAllowedOrigins)
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

func TestLoad_ChatProvider_FallsBackToLLMProvider(t *testing.T) {
	t.Setenv("CHAT_PROVIDER", "")
	t.Setenv("LLM_PROVIDER", "custom-llm")

	cfg := Load()
	if cfg.ChatProvider != "custom-llm" {
		t.Errorf("expected ChatProvider to fall back to LLM_PROVIDER 'custom-llm', got %q", cfg.ChatProvider)
	}
}

func TestLoad_ChatProvider_OverridesLLMProvider(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "ollama")
	t.Setenv("CHAT_PROVIDER", "openai-compat")

	cfg := Load()
	if cfg.ChatProvider != "openai-compat" {
		t.Errorf("expected ChatProvider 'openai-compat', got %q", cfg.ChatProvider)
	}
}

func TestLoad_OpenAICompatFields(t *testing.T) {
	t.Setenv("CHAT_PROVIDER", "openai-compat")
	t.Setenv("OPENAI_COMPAT_BASE_URL", "https://api.groq.com/openai")
	t.Setenv("OPENAI_COMPAT_API_KEY", "gsk_test123")
	t.Setenv("OPENAI_COMPAT_MODEL", "llama3-8b-8192")

	cfg := Load()
	if cfg.OpenAICompatBaseURL != "https://api.groq.com/openai" {
		t.Errorf("expected OpenAICompatBaseURL, got %q", cfg.OpenAICompatBaseURL)
	}
	if cfg.OpenAICompatAPIKey != "gsk_test123" {
		t.Errorf("expected OpenAICompatAPIKey, got %q", cfg.OpenAICompatAPIKey)
	}
	if cfg.OpenAICompatModel != "llama3-8b-8192" {
		t.Errorf("expected OpenAICompatModel, got %q", cfg.OpenAICompatModel)
	}
}

func TestLoad_CORSAllowedOriginsOverride(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://bff.example.com, http://localhost:5173, https://bff.example.com")

	cfg := Load()
	want := []string{"https://bff.example.com", "http://localhost:5173"}
	if !equalStrings(cfg.CORSAllowedOrigins, want) {
		t.Errorf("CORSAllowedOrigins = %#v; want %#v", cfg.CORSAllowedOrigins, want)
	}
}

func TestLoad_BFFOriginFeedsDefaultCORSAllowlist(t *testing.T) {
	t.Setenv("BFF_ORIGIN", "https://bff.internal")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")

	cfg := Load()
	if !containsString(cfg.CORSAllowedOrigins, "https://bff.internal") {
		t.Errorf("CORSAllowedOrigins = %#v; want BFF origin included", cfg.CORSAllowedOrigins)
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

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
