package agents

import (
	"errors"

	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
)

const (
	AgentTypeSupport     = "support"
	AgentTypeProspecting = "prospecting"
	AgentTypeKB          = "kb"
	AgentTypeInsights    = "insights"
)

var (
	ErrRunnerRegistryNil = errors.New("runner registry is nil")
	ErrGoAgentNil        = errors.New("go agent is nil")
)

// GoAgentRunners holds the explicit Go agent adapters registered in F1.6.
type GoAgentRunners struct {
	Support     *SupportAgent
	Prospecting *ProspectingAgent
	KB          *KBAgent
	Insights    *InsightsAgent
}

// RegisterCurrentGoRunners registers the current Go agent implementations into
// a RunnerRegistry using explicit agent_type mappings.
func RegisterCurrentGoRunners(registry *agent.RunnerRegistry, runners GoAgentRunners) error {
	if registry == nil {
		return ErrRunnerRegistryNil
	}
	if runners.Support == nil || runners.Prospecting == nil || runners.KB == nil || runners.Insights == nil {
		return ErrGoAgentNil
	}

	if err := registry.Register(AgentTypeSupport, &SupportRunner{Agent: runners.Support}); err != nil {
		return err
	}
	if err := registry.Register(AgentTypeProspecting, &ProspectingRunner{Agent: runners.Prospecting}); err != nil {
		return err
	}
	if err := registry.Register(AgentTypeKB, &KBRunner{Agent: runners.KB}); err != nil {
		return err
	}
	if err := registry.Register(AgentTypeInsights, &InsightsRunner{Agent: runners.Insights}); err != nil {
		return err
	}

	return nil
}
