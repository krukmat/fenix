// Package llm — Task 2.3: LLM provider router.
// Router selects a LLMProvider at request time.
// MVP: pass-through to defaultProvider.
// Future (Task 3.x): add no-cloud policy, budget checks, fallback chains.
package llm

import (
	"context"
	"fmt"
)

// Router selects a LLMProvider for each request (Task 2.3).
type Router struct {
	providers       map[string]LLMProvider
	defaultProvider string
}

// NewRouter creates a Router with an initial set of providers and a default key.
func NewRouter(providers map[string]LLMProvider, defaultProvider string) *Router {
	// defensive copy so the caller cannot mutate the internal map.
	ps := make(map[string]LLMProvider, len(providers))
	for k, v := range providers {
		ps[k] = v
	}
	return &Router{providers: ps, defaultProvider: defaultProvider}
}

// Register adds (or replaces) a provider under the given key.
// Useful for dynamic reconfiguration or tests.
func (r *Router) Register(key string, p LLMProvider) {
	r.providers[key] = p
}

// Route returns the provider for the current request.
// MVP implementation: always returns providers[defaultProvider].
// Returns an error if the default provider is not registered.
func (r *Router) Route(_ context.Context) (LLMProvider, error) {
	p, ok := r.providers[r.defaultProvider]
	if !ok {
		return nil, fmt.Errorf("llm router: provider %q not registered (available: %v)", r.defaultProvider, r.keys())
	}
	return p, nil
}

// keys returns the registered provider names (for error messages).
func (r *Router) keys() []string {
	out := make([]string, 0, len(r.providers))
	for k := range r.providers {
		out = append(out, k)
	}
	return out
}
