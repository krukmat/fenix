package agent

import (
	"errors"
	"strings"
	"sync"
)

var (
	ErrRunnerNotFound      = errors.New("runner not found")
	ErrRunnerAlreadyExists = errors.New("runner already registered")
	ErrRunnerNil           = errors.New("runner is nil")
	ErrRunnerTypeEmpty     = errors.New("agent type is required")
)

// RunnerRegistry maps an agent_type to a concrete AgentRunner.
//
// It stays intentionally small in F1.3 so orchestrator wiring, Go agent
// adapters, and declarative runners can build on a stable lookup contract.
type RunnerRegistry struct {
	mu      sync.RWMutex
	runners map[string]AgentRunner
}

// NewRunnerRegistry creates an empty runner registry.
func NewRunnerRegistry() *RunnerRegistry {
	return &RunnerRegistry{
		runners: make(map[string]AgentRunner),
	}
}

// Register stores a runner under a normalized agent type.
func (r *RunnerRegistry) Register(agentType string, runner AgentRunner) error {
	if runner == nil {
		return ErrRunnerNil
	}

	key, err := normalizeAgentType(agentType)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.runners[key]; exists {
		return ErrRunnerAlreadyExists
	}

	r.runners[key] = runner
	return nil
}

// Get returns the registered runner for an agent type.
func (r *RunnerRegistry) Get(agentType string) (AgentRunner, bool) {
	key, err := normalizeAgentType(agentType)
	if err != nil {
		return nil, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	runner, ok := r.runners[key]
	return runner, ok
}

// Resolve returns a registered runner or a deterministic lookup error.
func (r *RunnerRegistry) Resolve(agentType string) (AgentRunner, error) {
	runner, ok := r.Get(agentType)
	if !ok {
		return nil, ErrRunnerNotFound
	}
	return runner, nil
}

func normalizeAgentType(agentType string) (string, error) {
	key := strings.TrimSpace(agentType)
	if key == "" {
		return "", ErrRunnerTypeEmpty
	}
	return key, nil
}
