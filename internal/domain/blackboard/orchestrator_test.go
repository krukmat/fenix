package blackboard

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	isqlite "github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	_ "modernc.org/sqlite"
)

type arbitratorStub struct {
	result *ArbitrationResult
	err    error

	mu    sync.Mutex
	calls int
	block chan struct{}
}

func (s *arbitratorStub) RankWorkspace(_ context.Context, cognitiveWorkspaceID string, _ ArbitrationConfig) (*ArbitrationResult, error) {
	s.mu.Lock()
	s.calls++
	block := s.block
	s.mu.Unlock()
	if block != nil {
		<-block
	}
	if s.err != nil {
		return nil, s.err
	}
	if s.result != nil {
		return s.result, nil
	}
	return &ArbitrationResult{CognitiveWorkspaceID: cognitiveWorkspaceID, Ranked: []RankedHypothesis{}}, nil
}

func (s *arbitratorStub) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

type plannerStub struct {
	result *CollaborativePlanningResult
	err    error
	calls  int
}

func (s *plannerStub) BuildWorkspacePlan(_ context.Context, _ string, _ PlanningConfig) (*CollaborativePlanningResult, error) {
	s.calls++
	return s.result, s.err
}

type executorStub struct {
	outcome *ExecutionOutcome
	err     error
	calls   int
}

func (s *executorStub) Execute(_ context.Context, _ CognitiveWorkspace, _ *CollaborativePlanningResult) (*ExecutionOutcome, error) {
	s.calls++
	return s.outcome, s.err
}

func setupBlackboardPipelineDB(t *testing.T) (*sql.DB, CognitiveWorkspace) {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := isqlite.MigrateUp(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	workspace := CognitiveWorkspace{ID: "cw-pipeline", WorkspaceID: "ws-pipeline"}
	if _, err := db.Exec(`INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES (?, ?, ?, datetime('now'), datetime('now'))`, workspace.WorkspaceID, "Pipeline WS", "pipeline-ws"); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO cognitive_workspace (id, workspace_id, status, created_at)
		VALUES (?, ?, 'active', datetime('now'))`, workspace.ID, workspace.WorkspaceID); err != nil {
		t.Fatalf("cognitive_workspace insert: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })
	return db, workspace
}

func readyPlanningResult(cwID string) *CollaborativePlanningResult {
	proposal := CollaborativePlanProposal{
		ProposalID: "proposal-1",
		State:      PlanningStateReady,
		Steps: []ToolSequenceStep{
			{Sequence: 1, ToolName: "tool-1", Params: json.RawMessage(`{"n":1}`)},
			{Sequence: 2, ToolName: "tool-2", Params: json.RawMessage(`{"n":2}`)},
			{Sequence: 3, ToolName: "tool-3", Params: json.RawMessage(`{"n":3}`)},
		},
	}
	return &CollaborativePlanningResult{
		CognitiveWorkspaceID: cwID,
		GeneratedAt:          time.Date(2026, 5, 18, 16, 0, 0, 0, time.UTC),
		State:                PlanningStateReady,
		SelectedProposal:     &proposal,
		Proposals:            []CollaborativePlanProposal{proposal},
	}
}

func TestBlackboardOrchestrator_RunPipeline_NoHypotheses(t *testing.T) {
	t.Parallel()

	db, workspace := setupBlackboardPipelineDB(t)
	arb := &arbitratorStub{result: &ArbitrationResult{CognitiveWorkspaceID: workspace.ID, Ranked: []RankedHypothesis{}}}
	planner := &plannerStub{}
	executor := &executorStub{}
	orch := &Orchestrator{db: db, arb: arb, planner: planner, executor: executor, now: func() time.Time { return time.Date(2026, 5, 18, 16, 0, 0, 0, time.UTC) }}

	outcome, err := orch.RunPipeline(context.Background(), workspace.ID)
	if err != nil {
		t.Fatalf("RunPipeline() error = %v", err)
	}
	if outcome.DeferralReason != deferralReasonNoHypotheses {
		t.Fatalf("DeferralReason = %q, want %q", outcome.DeferralReason, deferralReasonNoHypotheses)
	}
	if planner.calls != 0 {
		t.Fatalf("planner calls = %d, want 0", planner.calls)
	}
	if executor.calls != 0 {
		t.Fatalf("executor calls = %d, want 0", executor.calls)
	}
}

func TestBlackboardOrchestrator_RunPipeline_HappyPath(t *testing.T) {
	t.Parallel()

	db, workspace := setupBlackboardPipelineDB(t)
	arb := &arbitratorStub{result: &ArbitrationResult{
		CognitiveWorkspaceID: workspace.ID,
		Ranked: []RankedHypothesis{
			{Rank: 1, Score: 0.9, Hypothesis: SignalHypothesis{ID: "hyp-1"}},
			{Rank: 2, Score: 0.8, Hypothesis: SignalHypothesis{ID: "hyp-2"}},
			{Rank: 3, Score: 0.7, Hypothesis: SignalHypothesis{ID: "hyp-3"}},
		},
	}}
	planner := &plannerStub{result: readyPlanningResult(workspace.ID)}
	executor := &executorStub{outcome: &ExecutionOutcome{
		CognitiveWorkspaceID: workspace.ID,
		WorkspaceID:          workspace.WorkspaceID,
		Executed: []ExecutedStep{
			{Step: ToolSequenceStep{Sequence: 1, ToolName: "tool-1"}},
			{Step: ToolSequenceStep{Sequence: 2, ToolName: "tool-2"}},
			{Step: ToolSequenceStep{Sequence: 3, ToolName: "tool-3"}},
		},
	}}
	orch := &Orchestrator{db: db, arb: arb, planner: planner, executor: executor, now: func() time.Time { return time.Date(2026, 5, 18, 16, 5, 0, 0, time.UTC) }}

	outcome, err := orch.RunPipeline(context.Background(), workspace.ID)
	if err != nil {
		t.Fatalf("RunPipeline() error = %v", err)
	}
	if len(outcome.Executed) != 3 {
		t.Fatalf("Executed len = %d, want 3", len(outcome.Executed))
	}
	if planner.calls != 1 || executor.calls != 1 {
		t.Fatalf("planner=%d executor=%d, want 1/1", planner.calls, executor.calls)
	}
}

func TestBlackboardOrchestrator_RunPipeline_DeduplicatesConcurrentCalls(t *testing.T) {
	t.Parallel()

	db, workspace := setupBlackboardPipelineDB(t)
	block := make(chan struct{})
	arb := &arbitratorStub{
		result: &ArbitrationResult{
			CognitiveWorkspaceID: workspace.ID,
			Ranked:               []RankedHypothesis{{Rank: 1, Score: 0.9, Hypothesis: SignalHypothesis{ID: "hyp-1"}}},
		},
		block: block,
	}
	planner := &plannerStub{result: readyPlanningResult(workspace.ID)}
	executor := &executorStub{outcome: &ExecutionOutcome{CognitiveWorkspaceID: workspace.ID, WorkspaceID: workspace.WorkspaceID}}
	orch := &Orchestrator{db: db, arb: arb, planner: planner, executor: executor, now: func() time.Time { return time.Date(2026, 5, 18, 16, 10, 0, 0, time.UTC) }}

	firstDone := make(chan error, 1)
	go func() {
		_, err := orch.RunPipeline(context.Background(), workspace.ID)
		firstDone <- err
	}()

	for arb.callCount() == 0 {
		time.Sleep(10 * time.Millisecond)
	}

	_, err := orch.RunPipeline(context.Background(), workspace.ID)
	if !errors.Is(err, ErrPipelineAlreadyRunning) {
		t.Fatalf("second RunPipeline() error = %v, want ErrPipelineAlreadyRunning", err)
	}

	close(block)
	if err := <-firstDone; err != nil {
		t.Fatalf("first RunPipeline() error = %v", err)
	}
}
