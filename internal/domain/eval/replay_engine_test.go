package eval_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/eval"
)

func TestReplayErrors_AreMatchable(t *testing.T) {
	t.Parallel()

	err := &eval.ReplaySourceError{
		Kind:        eval.ReplaySourceErrorTrace,
		WorkspaceID: "ws-1",
		EvalRunID:   "eval-run-1",
		SourceID:    "trace-1",
	}

	if !errors.Is(err, eval.ErrReplayTraceMissing) {
		t.Fatal("expected ReplaySourceError to match ErrReplayTraceMissing")
	}
	if errors.Is(err, eval.ErrReplayTimelineMissing) {
		t.Fatal("did not expect ReplaySourceError(trace) to match ErrReplayTimelineMissing")
	}
}

func TestReplayDTOs_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	traceID := "trace-1"
	cwID := "cw-1"
	actorID := "agent-1"
	abstentionReason := "not enough evidence"

	request := eval.ReplayRequest{
		EvalRunID:   "eval-run-1",
		WorkspaceID: "ws-1",
		Provenance: &eval.ReplayProvenance{
			Mode:                       eval.ReplayModeReplay,
			SourceAgentRunID:           replayStringPtr("agent-run-1"),
			SourceCognitiveWorkspaceID: &cwID,
			SourceTraceID:              &traceID,
		},
		ScenarioID:  "scenario-1",
		RequestedBy: "user-1",
		RequestedAt: now,
	}
	source := eval.ReplaySource{
		Provenance: *request.Provenance,
		SourceRun: &eval.ReplaySourceRun{
			RunID:                "agent-run-1",
			WorkspaceID:          "ws-1",
			AgentDefinitionID:    "agent-def-1",
			TriggerType:          "manual",
			Status:               "success",
			TraceID:              &traceID,
			CognitiveWorkspaceID: &cwID,
			TriggerContext:       json.RawMessage(`{"type":"case.updated"}`),
			Inputs:               json.RawMessage(`{"caseId":"case-1"}`),
			RetrievedEvidenceIDs: []string{"ev-1", "ev-2"},
			ReasoningTrace:       json.RawMessage(`{"steps":["observe","act"]}`),
			ToolCalls: []eval.TraceToolCall{{
				ToolName: "crm.case.get",
				Status:   "executed",
				Params:   json.RawMessage(`{"id":"case-1"}`),
			}},
			Output:           json.RawMessage(`{"summary":"done"}`),
			AbstentionReason: &abstentionReason,
			StartedAt:        now,
			CompletedAt:      replayTimePtr(now.Add(time.Second)),
		},
		ReasoningEvents: []eval.ReplayReasoningEvent{{
			ID:                   "evt-1",
			CognitiveWorkspaceID: cwID,
			ActorAgentID:         &actorID,
			EventType:            "observation",
			Payload:              json.RawMessage(`{"note":"hello"}`),
			CreatedAt:            now,
		}},
		AuditEvents: []eval.TraceAuditEvent{{
			ID:      "audit-1",
			Action:  "policy.decision",
			Outcome: "success",
			ActorID: actorID,
			Details: json.RawMessage(`{"action":"tool:crm.case.get","outcome":"allow"}`),
			At:      now,
		}},
	}
	input := eval.ReplayInput{
		Request:         request,
		Source:          source,
		ContextInputs:   json.RawMessage(`{"caseId":"case-1"}`),
		InputEvent:      json.RawMessage(`{"type":"case.updated"}`),
		EvidenceSources: []string{"ev-1", "ev-2"},
		ToolCalls:       source.SourceRun.ToolCalls,
		PolicyDecisions: []eval.TracePolicyDecision{{Action: "tool:crm.case.get", Outcome: "allow"}},
		ApprovalEvents:  []eval.TraceApprovalEvent{{ApprovalID: "ap-1", Action: "case.update", Status: "approved", DecidedAt: replayTimePtr(now)}},
	}
	artifact := eval.ReplayArtifact{
		Request:         request,
		Source:          source,
		Input:           input,
		FinalOutcome:    "success",
		Output:          json.RawMessage(`{"summary":"done"}`),
		ReasoningEvents: source.ReasoningEvents,
		ToolCalls:       input.ToolCalls,
		EvidenceSources: input.EvidenceSources,
		PolicyDecisions: input.PolicyDecisions,
		ApprovalEvents:  input.ApprovalEvents,
		AuditEvents:     source.AuditEvents,
		BuiltAt:         now.Add(2 * time.Second),
	}

	raw, err := json.Marshal(artifact)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var roundTrip eval.ReplayArtifact
	if err := json.Unmarshal(raw, &roundTrip); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if roundTrip.Request.EvalRunID != artifact.Request.EvalRunID {
		t.Fatalf("EvalRunID mismatch: got %q want %q", roundTrip.Request.EvalRunID, artifact.Request.EvalRunID)
	}
	if got := replayDeref(roundTrip.Source.SourceRun.TraceID); got != traceID {
		t.Fatalf("TraceID mismatch: got %q want %q", got, traceID)
	}
	if len(roundTrip.ReasoningEvents) != 1 || roundTrip.ReasoningEvents[0].EventType != "observation" {
		t.Fatalf("ReasoningEvents mismatch: %#v", roundTrip.ReasoningEvents)
	}
	if len(roundTrip.PolicyDecisions) != 1 || roundTrip.PolicyDecisions[0].Outcome != "allow" {
		t.Fatalf("PolicyDecisions mismatch: %#v", roundTrip.PolicyDecisions)
	}
}

func TestReplayLoader_LoadsSourceAgentRun(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	wsID := mustCreateWorkspace(t, db, "replay-run")
	mustInsertAgentDefinition(t, db, wsID, "agent-def-1")
	mustInsertAgentRun(t, db, wsID, "agent-run-1", "agent-def-1", nil, nil)
	mustInsertEvalSuite(t, db, wsID, "suite-1")
	mustInsertEvalRunWithProvenance(t, db, wsID, "eval-run-1", "suite-1", replayStringPtr("agent-run-1"), nil, nil)

	engine := eval.NewSQLiteReplayEngine(db)
	source, err := engine.LoadSource(context.Background(), eval.ReplayRequest{
		EvalRunID:   "eval-run-1",
		WorkspaceID: wsID,
	})
	if err != nil {
		t.Fatalf("LoadSource: %v", err)
	}
	if source.SourceRun == nil {
		t.Fatal("expected SourceRun to be loaded")
	}
	if source.SourceRun.RunID != "agent-run-1" {
		t.Fatalf("SourceRun.RunID = %q; want %q", source.SourceRun.RunID, "agent-run-1")
	}
}

func TestReplayLoader_LoadsReasoningTimelineInOrder(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	wsID := mustCreateWorkspace(t, db, "replay-timeline")
	mustInsertCognitiveWorkspace(t, db, wsID, "cw-1")
	base := time.Unix(1_700_000_100, 0).UTC()
	mustInsertReasoningEvent(t, db, "evt-2", "cw-1", "observation", `{"n":2}`, base.Add(time.Second))
	mustInsertReasoningEvent(t, db, "evt-1", "cw-1", "hypothesis", `{"n":1}`, base)
	mustInsertEvalSuite(t, db, wsID, "suite-1")
	mustInsertEvalRunWithProvenance(t, db, wsID, "eval-run-1", "suite-1", nil, replayStringPtr("cw-1"), nil)

	engine := eval.NewSQLiteReplayEngine(db)
	source, err := engine.LoadSource(context.Background(), eval.ReplayRequest{
		EvalRunID:   "eval-run-1",
		WorkspaceID: wsID,
	})
	if err != nil {
		t.Fatalf("LoadSource: %v", err)
	}
	if len(source.ReasoningEvents) != 2 {
		t.Fatalf("ReasoningEvents len = %d; want 2", len(source.ReasoningEvents))
	}
	if source.ReasoningEvents[0].ID != "evt-1" || source.ReasoningEvents[1].ID != "evt-2" {
		t.Fatalf("ReasoningEvents order = [%s %s]; want [evt-1 evt-2]",
			source.ReasoningEvents[0].ID, source.ReasoningEvents[1].ID)
	}
}

func TestReplayLoader_LoadsAuditTraceInOrder(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	wsID := mustCreateWorkspace(t, db, "replay-trace")
	mustInsertEvalSuite(t, db, wsID, "suite-1")
	traceID := "trace-1"
	base := time.Unix(1_700_000_200, 0).UTC()
	mustInsertAuditEvent(t, db, wsID, "audit-2", traceID, "tool.executed", base.Add(time.Second))
	mustInsertAuditEvent(t, db, wsID, "audit-1", traceID, "policy.decision", base)
	mustInsertEvalRunWithProvenance(t, db, wsID, "eval-run-1", "suite-1", nil, nil, &traceID)

	engine := eval.NewSQLiteReplayEngine(db)
	source, err := engine.LoadSource(context.Background(), eval.ReplayRequest{
		EvalRunID:   "eval-run-1",
		WorkspaceID: wsID,
	})
	if err != nil {
		t.Fatalf("LoadSource: %v", err)
	}
	if len(source.AuditEvents) != 2 {
		t.Fatalf("AuditEvents len = %d; want 2", len(source.AuditEvents))
	}
	if source.AuditEvents[0].ID != "audit-1" || source.AuditEvents[1].ID != "audit-2" {
		t.Fatalf("AuditEvents order = [%s %s]; want [audit-1 audit-2]",
			source.AuditEvents[0].ID, source.AuditEvents[1].ID)
	}
}

func TestReplayLoader_MissingRequiredSourceFailsExplicitly(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	wsID := mustCreateWorkspace(t, db, "replay-missing")
	mustInsertEvalSuite(t, db, wsID, "suite-1")
	traceID := "trace-missing"
	mustInsertEvalRunWithProvenance(t, db, wsID, "eval-run-1", "suite-1", nil, nil, &traceID)

	engine := eval.NewSQLiteReplayEngine(db)
	_, err := engine.LoadSource(context.Background(), eval.ReplayRequest{
		EvalRunID:   "eval-run-1",
		WorkspaceID: wsID,
	})
	if err == nil {
		t.Fatal("expected LoadSource to fail")
	}
	if !errors.Is(err, eval.ErrReplayTraceMissing) {
		t.Fatalf("expected ErrReplayTraceMissing, got %v", err)
	}
}

// TestReplayEngine_BuildsReplayInputFromSourceAggregate asserts that BuildReplay
// assembles a ReplayInput populated from the persisted source aggregate. (task-C2.3)
func TestReplayEngine_BuildsReplayInputFromSourceAggregate(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	wsID := mustCreateWorkspace(t, db, "build-input")
	mustInsertAgentDefinition(t, db, wsID, "agent-def-1")
	mustInsertAgentRun(t, db, wsID, "agent-run-1", "agent-def-1", nil, nil)
	mustInsertEvalSuite(t, db, wsID, "suite-1")
	mustInsertEvalRunWithProvenance(t, db, wsID, "eval-run-1", "suite-1", replayStringPtr("agent-run-1"), nil, nil)

	engine := eval.NewSQLiteReplayEngine(db)
	artifact, err := engine.BuildReplay(context.Background(), eval.ReplayRequest{
		EvalRunID:   "eval-run-1",
		WorkspaceID: wsID,
	})
	if err != nil {
		t.Fatalf("BuildReplay: %v", err)
	}
	if artifact.Input.Source.SourceRun == nil {
		t.Fatal("expected Input.Source.SourceRun to be populated from source aggregate")
	}
	if artifact.Input.Source.SourceRun.RunID != "agent-run-1" {
		t.Fatalf("Input.Source.SourceRun.RunID = %q; want %q", artifact.Input.Source.SourceRun.RunID, "agent-run-1")
	}
}

// TestReplayEngine_ProducesNormalizedArtifact asserts that BuildReplay returns a
// ReplayArtifact with all canonical fields populated. (task-C2.3)
func TestReplayEngine_ProducesNormalizedArtifact(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	wsID := mustCreateWorkspace(t, db, "normalized")
	mustInsertAgentDefinition(t, db, wsID, "agent-def-1")
	mustInsertAgentRun(t, db, wsID, "agent-run-1", "agent-def-1", nil, nil)
	mustInsertEvalSuite(t, db, wsID, "suite-1")
	mustInsertEvalRunWithProvenance(t, db, wsID, "eval-run-1", "suite-1", replayStringPtr("agent-run-1"), nil, nil)

	engine := eval.NewSQLiteReplayEngine(db)
	artifact, err := engine.BuildReplay(context.Background(), eval.ReplayRequest{
		EvalRunID:   "eval-run-1",
		WorkspaceID: wsID,
	})
	if err != nil {
		t.Fatalf("BuildReplay: %v", err)
	}
	if artifact.FinalOutcome == "" {
		t.Fatal("expected FinalOutcome to be set in normalized artifact")
	}
	if artifact.Request.EvalRunID != "eval-run-1" {
		t.Fatalf("Request.EvalRunID = %q; want %q", artifact.Request.EvalRunID, "eval-run-1")
	}
	if artifact.BuiltAt.IsZero() {
		t.Fatal("expected BuiltAt to be set")
	}
}

// TestReplayEngine_DoesNotMutateSourceState asserts that BuildReplay leaves the
// original agent_run and eval_run rows unchanged. (task-C2.3)
func TestReplayEngine_DoesNotMutateSourceState(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	wsID := mustCreateWorkspace(t, db, "no-mutate")
	mustInsertAgentDefinition(t, db, wsID, "agent-def-1")
	mustInsertAgentRun(t, db, wsID, "agent-run-1", "agent-def-1", nil, nil)
	mustInsertEvalSuite(t, db, wsID, "suite-1")
	mustInsertEvalRunWithProvenance(t, db, wsID, "eval-run-1", "suite-1", replayStringPtr("agent-run-1"), nil, nil)

	var statusBefore string
	if err := db.QueryRow(`SELECT status FROM agent_run WHERE id = ?`, "agent-run-1").Scan(&statusBefore); err != nil {
		t.Fatalf("read agent_run before: %v", err)
	}
	var evalStatusBefore string
	if err := db.QueryRow(`SELECT status FROM eval_run WHERE id = ?`, "eval-run-1").Scan(&evalStatusBefore); err != nil {
		t.Fatalf("read eval_run before: %v", err)
	}

	engine := eval.NewSQLiteReplayEngine(db)
	if _, err := engine.BuildReplay(context.Background(), eval.ReplayRequest{
		EvalRunID:   "eval-run-1",
		WorkspaceID: wsID,
	}); err != nil {
		t.Fatalf("BuildReplay: %v", err)
	}

	var statusAfter string
	if err := db.QueryRow(`SELECT status FROM agent_run WHERE id = ?`, "agent-run-1").Scan(&statusAfter); err != nil {
		t.Fatalf("read agent_run after: %v", err)
	}
	var evalStatusAfter string
	if err := db.QueryRow(`SELECT status FROM eval_run WHERE id = ?`, "eval-run-1").Scan(&evalStatusAfter); err != nil {
		t.Fatalf("read eval_run after: %v", err)
	}

	if statusAfter != statusBefore {
		t.Fatalf("agent_run.status mutated: was %q, now %q", statusBefore, statusAfter)
	}
	if evalStatusAfter != evalStatusBefore {
		t.Fatalf("eval_run.status mutated: was %q, now %q", evalStatusBefore, evalStatusAfter)
	}
}

// TestRunnerService_Run_WithoutProvenance_UsesLegacyPath asserts that a run
// without replay provenance uses the keyword-scoring path unchanged. (task-C2.4)
func TestRunnerService_Run_WithoutProvenance_UsesLegacyPath(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	wsID := mustCreateWorkspace(t, db, "legacy-path")
	svc := eval.NewSuiteService(db)
	suite, err := svc.Create(context.Background(), eval.CreateSuiteInput{
		WorkspaceID: wsID,
		Name:        "Legacy Suite",
		Domain:      "support",
		TestCases:   []eval.TestCase{{Input: "hello", ExpectedKeywords: []string{"hello"}, ShouldAbstain: false}},
		Thresholds:  eval.Thresholds{Groundedness: 0.5, Exactitude: 0.5, Abstention: 0.5, Policy: 0.5},
	})
	if err != nil {
		t.Fatalf("create suite: %v", err)
	}

	runner := eval.NewRunnerService(db)
	run, err := runner.Run(context.Background(), eval.RunInput{
		WorkspaceID: wsID,
		EvalSuiteID: suite.ID,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if run.Status != "passed" && run.Status != "failed" {
		t.Fatalf("expected status passed or failed, got %q", run.Status)
	}
	if run.ReplayArtifact != nil {
		t.Fatal("expected ReplayArtifact to be nil on legacy path")
	}
}

// TestRunnerService_Run_WithReplayProvenance_UsesReplayEngine asserts that a run
// with valid replay source refs populates ReplayArtifact on the result. (task-C2.4)
func TestRunnerService_Run_WithReplayProvenance_UsesReplayEngine(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	wsID := mustCreateWorkspace(t, db, "replay-path")
	mustInsertAgentDefinition(t, db, wsID, "agent-def-1")
	mustInsertAgentRun(t, db, wsID, "agent-run-1", "agent-def-1", nil, nil)
	mustInsertEvalSuite(t, db, wsID, "suite-1")

	svc := eval.NewSuiteService(db)
	suite, err := svc.Create(context.Background(), eval.CreateSuiteInput{
		WorkspaceID: wsID,
		Name:        "Replay Suite",
		Domain:      "support",
		TestCases:   []eval.TestCase{{Input: "hello", ExpectedKeywords: []string{"hello"}, ShouldAbstain: false}},
		Thresholds:  eval.Thresholds{Groundedness: 0.5, Exactitude: 0.5, Abstention: 0.5, Policy: 0.5},
	})
	if err != nil {
		t.Fatalf("create suite: %v", err)
	}

	runner := eval.NewRunnerServiceWithReplay(db, eval.NewSQLiteReplayEngine(db))
	run, err := runner.Run(context.Background(), eval.RunInput{
		WorkspaceID: wsID,
		EvalSuiteID: suite.ID,
		Provenance: &eval.ReplayProvenance{
			Mode:             eval.ReplayModeReplay,
			SourceAgentRunID: replayStringPtr("agent-run-1"),
		},
	})
	if err != nil {
		t.Fatalf("Run with replay provenance: %v", err)
	}
	if run.ReplayArtifact == nil {
		t.Fatal("expected ReplayArtifact to be populated when replay provenance is present")
	}
	if run.ReplayArtifact.FinalOutcome == "" {
		t.Fatal("expected ReplayArtifact.FinalOutcome to be set")
	}
}

func TestRunnerService_Run_WithScoring_PopulatesScoredResult(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	wsID := mustCreateWorkspace(t, db, "replay-score")
	mustInsertAgentDefinition(t, db, wsID, "agent-def-1")
	traceID := "trace-score-1"
	mustInsertAgentRun(t, db, wsID, "agent-run-1", "agent-def-1", nil, &traceID)
	mustInsertEvalSuite(t, db, wsID, "suite-1")
	mustInsertAuditEvent(t, db, wsID, "audit-1", traceID, "tool.executed", time.Unix(1_700_000_300, 0).UTC())

	svc := eval.NewSuiteService(db)
	suite, err := svc.Create(context.Background(), eval.CreateSuiteInput{
		WorkspaceID: wsID,
		Name:        "Replay Score Suite",
		Domain:      "support",
		TestCases:   []eval.TestCase{{Input: "hello", ExpectedKeywords: []string{"hello"}, ShouldAbstain: false}},
		Thresholds:  eval.Thresholds{Groundedness: 0.5, Exactitude: 0.5, Abstention: 0.5, Policy: 0.5},
	})
	if err != nil {
		t.Fatalf("create suite: %v", err)
	}

	runner := eval.NewRunnerServiceWithReplay(db, eval.NewSQLiteReplayEngine(db))
	run, err := runner.Run(context.Background(), eval.RunInput{
		WorkspaceID: wsID,
		EvalSuiteID: suite.ID,
		Provenance: &eval.ReplayProvenance{
			Mode:             eval.ReplayModeReplay,
			SourceAgentRunID: replayStringPtr("agent-run-1"),
			SourceTraceID:    &traceID,
		},
		Scenario: &eval.GoldenScenario{
			ID:          "sc-replay-score",
			Title:       "Replay score",
			Description: "Runner integration scoring test",
			Domain:      "support",
			InputEvent:  eval.ScenarioInputEvent{Type: "case.updated"},
			Expected: eval.ScenarioExpected{
				FinalOutcome:     "success",
				RequiredEvidence: []string{"ev-1"},
				ToolCalls:        []eval.ExpectedToolCall{{ToolName: "crm.case.get", Required: true}},
				AuditEvents:      []string{"tool.executed"},
			},
			Thresholds: eval.ScenarioThresholds{MinScore: 70, MaxLatencyMs: 10_000, MaxToolCalls: 2, MaxRetries: 1},
		},
	})
	if err != nil {
		t.Fatalf("Run with replay scoring: %v", err)
	}
	if run.ScoredResult == nil {
		t.Fatal("expected ScoredResult to be populated")
	}
	if run.Scores.ScoredResult == nil {
		t.Fatal("expected Scores.ScoredResult to be persisted in the in-memory run")
	}
	if run.ScoredResult.Scorecard.TotalScore <= 0 {
		t.Fatalf("expected positive scorecard total, got %f", run.ScoredResult.Scorecard.TotalScore)
	}

	persisted, err := runner.GetRun(context.Background(), wsID, run.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if persisted.ScoredResult == nil {
		t.Fatal("expected persisted run to include ScoredResult")
	}
	if persisted.ScoredResult.ScenarioID != "sc-replay-score" {
		t.Fatalf("ScenarioID = %q; want %q", persisted.ScoredResult.ScenarioID, "sc-replay-score")
	}
}

func TestRunnerService_Run_WithoutScenario_NilScoredResult(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	wsID := mustCreateWorkspace(t, db, "replay-noscenario")
	mustInsertAgentDefinition(t, db, wsID, "agent-def-1")
	mustInsertAgentRun(t, db, wsID, "agent-run-1", "agent-def-1", nil, nil)

	svc := eval.NewSuiteService(db)
	suite, err := svc.Create(context.Background(), eval.CreateSuiteInput{
		WorkspaceID: wsID,
		Name:        "Replay No Scenario Suite",
		Domain:      "support",
		TestCases:   []eval.TestCase{{Input: "hello", ExpectedKeywords: []string{"hello"}, ShouldAbstain: false}},
		Thresholds:  eval.Thresholds{Groundedness: 0.5, Exactitude: 0.5, Abstention: 0.5, Policy: 0.5},
	})
	if err != nil {
		t.Fatalf("create suite: %v", err)
	}

	runner := eval.NewRunnerServiceWithReplay(db, eval.NewSQLiteReplayEngine(db))
	run, err := runner.Run(context.Background(), eval.RunInput{
		WorkspaceID: wsID,
		EvalSuiteID: suite.ID,
		Provenance: &eval.ReplayProvenance{
			Mode:             eval.ReplayModeReplay,
			SourceAgentRunID: replayStringPtr("agent-run-1"),
		},
	})
	if err != nil {
		t.Fatalf("Run without scenario: %v", err)
	}
	if run.ScoredResult != nil {
		t.Fatal("expected ScoredResult to be nil when scenario is absent")
	}
	if run.Scores.ScoredResult != nil {
		t.Fatal("expected Scores.ScoredResult to be nil when scenario is absent")
	}
}

func replayStringPtr(v string) *string { return &v }

func replayTimePtr(v time.Time) *time.Time { return &v }

func replayDeref(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func mustInsertAgentDefinition(t *testing.T, db *sql.DB, wsID, agentDefinitionID string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO agent_definition (
			id, workspace_id, name, agent_type, allowed_tools, limits, trigger_config, status, created_at, updated_at
		) VALUES (?, ?, ?, 'support', '[]', '{}', '{}', 'active', datetime('now'), datetime('now'))`,
		agentDefinitionID, wsID, "Agent "+agentDefinitionID,
	)
	if err != nil {
		t.Fatalf("insert agent_definition: %v", err)
	}
}

func mustInsertAgentRun(
	t *testing.T,
	db *sql.DB,
	wsID, runID, agentDefinitionID string,
	cognitiveWorkspaceID, traceID *string,
) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO agent_run (
			id, workspace_id, agent_definition_id, trigger_type, status, trigger_context, inputs,
			retrieval_queries, retrieved_evidence_ids, reasoning_trace, tool_calls, output,
			trace_id, started_at, completed_at, created_at, updated_at, cognitive_workspace_id
		) VALUES (?, ?, ?, 'manual', 'success', ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'), datetime('now'), datetime('now'), ?)`,
		runID, wsID, agentDefinitionID,
		`{"type":"case.updated"}`,
		`{"caseId":"case-1"}`,
		`["search cases"]`,
		`["ev-1"]`,
		`{"steps":["observe"]}`,
		`[{"tool":"crm.case.get","status":"executed","params":{"id":"case-1"}}]`,
		`{"summary":"done"}`,
		traceID,
		cognitiveWorkspaceID,
	)
	if err != nil {
		t.Fatalf("insert agent_run: %v", err)
	}
}

func mustInsertEvalSuite(t *testing.T, db *sql.DB, wsID, suiteID string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO eval_suite (id, workspace_id, name, domain, test_cases, thresholds, created_at, updated_at)
		VALUES (?, ?, ?, 'support', '[]', '{}', datetime('now'), datetime('now'))`,
		suiteID, wsID, "Suite "+suiteID,
	)
	if err != nil {
		t.Fatalf("insert eval_suite: %v", err)
	}
}

func mustInsertEvalRunWithProvenance(
	t *testing.T,
	db *sql.DB,
	wsID, evalRunID, suiteID string,
	sourceAgentRunID, sourceCognitiveWorkspaceID, sourceTraceID *string,
) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO eval_run (
			id, workspace_id, eval_suite_id, status, scores, details,
			source_agent_run_id, source_cognitive_workspace_id, source_trace_id, replay_mode,
			started_at, created_at
		) VALUES (?, ?, ?, 'running', '{}', '[]', ?, ?, ?, 'replay', datetime('now'), datetime('now'))`,
		evalRunID, wsID, suiteID, sourceAgentRunID, sourceCognitiveWorkspaceID, sourceTraceID,
	)
	if err != nil {
		t.Fatalf("insert eval_run: %v", err)
	}
}

func mustInsertCognitiveWorkspace(t *testing.T, db *sql.DB, wsID, cwID string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO cognitive_workspace (id, workspace_id, status, created_at)
		VALUES (?, ?, 'active', datetime('now'))`,
		cwID, wsID,
	)
	if err != nil {
		t.Fatalf("insert cognitive_workspace: %v", err)
	}
}

func mustInsertReasoningEvent(t *testing.T, db *sql.DB, id, cwID, eventType, payload string, createdAt time.Time) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO reasoning_event (id, cognitive_workspace_id, actor_agent_id, event_type, payload, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		id, cwID, "agent-1", eventType, payload, createdAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("insert reasoning_event: %v", err)
	}
}

func mustInsertAuditEvent(t *testing.T, db *sql.DB, wsID, eventID, traceID, action string, createdAt time.Time) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO audit_event (
			id, workspace_id, actor_id, actor_type, action, details, permissions_checked, outcome, trace_id, created_at
		) VALUES (?, ?, 'agent-1', 'agent', ?, '{}', '[]', 'success', ?, ?)`,
		eventID, wsID, action, traceID, createdAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("insert audit_event: %v", err)
	}
}
