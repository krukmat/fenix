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
	if err := validateGoAgentRunners(runners); err != nil {
		return err
	}
	entries := []struct {
		agentType string
		runner    agent.Runner
	}{
		{AgentTypeSupport, &SupportRunner{Agent: runners.Support}},
		{AgentTypeProspecting, &ProspectingRunner{Agent: runners.Prospecting}},
		{AgentTypeKB, &KBRunner{Agent: runners.KB}},
		{AgentTypeInsights, &InsightsRunner{Agent: runners.Insights}},
	}
	for _, e := range entries {
		if err := registry.Register(e.agentType, e.runner); err != nil {
			return err
		}
	}
	return nil
}

func validateGoAgentRunners(runners GoAgentRunners) error {
	if runners.Support == nil || runners.Prospecting == nil || runners.KB == nil || runners.Insights == nil {
		return ErrGoAgentNil
	}
	return nil
}
