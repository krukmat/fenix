// Task 2.3: Compile-time interface satisfaction check.
// Ensures *OllamaProvider satisfies LLMProvider without running any HTTP calls.
// Traces: FR-092
package llm

import "testing"

// TestOllamaProvider_ImplementsLLMProvider is a compile-time check.
// If OllamaProvider does not satisfy LLMProvider, this file will not compile.
func TestOllamaProvider_ImplementsLLMProvider(t *testing.T) {
	t.Parallel()

	// compile-time assertion: *OllamaProvider must implement LLMProvider.
	var _ LLMProvider = &OllamaProvider{}
}
