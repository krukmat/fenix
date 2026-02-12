// Package llm defines the model-agnostic LLM provider abstraction (Task 2.3).
// All types here are shared between the provider interface and adapters.
package llm

// Message represents a single turn in a conversation (role + content).
type Message struct {
	Role    string // "system" | "user" | "assistant"
	Content string
}

// ChatRequest is the input for a non-streaming chat completion.
type ChatRequest struct {
	// Model overrides the provider default when non-empty.
	Model       string
	Messages    []Message
	Temperature float32
	MaxTokens   int
}

// ChatResponse is the output from a non-streaming chat completion.
type ChatResponse struct {
	Content    string // The assistant message text.
	StopReason string // "stop" | "length" | "error"
	Tokens     int    // Total tokens consumed (prompt + completion).
}

// EmbedRequest is the input for a batch embedding call.
type EmbedRequest struct {
	// Model overrides the provider default when non-empty.
	Model string
	Texts []string
}

// EmbedResponse is the output from a batch embedding call.
// Embeddings[i] corresponds to Texts[i] in the request.
type EmbedResponse struct {
	Embeddings [][]float32 // float32 matches sqlite-vec BLOB format.
	Tokens     int         // Total tokens consumed.
}

// ModelMeta describes the model / provider identity.
type ModelMeta struct {
	ID        string // e.g. "nomic-embed-text", "llama3.2:3b"
	Provider  string // e.g. "ollama", "openai"
	Version   string // e.g. "v1.5"
	MaxTokens int    // Maximum context window size.
}
