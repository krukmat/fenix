package blackboard

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

const (
	defaultPlanningMemoryKey = "planning/last_collaborative_plan"
	defaultMinReadyScore     = 0.6

	signalArtifactMemoryKey   = "specialized_agents/blackboard-signal-agent/last_artifact"
	evidenceArtifactMemoryKey = "specialized_agents/blackboard-evidence-agent/last_artifact"
	policyArtifactMemoryKey   = "specialized_agents/blackboard-policy-agent/last_artifact"
)

// ErrArbitrationResultNotFound is returned when collaborative planning is requested
// before any arbitration result has been persisted for the workspace.
var ErrArbitrationResultNotFound = errors.New("blackboard arbitration result not found")

// Planner builds policy-aware tool-sequence proposals from arbitration output.
type Planner interface {
	BuildWorkspacePlan(ctx context.Context, cognitiveWorkspaceID string, config PlanningConfig) (*CollaborativePlanningResult, error)
}

type sqlitePlanner struct {
	memory MemoryStore
}

// NewPlanner returns a Planner backed by the shared blackboard memory store.
func NewPlanner(db *sql.DB) Planner {
	return &sqlitePlanner{memory: NewMemoryStore(db)}
}

type planningArtifact struct {
	Contributor  string `json:"contributor"`
	ArtifactType string `json:"artifact_type"`
	Summary      string `json:"summary"`
}

func (p *sqlitePlanner) BuildWorkspacePlan(ctx context.Context, cognitiveWorkspaceID string, config PlanningConfig) (*CollaborativePlanningResult, error) {
	cfg := normalizePlanningConfig(config)

	arbitration, err := p.loadArbitrationResult(ctx, cognitiveWorkspaceID, cfg.ArbitrationMemoryKey)
	if err != nil {
		return nil, err
	}

	evidence, err := p.loadOptionalArtifact(ctx, cognitiveWorkspaceID, evidenceArtifactMemoryKey)
	if err != nil {
		return nil, err
	}
	policy, err := p.loadOptionalArtifact(ctx, cognitiveWorkspaceID, policyArtifactMemoryKey)
	if err != nil {
		return nil, err
	}
	signal, err := p.loadOptionalArtifact(ctx, cognitiveWorkspaceID, signalArtifactMemoryKey)
	if err != nil {
		return nil, err
	}

	result := BuildCollaborativePlan(arbitration, signal, evidence, policy, cfg)
	if cfg.PersistResult {
		persistErr := p.persistResult(ctx, result, cfg)
		if persistErr != nil {
			return nil, persistErr
		}
	}
	return result, nil
}

// BuildCollaborativePlan synthesizes deterministic proposals from ranked hypotheses and
// the latest specialized-agent artifacts.
func BuildCollaborativePlan(
	arbitration *ArbitrationResult,
	signal, evidence, policy *planningArtifact,
	config PlanningConfig,
) *CollaborativePlanningResult {
	cfg := normalizePlanningConfig(config)
	if arbitration == nil {
		return &CollaborativePlanningResult{
			GeneratedAt: cfg.Now,
			State:       PlanningStateNoAction,
			Proposals:   []CollaborativePlanProposal{},
		}
	}

	proposals := make([]CollaborativePlanProposal, 0, len(arbitration.Ranked))
	for _, ranked := range arbitration.Ranked {
		state := resolvePlanningState(ranked.Score, evidence != nil, policy != nil, cfg.MinReadyScore)
		proposal := CollaborativePlanProposal{
			ProposalID:     fmt.Sprintf("proposal-%s", ranked.Hypothesis.ID),
			HypothesisID:   ranked.Hypothesis.ID,
			HypothesisRank: ranked.Rank,
			Summary:        proposalSummary(ranked, signal, evidence, policy),
			Score:          ranked.Score,
			State:          state,
			Constraints:    buildConstraints(ranked.Score, evidence, policy, cfg.MinReadyScore),
			Contributors:   buildContributors(signal, evidence, policy),
			Steps:          buildSteps(ranked, evidence, policy),
		}
		proposals = append(proposals, proposal)
	}

	result := &CollaborativePlanningResult{
		CognitiveWorkspaceID: arbitration.CognitiveWorkspaceID,
		GeneratedAt:          cfg.Now,
		State:                PlanningStateNoAction,
		Proposals:            proposals,
	}
	if len(proposals) == 0 {
		return result
	}

	selected := proposals[0]
	result.SelectedProposal = &selected
	result.State = selected.State
	return result
}

func (p *sqlitePlanner) loadArbitrationResult(ctx context.Context, cognitiveWorkspaceID, key string) (*ArbitrationResult, error) {
	entry, err := p.memory.Get(ctx, cognitiveWorkspaceID, key)
	if errors.Is(err, ErrMemoryNotFound) {
		return nil, ErrArbitrationResultNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("planner load arbitration result: %w", err)
	}

	var result ArbitrationResult
	if unmarshalErr := json.Unmarshal(entry.Value, &result); unmarshalErr != nil {
		return nil, fmt.Errorf("planner unmarshal arbitration result: %w", unmarshalErr)
	}
	return &result, nil
}

func (p *sqlitePlanner) loadOptionalArtifact(ctx context.Context, cognitiveWorkspaceID, key string) (*planningArtifact, error) {
	entry, err := p.memory.Get(ctx, cognitiveWorkspaceID, key)
	if errors.Is(err, ErrMemoryNotFound) || errors.Is(err, ErrMemoryExpired) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("planner load artifact %s: %w", key, err)
	}

	var artifact planningArtifact
	if unmarshalErr := json.Unmarshal(entry.Value, &artifact); unmarshalErr != nil {
		return nil, fmt.Errorf("planner unmarshal artifact %s: %w", key, unmarshalErr)
	}
	return &artifact, nil
}

func (p *sqlitePlanner) persistResult(ctx context.Context, result *CollaborativePlanningResult, config PlanningConfig) error {
	raw, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("planner marshal result: %w", err)
	}

	setErr := p.memory.Set(ctx, AgentMemory{
		ID:                   uuid.NewV7().String(),
		CognitiveWorkspaceID: result.CognitiveWorkspaceID,
		Key:                  config.ResultMemoryKey,
		Value:                raw,
		Scope:                MemoryScopeSession,
		CreatedAt:            config.Now,
		UpdatedAt:            config.Now,
	})
	if setErr != nil {
		return fmt.Errorf("planner persist result: %w", setErr)
	}
	return nil
}

func normalizePlanningConfig(config PlanningConfig) PlanningConfig {
	cfg := config
	if cfg.Now.IsZero() {
		cfg.Now = time.Now().UTC()
	} else {
		cfg.Now = cfg.Now.UTC()
	}
	if cfg.ArbitrationMemoryKey == "" {
		cfg.ArbitrationMemoryKey = DefaultArbitrationMemoryKey
	}
	if cfg.ResultMemoryKey == "" {
		cfg.ResultMemoryKey = defaultPlanningMemoryKey
	}
	if cfg.MinReadyScore <= 0 {
		cfg.MinReadyScore = defaultMinReadyScore
	}
	if !cfg.PersistResult {
		cfg.PersistResult = true
	}
	return cfg
}

func resolvePlanningState(score float64, hasEvidence, hasPolicy bool, minReadyScore float64) PlanningState {
	switch {
	case !hasEvidence:
		return PlanningStateAwaitingEvidence
	case hasPolicy:
		return PlanningStatePendingApproval
	case score < minReadyScore:
		return PlanningStateNeedsReview
	default:
		return PlanningStateReady
	}
}

func proposalSummary(ranked RankedHypothesis, signal, evidence, policy *planningArtifact) string {
	parts := []string{
		fmt.Sprintf("Plan for ranked hypothesis %d", ranked.Rank),
		ranked.Hypothesis.Content,
	}
	if signal != nil && signal.Summary != "" {
		parts = append(parts, signal.Summary)
	}
	if evidence != nil && evidence.Summary != "" {
		parts = append(parts, evidence.Summary)
	}
	if policy != nil && policy.Summary != "" {
		parts = append(parts, policy.Summary)
	}
	return strings.Join(parts, " | ")
}

func buildConstraints(score float64, evidence, policy *planningArtifact, minReadyScore float64) []string {
	constraints := []string{}
	if evidence == nil {
		constraints = append(constraints, "evidence artifact is missing")
	}
	if policy != nil && policy.Summary != "" {
		constraints = append(constraints, policy.Summary)
	}
	if score < minReadyScore {
		constraints = append(constraints, fmt.Sprintf("score %.4f is below ready threshold %.4f", score, minReadyScore))
	}
	return constraints
}

func buildContributors(signal, evidence, policy *planningArtifact) []string {
	contributors := []string{}
	for _, artifact := range []*planningArtifact{signal, evidence, policy} {
		if artifact == nil || artifact.Contributor == "" {
			continue
		}
		contributors = append(contributors, artifact.Contributor)
	}
	return contributors
}

func buildSteps(ranked RankedHypothesis, evidence, policy *planningArtifact) []ToolSequenceStep {
	steps := []ToolSequenceStep{
		{
			Sequence: 1,
			ToolName: "review_hypothesis",
			Reason:   ranked.Hypothesis.Content,
		},
	}

	next := 2
	if evidence != nil {
		steps = append(steps, ToolSequenceStep{
			Sequence: next,
			ToolName: "validate_evidence",
			Reason:   evidence.Summary,
		})
		next++
	}
	if policy != nil {
		steps = append(steps, ToolSequenceStep{
			Sequence:         next,
			ToolName:         "request_policy_approval",
			Reason:           policy.Summary,
			RequiresApproval: true,
		})
		next++
	}
	steps = append(steps, ToolSequenceStep{
		Sequence: next,
		ToolName: "execute_governed_action",
		Reason:   "Execute the highest-ranked governed action when all constraints are satisfied.",
	})
	return steps
}
