package blackboard

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/policy"
	isqlite "github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	_ "modernc.org/sqlite"
)

type plannerPolicyStub struct {
	decisions map[int]PlannerPolicyDecision
	calls     []int
}

func (s *plannerPolicyStub) DecideStep(_ context.Context, _, _ string, step ToolSequenceStep) (PlannerPolicyDecision, error) {
	s.calls = append(s.calls, step.Sequence)
	if decision, ok := s.decisions[step.Sequence]; ok {
		return decision, nil
	}
	return PlannerPolicyDecision{Effect: PlannerPolicyEffectAllow}, nil
}

type plannerApprovalStub struct {
	requestID string
	inputs    []policy.CreateApprovalRequestInput
}

func (s *plannerApprovalStub) CreateApprovalRequest(_ context.Context, input policy.CreateApprovalRequestInput) (*policy.ApprovalRequest, error) {
	s.inputs = append(s.inputs, input)
	return &policy.ApprovalRequest{
		ID:          s.requestID,
		WorkspaceID: input.WorkspaceID,
		RequestedBy: input.RequestedBy,
		ApproverID:  input.ApproverID,
		Action:      input.Action,
		Reason:      input.Reason,
		Status:      policy.ApprovalStatusPending,
		ExpiresAt:   input.ExpiresAt,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}, nil
}

type plannerToolCall struct {
	workspaceID string
	toolName    string
	params      json.RawMessage
}

type plannerToolStub struct {
	results map[string]json.RawMessage
	errs    map[string]error
	calls   []plannerToolCall
}

func (s *plannerToolStub) Execute(_ context.Context, workspaceID, toolName string, params json.RawMessage) (json.RawMessage, error) {
	s.calls = append(s.calls, plannerToolCall{
		workspaceID: workspaceID,
		toolName:    toolName,
		params:      append(json.RawMessage(nil), params...),
	})
	if result, ok := s.results[toolName]; ok {
		return append(json.RawMessage(nil), result...), s.errs[toolName]
	}
	return json.RawMessage(`{"tool":"` + toolName + `"}`), s.errs[toolName]
}

func setupPlannerExecutorDB(t *testing.T) (*sql.DB, CognitiveWorkspace) {
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

	workspace := CognitiveWorkspace{ID: "cw-exec", WorkspaceID: "ws-exec"}
	if _, err := db.Exec(`INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES (?, ?, ?, datetime('now'), datetime('now'))`, workspace.WorkspaceID, "Exec WS", "exec-ws"); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO cognitive_workspace (id, workspace_id, status, created_at)
		VALUES (?, ?, 'active', datetime('now'))`, workspace.ID, workspace.WorkspaceID); err != nil {
		t.Fatalf("cognitive_workspace insert: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })
	return db, workspace
}

func newTestPlannerExecutor(
	db *sql.DB,
	now time.Time,
	policyStub *plannerPolicyStub,
	approvalStub *plannerApprovalStub,
	toolStub *plannerToolStub,
) *PlannerExecutor {
	return &PlannerExecutor{
		memory:            NewMemoryStore(db),
		timeline:          NewReasoningTimeline(db),
		policy:            policyStub,
		approval:          approvalStub,
		tools:             toolStub,
		now:               func() time.Time { return now },
		resultMemoryKey:   DefaultPlannerExecutionResultMemoryKey,
		pendingMemoryKey:  DefaultPlannerPendingExecutionMemoryKey,
		defaultApproverID: "approver-1",
		approvalTTL:       time.Hour,
	}
}

func readyPlan(cwID string, steps ...ToolSequenceStep) *CollaborativePlanningResult {
	proposal := CollaborativePlanProposal{
		ProposalID:   "proposal-1",
		HypothesisID: "hyp-1",
		State:        PlanningStateReady,
		Steps:        steps,
	}
	return &CollaborativePlanningResult{
		CognitiveWorkspaceID: cwID,
		GeneratedAt:          time.Date(2026, 5, 18, 11, 0, 0, 0, time.UTC),
		State:                PlanningStateReady,
		SelectedProposal:     &proposal,
		Proposals:            []CollaborativePlanProposal{proposal},
	}
}

func loadStoredOutcome(t *testing.T, executor *PlannerExecutor, cwID string) *ExecutionOutcome {
	t.Helper()

	entry, err := executor.memory.Get(context.Background(), cwID, executor.resultKey())
	if err != nil {
		t.Fatalf("memory.Get(result): %v", err)
	}
	var outcome ExecutionOutcome
	if err := json.Unmarshal(entry.Value, &outcome); err != nil {
		t.Fatalf("json.Unmarshal(outcome): %v", err)
	}
	return &outcome
}

func timelineEvents(t *testing.T, executor *PlannerExecutor, cwID string, eventType EventType) []ReasoningEvent {
	t.Helper()

	events, err := executor.timeline.List(context.Background(), cwID, TimelineFilter{EventType: eventType})
	if err != nil {
		t.Fatalf("timeline.List(%s): %v", eventType, err)
	}
	return events
}

func TestPlannerExecutor_ExecuteDefersWhenPlanIsNotReady(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		state      PlanningState
		selected   *CollaborativePlanProposal
		wantReason string
	}{
		{name: "awaiting evidence", state: PlanningStateAwaitingEvidence, selected: nil, wantReason: "awaiting_evidence"},
		{name: "needs review", state: PlanningStateNeedsReview, selected: nil, wantReason: "needs_review"},
		{name: "pending approval", state: PlanningStatePendingApproval, selected: &CollaborativePlanProposal{ProposalID: "proposal-1"}, wantReason: "pending_approval"},
		{name: "ready without selected proposal", state: PlanningStateReady, selected: nil, wantReason: "missing_selected_proposal"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, workspace := setupPlannerExecutorDB(t)
			now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
			policyStub := &plannerPolicyStub{}
			approvalStub := &plannerApprovalStub{requestID: "apr-1"}
			toolStub := &plannerToolStub{}
			executor := newTestPlannerExecutor(db, now, policyStub, approvalStub, toolStub)

			plan := &CollaborativePlanningResult{
				CognitiveWorkspaceID: workspace.ID,
				GeneratedAt:          now,
				State:                tc.state,
				SelectedProposal:     tc.selected,
			}

			outcome, err := executor.Execute(context.Background(), workspace, plan)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if outcome.DeferralReason != tc.wantReason {
				t.Fatalf("DeferralReason = %q, want %q", outcome.DeferralReason, tc.wantReason)
			}
			if len(policyStub.calls) != 0 {
				t.Fatalf("policy calls = %v, want none", policyStub.calls)
			}
			if len(approvalStub.inputs) != 0 {
				t.Fatalf("approval calls = %d, want 0", len(approvalStub.inputs))
			}
			if len(toolStub.calls) != 0 {
				t.Fatalf("tool calls = %d, want 0", len(toolStub.calls))
			}

			stored := loadStoredOutcome(t, executor, workspace.ID)
			if stored.DeferralReason != tc.wantReason {
				t.Fatalf("stored DeferralReason = %q, want %q", stored.DeferralReason, tc.wantReason)
			}
		})
	}
}

func TestPlannerExecutor_ExecuteRunsAllowedStepsInOrder(t *testing.T) {
	t.Parallel()

	db, workspace := setupPlannerExecutorDB(t)
	now := time.Date(2026, 5, 18, 12, 30, 0, 0, time.UTC)
	policyStub := &plannerPolicyStub{}
	approvalStub := &plannerApprovalStub{requestID: "apr-1"}
	toolStub := &plannerToolStub{
		results: map[string]json.RawMessage{
			"tool-1": json.RawMessage(`{"ok":1}`),
			"tool-2": json.RawMessage(`{"ok":2}`),
			"tool-3": json.RawMessage(`{"ok":3}`),
		},
	}
	executor := newTestPlannerExecutor(db, now, policyStub, approvalStub, toolStub)

	outcome, err := executor.Execute(context.Background(), workspace, readyPlan(workspace.ID,
		ToolSequenceStep{Sequence: 1, ToolName: "tool-1"},
		ToolSequenceStep{Sequence: 2, ToolName: "tool-2"},
		ToolSequenceStep{Sequence: 3, ToolName: "tool-3"},
	))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(outcome.Executed) != 3 {
		t.Fatalf("Executed len = %d, want 3", len(outcome.Executed))
	}
	for i, call := range toolStub.calls {
		if call.toolName != "tool-"+string(rune('1'+i)) {
			t.Fatalf("tool call %d = %q", i, call.toolName)
		}
	}
	if outcome.DeferralReason != "" {
		t.Fatalf("DeferralReason = %q, want empty", outcome.DeferralReason)
	}
	if len(timelineEvents(t, executor, workspace.ID, EventTypeObservation)) != 3 {
		t.Fatalf("observation events != 3")
	}
}

func TestPlannerExecutor_ExecutePausesForApproval(t *testing.T) {
	t.Parallel()

	db, workspace := setupPlannerExecutorDB(t)
	now := time.Date(2026, 5, 18, 13, 0, 0, 0, time.UTC)
	policyStub := &plannerPolicyStub{}
	approvalStub := &plannerApprovalStub{requestID: "apr-approval"}
	toolStub := &plannerToolStub{}
	executor := newTestPlannerExecutor(db, now, policyStub, approvalStub, toolStub)

	ctx := ctxkeys.WithValue(context.Background(), ctxkeys.UserID, "user-1")
	outcome, err := executor.Execute(ctx, workspace, readyPlan(workspace.ID,
		ToolSequenceStep{Sequence: 1, ToolName: "tool-1"},
		ToolSequenceStep{Sequence: 2, ToolName: "tool-2", RequiresApproval: true},
		ToolSequenceStep{Sequence: 3, ToolName: "tool-3"},
	))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(outcome.Executed) != 1 {
		t.Fatalf("Executed len = %d, want 1", len(outcome.Executed))
	}
	if outcome.Pending == nil || outcome.Pending.ApprovalID != "apr-approval" {
		t.Fatalf("Pending = %#v, want approval apr-approval", outcome.Pending)
	}
	if len(approvalStub.inputs) != 1 {
		t.Fatalf("approval requests = %d, want 1", len(approvalStub.inputs))
	}
	if approvalStub.inputs[0].RequestedBy != "user-1" {
		t.Fatalf("RequestedBy = %q, want user-1", approvalStub.inputs[0].RequestedBy)
	}
	if len(timelineEvents(t, executor, workspace.ID, EventTypeIntent)) != 1 {
		t.Fatalf("intent events != 1")
	}
	if _, err := executor.loadPendingState(context.Background(), workspace.ID); err != nil {
		t.Fatalf("loadPendingState() error = %v", err)
	}
}

func TestPlannerExecutor_ExecuteStopsOnPolicyDeny(t *testing.T) {
	t.Parallel()

	db, workspace := setupPlannerExecutorDB(t)
	now := time.Date(2026, 5, 18, 13, 30, 0, 0, time.UTC)
	policyStub := &plannerPolicyStub{
		decisions: map[int]PlannerPolicyDecision{
			2: {Effect: PlannerPolicyEffectDeny, Reason: "too_sensitive"},
		},
	}
	executor := newTestPlannerExecutor(db, now, policyStub, &plannerApprovalStub{requestID: "apr-1"}, &plannerToolStub{})

	outcome, err := executor.Execute(context.Background(), workspace, readyPlan(workspace.ID,
		ToolSequenceStep{Sequence: 1, ToolName: "tool-1"},
		ToolSequenceStep{Sequence: 2, ToolName: "tool-2"},
	))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(outcome.Executed) != 1 {
		t.Fatalf("Executed len = %d, want 1", len(outcome.Executed))
	}
	if outcome.DeferralReason != deferralReasonPolicyDeny {
		t.Fatalf("DeferralReason = %q, want %q", outcome.DeferralReason, deferralReasonPolicyDeny)
	}
	if len(timelineEvents(t, executor, workspace.ID, EventTypeRisk)) != 1 {
		t.Fatalf("risk events != 1")
	}
}

func TestPlannerExecutor_ExecuteStopsOnToolFailure(t *testing.T) {
	t.Parallel()

	db, workspace := setupPlannerExecutorDB(t)
	now := time.Date(2026, 5, 18, 14, 0, 0, 0, time.UTC)
	toolStub := &plannerToolStub{
		errs: map[string]error{"tool-2": errors.New("boom")},
	}
	executor := newTestPlannerExecutor(db, now, &plannerPolicyStub{}, &plannerApprovalStub{requestID: "apr-1"}, toolStub)

	outcome, err := executor.Execute(context.Background(), workspace, readyPlan(workspace.ID,
		ToolSequenceStep{Sequence: 1, ToolName: "tool-1"},
		ToolSequenceStep{Sequence: 2, ToolName: "tool-2"},
	))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(outcome.Executed) != 2 {
		t.Fatalf("Executed len = %d, want 2", len(outcome.Executed))
	}
	if outcome.DeferralReason != deferralReasonToolFailure {
		t.Fatalf("DeferralReason = %q, want %q", outcome.DeferralReason, deferralReasonToolFailure)
	}
	if outcome.Executed[1].Error == "" {
		t.Fatal("expected step 2 error")
	}
}

func TestPlannerExecutor_ExecuteGeneratesDeterministicIdempotencyKeys(t *testing.T) {
	t.Parallel()

	db, workspace := setupPlannerExecutorDB(t)
	now := time.Date(2026, 5, 18, 14, 30, 0, 0, time.UTC)
	executor := newTestPlannerExecutor(db, now, &plannerPolicyStub{}, &plannerApprovalStub{requestID: "apr-1"}, &plannerToolStub{})
	plan := readyPlan(workspace.ID,
		ToolSequenceStep{Sequence: 1, ToolName: "tool-1"},
		ToolSequenceStep{Sequence: 2, ToolName: "tool-2"},
	)

	first, err := executor.Execute(context.Background(), workspace, plan)
	if err != nil {
		t.Fatalf("first Execute() error = %v", err)
	}
	second, err := executor.Execute(context.Background(), workspace, plan)
	if err != nil {
		t.Fatalf("second Execute() error = %v", err)
	}
	for i := range first.Executed {
		if first.Executed[i].IdempotencyKey != second.Executed[i].IdempotencyKey {
			t.Fatalf("idempotency key %d differs: %q vs %q", i, first.Executed[i].IdempotencyKey, second.Executed[i].IdempotencyKey)
		}
	}
}

func TestPlannerExecutor_ResumeFromApproval(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		approved        bool
		wantExecutedLen int
		wantDeferral    string
		wantToolCalls   int
	}{
		{name: "approved continues from pending step", approved: true, wantExecutedLen: 3, wantToolCalls: 3},
		{name: "rejected stops cleanly", approved: false, wantExecutedLen: 1, wantDeferral: deferralReasonApprovalDenied, wantToolCalls: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, workspace := setupPlannerExecutorDB(t)
			now := time.Date(2026, 5, 18, 15, 0, 0, 0, time.UTC)
			toolStub := &plannerToolStub{}
			executor := newTestPlannerExecutor(db, now, &plannerPolicyStub{}, &plannerApprovalStub{requestID: "apr-1"}, toolStub)

			_, err := executor.Execute(context.Background(), workspace, readyPlan(workspace.ID,
				ToolSequenceStep{Sequence: 1, ToolName: "tool-1"},
				ToolSequenceStep{Sequence: 2, ToolName: "tool-2", RequiresApproval: true},
				ToolSequenceStep{Sequence: 3, ToolName: "tool-3"},
			))
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			outcome, err := executor.ResumeFromApproval(context.Background(), ApprovalDecision{
				CognitiveWorkspaceID: workspace.ID,
				ApprovalID:           "apr-1",
				Approved:             tc.approved,
				DecidedBy:            "approver-1",
				DecidedAt:            now.Add(5 * time.Minute),
			})
			if err != nil {
				t.Fatalf("ResumeFromApproval() error = %v", err)
			}
			if len(outcome.Executed) != tc.wantExecutedLen {
				t.Fatalf("Executed len = %d, want %d", len(outcome.Executed), tc.wantExecutedLen)
			}
			if outcome.DeferralReason != tc.wantDeferral {
				t.Fatalf("DeferralReason = %q, want %q", outcome.DeferralReason, tc.wantDeferral)
			}
			if len(toolStub.calls) != tc.wantToolCalls {
				t.Fatalf("tool calls = %d, want %d", len(toolStub.calls), tc.wantToolCalls)
			}
			if _, err := executor.loadPendingState(context.Background(), workspace.ID); !errors.Is(err, ErrPlannerExecutionPendingNotFound) {
				t.Fatalf("pending state error = %v, want ErrPlannerExecutionPendingNotFound", err)
			}
		})
	}
}

func TestPlannerExecutor_ExecuteWritesOneObservationPerExecutedStep(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name             string
		steps            []ToolSequenceStep
		toolErrs         map[string]error
		wantObservations int
	}{
		{
			name: "all successful",
			steps: []ToolSequenceStep{
				{Sequence: 1, ToolName: "tool-1"},
				{Sequence: 2, ToolName: "tool-2"},
				{Sequence: 3, ToolName: "tool-3"},
			},
			wantObservations: 3,
		},
		{
			name: "failure still records failed execution",
			steps: []ToolSequenceStep{
				{Sequence: 1, ToolName: "tool-1"},
				{Sequence: 2, ToolName: "tool-2"},
			},
			toolErrs:         map[string]error{"tool-2": errors.New("boom")},
			wantObservations: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, workspace := setupPlannerExecutorDB(t)
			now := time.Date(2026, 5, 18, 15, 30, 0, 0, time.UTC)
			executor := newTestPlannerExecutor(db, now, &plannerPolicyStub{}, &plannerApprovalStub{requestID: "apr-1"}, &plannerToolStub{errs: tc.toolErrs})

			if _, err := executor.Execute(context.Background(), workspace, readyPlan(workspace.ID, tc.steps...)); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			observations := timelineEvents(t, executor, workspace.ID, EventTypeObservation)
			if len(observations) != tc.wantObservations {
				t.Fatalf("observation events = %d, want %d", len(observations), tc.wantObservations)
			}
		})
	}
}
