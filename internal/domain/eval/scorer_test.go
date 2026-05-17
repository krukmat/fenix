package eval

import (
	"encoding/json"
	"testing"
	"time"
)

func TestScoringService_Score_ProducesScorecard(t *testing.T) {
	t.Parallel()

	service := NewScoringService()
	scenario := makeScoringScenario()
	artifact := makeReplayArtifactForScoring()

	scored := service.Score(artifact, scenario)

	if scored.Scorecard.TotalScore <= 0 {
		t.Fatalf("expected positive total score, got %f", scored.Scorecard.TotalScore)
	}
	if scored.ScenarioID != scenario.ID {
		t.Fatalf("ScenarioID = %q; want %q", scored.ScenarioID, scenario.ID)
	}
	if scored.HardGateAssessment.Scorecard.TotalScore != scored.Scorecard.TotalScore {
		t.Fatal("expected hard gate assessment to embed the same scorecard")
	}
}

func TestScoringService_Score_HardGateFails_WhenForbiddenTool(t *testing.T) {
	t.Parallel()

	service := NewScoringService()
	scenario := makeScoringScenario()
	scenario.Expected.ForbiddenToolCalls = []ForbiddenToolCall{{
		ToolName: "send_email",
		Reason:   "external outreach forbidden",
	}}

	artifact := makeReplayArtifactForScoring()
	artifact.ToolCalls = append(artifact.ToolCalls, TraceToolCall{
		ToolName: "send_email",
		Status:   traceStatusExecuted,
	})

	scored := service.Score(artifact, scenario)

	if scored.HardGateAssessment.FinalVerdict != VerdictFailedValidation {
		t.Fatalf("FinalVerdict = %q; want %q", scored.HardGateAssessment.FinalVerdict, VerdictFailedValidation)
	}
	if scored.Pass {
		t.Fatal("expected scoring result to fail when forbidden tool executes")
	}
}

func TestScoringService_Score_Deterministic(t *testing.T) {
	t.Parallel()

	service := NewScoringService()
	scenario := makeScoringScenario()
	artifact := makeReplayArtifactForScoring()

	first := service.Score(artifact, scenario)
	second := service.Score(artifact, scenario)

	raw1, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("Marshal(first): %v", err)
	}
	raw2, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("Marshal(second): %v", err)
	}
	if string(raw1) != string(raw2) {
		t.Fatalf("Score() is not deterministic:\nfirst=%s\nsecond=%s", raw1, raw2)
	}
}

func makeScoringScenario() GoldenScenario {
	return GoldenScenario{
		ID:          "scoring-scenario-1",
		Title:       "Replay scoring scenario",
		Description: "Deterministic scoring path",
		Domain:      "support",
		InputEvent: ScenarioInputEvent{
			Type:    "case.updated",
			Payload: map[string]any{"caseId": "case-1"},
		},
		Expected: ScenarioExpected{
			FinalOutcome:     "success",
			RequiredEvidence: []string{"ev-1"},
			ToolCalls: []ExpectedToolCall{{
				ToolName: "crm.case.get",
				Required: true,
			}},
			AuditEvents: []string{"tool.executed"},
			FinalState: map[string]any{
				"case.status": "Closed",
			},
		},
		Thresholds: ScenarioThresholds{
			MinScore:     80,
			MaxLatencyMs: 5_000,
			MaxToolCalls: 2,
			MaxRetries:   1,
		},
	}
}

func makeReplayArtifactForScoring() ReplayArtifact {
	startedAt := time.Unix(1_700_000_000, 0).UTC()
	completedAt := startedAt.Add(2 * time.Second)
	finalState, _ := json.Marshal(map[string]any{"case.status": "Closed"})

	return ReplayArtifact{
		Request: ReplayRequest{
			EvalRunID:   "eval-run-1",
			WorkspaceID: "ws-1",
		},
		Source: ReplaySource{
			SourceRun: &ReplaySourceRun{
				RunID:                "agent-run-1",
				WorkspaceID:          "ws-1",
				AgentDefinitionID:    "agent-def-1",
				TriggerType:          "manual",
				Status:               "success",
				TriggerContext:       json.RawMessage(`{"type":"case.updated"}`),
				Inputs:               json.RawMessage(`{"caseId":"case-1"}`),
				RetrievedEvidenceIDs: []string{"ev-1"},
				ReasoningTrace:       json.RawMessage(`{"steps":["observe","act"]}`),
				Output:               finalState,
				StartedAt:            startedAt,
				CompletedAt:          &completedAt,
			},
		},
		Input: ReplayInput{
			InputEvent:      json.RawMessage(`{"type":"case.updated"}`),
			ContextInputs:   json.RawMessage(`{"caseId":"case-1"}`),
			EvidenceSources: []string{"ev-1"},
		},
		FinalOutcome:    "success",
		Output:          finalState,
		EvidenceSources: []string{"ev-1"},
		ToolCalls: []TraceToolCall{{
			ToolName: "crm.case.get",
			Status:   traceStatusExecuted,
			Params:   json.RawMessage(`{"id":"case-1"}`),
		}},
		AuditEvents: []TraceAuditEvent{{
			ID:      "audit-1",
			Action:  "tool.executed",
			Outcome: "success",
			ActorID: "agent-1",
			At:      startedAt.Add(time.Second),
		}},
		BuiltAt: completedAt,
	}
}
