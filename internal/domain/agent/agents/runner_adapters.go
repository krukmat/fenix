package agents

import (
	"context"
	"encoding/json"

	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
)

// SupportRunner adapts SupportAgent to the AgentRunner contract without
// changing the typed public API of the agent.
type SupportRunner struct {
	Agent *SupportAgent
}

// ProspectingRunner adapts ProspectingAgent to the AgentRunner contract.
type ProspectingRunner struct {
	Agent *ProspectingAgent
}

// KBRunner adapts KBAgent to the AgentRunner contract.
type KBRunner struct {
	Agent *KBAgent
}

// InsightsRunner adapts InsightsAgent to the AgentRunner contract.
type InsightsRunner struct {
	Agent *InsightsAgent
}

var (
	_ agent.AgentRunner = (*SupportRunner)(nil)
	_ agent.AgentRunner = (*ProspectingRunner)(nil)
	_ agent.AgentRunner = (*KBRunner)(nil)
	_ agent.AgentRunner = (*InsightsRunner)(nil)
)

func (r *SupportRunner) Run(ctx context.Context, rc *agent.RunContext, input agent.TriggerAgentInput) (*agent.Run, error) {
	_ = rc
	cfg, err := decodeSupportAgentInput(input)
	if err != nil {
		return nil, err
	}
	return r.Agent.Run(ctx, cfg)
}

func (r *ProspectingRunner) Run(ctx context.Context, rc *agent.RunContext, input agent.TriggerAgentInput) (*agent.Run, error) {
	_ = rc
	cfg, err := decodeProspectingAgentInput(input)
	if err != nil {
		return nil, err
	}
	return r.Agent.Run(ctx, cfg)
}

func (r *KBRunner) Run(ctx context.Context, rc *agent.RunContext, input agent.TriggerAgentInput) (*agent.Run, error) {
	_ = rc
	cfg, err := decodeKBAgentInput(input)
	if err != nil {
		return nil, err
	}
	return r.Agent.Run(ctx, cfg)
}

func (r *InsightsRunner) Run(ctx context.Context, rc *agent.RunContext, input agent.TriggerAgentInput) (*agent.Run, error) {
	_ = rc
	cfg, err := decodeInsightsAgentInput(input)
	if err != nil {
		return nil, err
	}
	return r.Agent.Run(ctx, cfg)
}

func decodeSupportAgentInput(input agent.TriggerAgentInput) (SupportAgentConfig, error) {
	cfg := SupportAgentConfig{WorkspaceID: input.WorkspaceID}
	if len(input.Inputs) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(input.Inputs, &cfg); err != nil {
		return SupportAgentConfig{}, err
	}
	if cfg.WorkspaceID == "" {
		cfg.WorkspaceID = input.WorkspaceID
	}
	return cfg, nil
}

func decodeProspectingAgentInput(input agent.TriggerAgentInput) (ProspectingAgentConfig, error) {
	cfg := ProspectingAgentConfig{
		WorkspaceID:       input.WorkspaceID,
		TriggeredByUserID: input.TriggeredBy,
	}
	if len(input.Inputs) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(input.Inputs, &cfg); err != nil {
		return ProspectingAgentConfig{}, err
	}
	if cfg.WorkspaceID == "" {
		cfg.WorkspaceID = input.WorkspaceID
	}
	if cfg.TriggeredByUserID == nil {
		cfg.TriggeredByUserID = input.TriggeredBy
	}
	return cfg, nil
}

func decodeKBAgentInput(input agent.TriggerAgentInput) (KBAgentConfig, error) {
	cfg := KBAgentConfig{
		WorkspaceID:       input.WorkspaceID,
		TriggeredByUserID: input.TriggeredBy,
	}
	if len(input.Inputs) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(input.Inputs, &cfg); err != nil {
		return KBAgentConfig{}, err
	}
	if cfg.WorkspaceID == "" {
		cfg.WorkspaceID = input.WorkspaceID
	}
	if cfg.TriggeredByUserID == nil {
		cfg.TriggeredByUserID = input.TriggeredBy
	}
	return cfg, nil
}

func decodeInsightsAgentInput(input agent.TriggerAgentInput) (InsightsAgentConfig, error) {
	cfg := InsightsAgentConfig{
		WorkspaceID:       input.WorkspaceID,
		TriggeredByUserID: input.TriggeredBy,
	}
	if len(input.Inputs) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(input.Inputs, &cfg); err != nil {
		return InsightsAgentConfig{}, err
	}
	if cfg.WorkspaceID == "" {
		cfg.WorkspaceID = input.WorkspaceID
	}
	if cfg.TriggeredByUserID == nil {
		cfg.TriggeredByUserID = input.TriggeredBy
	}
	return cfg, nil
}
