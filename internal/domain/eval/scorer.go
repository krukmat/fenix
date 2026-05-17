package eval

import "encoding/json"

// ScoredResult is the deterministic scoring projection persisted in eval_run.scores.
type ScoredResult struct {
	ScenarioID         string             `json:"scenario_id,omitempty"`
	Scorecard          Scorecard          `json:"scorecard"`
	HardGateAssessment HardGateAssessment `json:"hard_gate_assessment"`
	Pass               bool               `json:"pass"`
}

// ScoringService orchestrates deterministic scoring over replay artifacts.
type ScoringService struct {
	Weights ScorecardWeights
}

// NewScoringService constructs a ScoringService with default scorecard weights.
func NewScoringService() *ScoringService {
	return &ScoringService{}
}

// Score computes a deterministic scorecard and hard-gate assessment for one replay.
func (s ScoringService) Score(artifact ReplayArtifact, scenario GoldenScenario) ScoredResult {
	trace := buildActualRunTrace(artifact, scenario)
	result := Compare(scenario, trace)
	metrics := ComputeMetrics(scenario, trace, result)
	scorecard := NewScorecard(metrics, s.scorecardWeights())
	violations := EvaluateHardGates(scenario, trace, result)
	assessment := ApplyHardGates(scorecard, violations)

	return ScoredResult{
		ScenarioID:         scenario.ID,
		Scorecard:          scorecard,
		HardGateAssessment: assessment,
		Pass:               result.Pass && assessment.FinalVerdict != VerdictFailedValidation,
	}
}

func (s ScoringService) scorecardWeights() ScorecardWeights {
	if s.Weights == (ScorecardWeights{}) {
		return DefaultScorecardWeights()
	}
	return s.Weights
}

func buildActualRunTrace(artifact ReplayArtifact, scenario GoldenScenario) ActualRunTrace {
	trace := ActualRunTrace{
		RunID:           artifact.Request.EvalRunID,
		WorkspaceID:     artifact.Request.WorkspaceID,
		ScenarioID:      scenario.ID,
		InputEvent:      cloneRawMessage(firstNonEmptyRawMessage(artifact.Input.InputEvent)),
		ContextInputs:   cloneRawMessage(firstNonEmptyRawMessage(artifact.Input.ContextInputs)),
		EvidenceSources: append([]string(nil), artifact.EvidenceSources...),
		PolicyDecisions: append([]TracePolicyDecision(nil), artifact.PolicyDecisions...),
		ApprovalEvents:  append([]TraceApprovalEvent(nil), artifact.ApprovalEvents...),
		ToolCalls:       append([]TraceToolCall(nil), artifact.ToolCalls...),
		AuditEvents:     append([]TraceAuditEvent(nil), artifact.AuditEvents...),
		FinalOutcome:    artifact.FinalOutcome,
		Output:          cloneRawMessage(artifact.Output),
		StartedAt:       artifact.BuiltAt,
		CompletedAt:     nil,
	}

	if trace.FinalOutcome == "" && artifact.Source.SourceRun != nil {
		trace.FinalOutcome = artifact.Source.SourceRun.Status
	}

	if artifact.Source.SourceRun != nil {
		run := artifact.Source.SourceRun
		trace.RunID = run.RunID
		trace.WorkspaceID = run.WorkspaceID
		trace.AgentDefinitionID = run.AgentDefinitionID
		trace.TriggerType = run.TriggerType
		trace.InputEvent = cloneRawMessage(firstNonEmptyRawMessage(trace.InputEvent, run.TriggerContext))
		trace.ContextInputs = cloneRawMessage(firstNonEmptyRawMessage(trace.ContextInputs, run.Inputs))
		trace.EvidenceSources = firstNonEmptyStrings(trace.EvidenceSources, run.RetrievedEvidenceIDs)
		trace.FinalOutcome = firstNonEmptyString(trace.FinalOutcome, run.Status)
		trace.Output = cloneRawMessage(firstNonEmptyRawMessage(trace.Output, run.Output))
		trace.AbstentionReason = run.AbstentionReason
		trace.ReasoningTrace = cloneRawMessage(run.ReasoningTrace)
		trace.StartedAt = run.StartedAt
		trace.CompletedAt = run.CompletedAt
		if run.CompletedAt != nil {
			latencyMs := run.CompletedAt.Sub(run.StartedAt).Milliseconds()
			trace.LatencyMs = &latencyMs
		}
	}

	trace.ContractValidation = validateContract(&trace)
	return trace
}

func firstNonEmptyRawMessage(values ...json.RawMessage) json.RawMessage {
	for _, value := range values {
		if len(value) > 0 {
			return value
		}
	}
	return nil
}

func firstNonEmptyStrings(primary []string, fallback []string) []string {
	if len(primary) > 0 {
		return append([]string(nil), primary...)
	}
	return append([]string(nil), fallback...)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func cloneRawMessage(in json.RawMessage) json.RawMessage {
	if len(in) == 0 {
		return nil
	}
	out := make(json.RawMessage, len(in))
	copy(out, in)
	return out
}
