package llm

import (
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/infra/config"
)

func TestNewChatProvider_Ollama(t *testing.T) {
	t.Parallel()

	provider, err := NewChatProvider(config.Config{
		ChatProvider:      "ollama",
		OllamaBaseURL:     "http://localhost:11434",
		OllamaModel:       "nomic-embed-text",
		OllamaChatModel:   "llama3.2:3b",
		EmbedProvider:     "ollama",
		OpenAICompatModel: "",
	})
	if err != nil {
		t.Fatalf("NewChatProvider returned error: %v", err)
	}
	if _, ok := provider.(*OllamaProvider); !ok {
		t.Fatalf("expected *OllamaProvider, got %T", provider)
	}
}

func TestNewChatProvider_OpenAICompat(t *testing.T) {
	t.Parallel()

	provider, err := NewChatProvider(config.Config{
		ChatProvider:        "openai-compat",
		OpenAICompatBaseURL: "https://api.groq.com/openai",
		OpenAICompatAPIKey:  "gsk_test",
		OpenAICompatModel:   "llama3-8b-8192",
		OllamaBaseURL:       "http://localhost:11434",
		OllamaModel:         "nomic-embed-text",
		OllamaChatModel:     "llama3.2:3b",
	})
	if err != nil {
		t.Fatalf("NewChatProvider returned error: %v", err)
	}
	if _, ok := provider.(*OpenAICompatProvider); !ok {
		t.Fatalf("expected *OpenAICompatProvider, got %T", provider)
	}
}

func TestNewChatProvider_Unknown_ReturnsError(t *testing.T) {
	t.Parallel()

	_, err := NewChatProvider(config.Config{ChatProvider: "gradient"})
	if err == nil {
		t.Fatal("expected error for unknown chat provider")
	}
}

func TestNewEmbedProvider_Ollama(t *testing.T) {
	t.Parallel()

	provider, err := NewEmbedProvider(config.Config{
		EmbedProvider:   "ollama",
		OllamaBaseURL:   "http://localhost:11434",
		OllamaModel:     "nomic-embed-text",
		OllamaChatModel: "llama3.2:3b",
	})
	if err != nil {
		t.Fatalf("NewEmbedProvider returned error: %v", err)
	}
	if _, ok := provider.(*OllamaProvider); !ok {
		t.Fatalf("expected *OllamaProvider, got %T", provider)
	}
}

func TestNewEmbedProvider_Unknown_ReturnsError(t *testing.T) {
	t.Parallel()

	_, err := NewEmbedProvider(config.Config{EmbedProvider: "openai-compat"})
	if err == nil {
		t.Fatal("expected error for unknown embed provider")
	}
}
