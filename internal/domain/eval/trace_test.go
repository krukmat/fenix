package eval_test

// F2-T1: Tests for ActualRunTrace DTO shape and required fields.

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/eval"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite/sqlcgen"
)

// stubTraceStore implements eval.TraceStore with in-memory data for deterministic testing.
type stubTraceStore struct {
	run      sqlcgen.AgentRun
	events   []sqlcgen.AuditEvent
	approvals []sqlcgen.ApprovalRequest
}

func (s *stubTraceStore) GetAgentRunByID(ctx context.Context, arg sqlcgen.GetAgentRunByIDParams) (sqlcgen.AgentRun, error) {
	return s.run, nil
}

func (s *stubTraceStore) ListAuditEventsByTraceID(ctx context.Context, traceID *string) ([]sqlcgen.AuditEvent, error) {
	return s.events, nil
}

func (s *stubTraceStore) ListApprovalRequestsByIDs(ctx context.Context, ids []string) ([]sqlcgen.ApprovalRequest, error) {
	return s.approvals, nil
}

// --- helpers ---

func strPtr(s string) *string { return &s }
func int64Ptr(i int64) *int64 { return &i }
func float64Ptr(f float64) *float64 { return &f }

func makeRun(traceID *string) sqlcgen.AgentRun {
	now := time.Now().UTC()
	completed := now.Add(500 * time.Millisecond)
	return sqlcgen.AgentRun{
		ID:                   "run-001",
		WorkspaceID:          "ws-001",
		AgentDefinitionID:    "agent-support-v1",
		TriggerType:          "event",
		TriggerContext:       json.RawMessage(`{"event":"case.created","case_id":"CASE-001"}`),
		Status:               "success",
		Inputs:               json.RawMessage(`{"case_id":"CASE-001"}`),
		RetrievalQueries:     json.RawMessage(`["open cases for ACC-001"]`),
		RetrievedEvidenceIds: json.RawMessage(`["ev-001","ev-002"]`),
		ReasoningTrace:       json.RawMessage(`[{"step":1,"thought":"retrieving context"}]`),
		ToolCalls:            json.RawMessage(`[{"tool":"send_email","status":"executed"}]`),
		Output:               json.RawMessage(`{"reply":"Case acknowledged."}`),
		TotalTokens:          int64Ptr(512),
		TotalCost:            float64Ptr(0.007),
		LatencyMs:            int64Ptr(480),
		TraceID:              traceID,
		StartedAt:            now,
		CompletedAt:          &completed,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}

func makeAuditEvents(traceID string) []sqlcgen.AuditEvent {
	now := time.Now().UTC()
	return []sqlcgen.AuditEvent{
		{
			ID:                 "evt-001",
			WorkspaceID:        "ws-001",
			ActorID:            "run-001",
			ActorType:          "agent",
			Action:             "agent.run.started",
			Details:            json.RawMessage(`{}`),
			PermissionsChecked: json.RawMessage(`[]`),
			Outcome:            "success",
			TraceID:            strPtr(traceID),
			CreatedAt:          now,
		},
		{
			ID:                 "evt-002",
			WorkspaceID:        "ws-001",
			ActorID:            "run-001",
			ActorType:          "agent",
			Action:             "tool.executed",
			EntityType:         strPtr("tool"),
			EntityID:           strPtr("send_email"),
			Details:            json.RawMessage(`{"approval_id":"apr-001"}`),
			PermissionsChecked: json.RawMessage(`["send_email"]`),
			Outcome:            "success",
			TraceID:            strPtr(traceID),
			CreatedAt:          now.Add(200 * time.Millisecond),
		},
		{
			ID:                 "evt-003",
			WorkspaceID:        "ws-001",
			ActorID:            "run-001",
			ActorType:          "agent",
			Action:             "policy.decision",
			Details:            json.RawMessage(`{"action":"tool:send_email","outcome":"require_approval"}`),
			PermissionsChecked: json.RawMessage(`["policy:send_email"]`),
			Outcome:            "success",
			TraceID:            strPtr(traceID),
			CreatedAt:          now.Add(100 * time.Millisecond),
		},
	}
}

func makeApprovals() []sqlcgen.ApprovalRequest {
	now := time.Now().UTC()
	decided := now.Add(300 * time.Millisecond)
	return []sqlcgen.ApprovalRequest{
		{
			ID:          "apr-001",
			WorkspaceID: "ws-001",
			RequestedBy: "run-001",
			ApproverID:  "user-manager-01",
			Action:      "tool:send_email",
			Payload:     json.RawMessage(`{"to":"customer@example.com"}`),
			Status:      "approved",
			ExpiresAt:   now.Add(24 * time.Hour),
			DecidedAt:   &decided,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}
}

// --- tests ---

func TestTraceBuilder_RequiredFields(t *testing.T) {
	t.Parallel()
	traceID := "trace-abc-001"
	store := &stubTraceStore{
		run:       makeRun(strPtr(traceID)),
		events:    makeAuditEvents(traceID),
		approvals: makeApprovals(),
	}

	builder := eval.NewTraceBuilder(store)
	trace, err := builder.Build(context.Background(), "run-001", "ws-001")
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	if trace.RunID == "" {
		t.Error("RunID must not be empty")
	}
	if trace.WorkspaceID == "" {
		t.Error("WorkspaceID must not be empty")
	}
	if trace.AgentDefinitionID == "" {
		t.Error("AgentDefinitionID must not be empty")
	}
	if trace.FinalOutcome == "" {
		t.Error("FinalOutcome must not be empty")
	}
	if trace.TriggerType == "" {
		t.Error("TriggerType must not be empty")
	}
}

func TestTraceBuilder_AuditEventsPopulated(t *testing.T) {
	t.Parallel()
	traceID := "trace-abc-002"
	store := &stubTraceStore{
		run:       makeRun(strPtr(traceID)),
		events:    makeAuditEvents(traceID),
		approvals: makeApprovals(),
	}

	trace, err := eval.NewTraceBuilder(store).Build(context.Background(), "run-001", "ws-001")
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	if len(trace.AuditEvents) != 3 {
		t.Errorf("expected 3 audit events, got %d", len(trace.AuditEvents))
	}
}

func TestTraceBuilder_PolicyDecisionsExtracted(t *testing.T) {
	t.Parallel()
	traceID := "trace-abc-003"
	store := &stubTraceStore{
		run:       makeRun(strPtr(traceID)),
		events:    makeAuditEvents(traceID),
		approvals: makeApprovals(),
	}

	trace, err := eval.NewTraceBuilder(store).Build(context.Background(), "run-001", "ws-001")
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	if len(trace.PolicyDecisions) == 0 {
		t.Error("expected at least one policy decision extracted from audit events")
	}
	pd := trace.PolicyDecisions[0]
	if pd.Action == "" {
		t.Error("PolicyDecision.Action must not be empty")
	}
	if pd.Outcome == "" {
		t.Error("PolicyDecision.Outcome must not be empty")
	}
}

func TestTraceBuilder_ApprovalEventsPopulated(t *testing.T) {
	t.Parallel()
	traceID := "trace-abc-004"
	store := &stubTraceStore{
		run:       makeRun(strPtr(traceID)),
		events:    makeAuditEvents(traceID),
		approvals: makeApprovals(),
	}

	trace, err := eval.NewTraceBuilder(store).Build(context.Background(), "run-001", "ws-001")
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	if len(trace.ApprovalEvents) == 0 {
		t.Error("expected at least one approval event")
	}
	ae := trace.ApprovalEvents[0]
	if ae.ApprovalID == "" {
		t.Error("ApprovalEvent.ApprovalID must not be empty")
	}
	if ae.Status == "" {
		t.Error("ApprovalEvent.Status must not be empty")
	}
}

func TestTraceBuilder_ToolCallsPopulated(t *testing.T) {
	t.Parallel()
	traceID := "trace-abc-005"
	store := &stubTraceStore{
		run:       makeRun(strPtr(traceID)),
		events:    makeAuditEvents(traceID),
		approvals: makeApprovals(),
	}

	trace, err := eval.NewTraceBuilder(store).Build(context.Background(), "run-001", "ws-001")
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	if len(trace.ToolCalls) == 0 {
		t.Error("expected at least one tool call")
	}
	tc := trace.ToolCalls[0]
	if tc.ToolName == "" {
		t.Error("ToolCall.ToolName must not be empty")
	}
	if tc.Status == "" {
		t.Error("ToolCall.Status must not be empty")
	}
}

func TestTraceBuilder_CostSignals(t *testing.T) {
	t.Parallel()
	traceID := "trace-abc-006"
	store := &stubTraceStore{
		run:       makeRun(strPtr(traceID)),
		events:    makeAuditEvents(traceID),
		approvals: []sqlcgen.ApprovalRequest{},
	}

	trace, err := eval.NewTraceBuilder(store).Build(context.Background(), "run-001", "ws-001")
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	if trace.TotalTokens == nil || *trace.TotalTokens != 512 {
		t.Errorf("expected TotalTokens=512, got %v", trace.TotalTokens)
	}
	if trace.TotalCost == nil || *trace.TotalCost != 0.007 {
		t.Errorf("expected TotalCost=0.007, got %v", trace.TotalCost)
	}
	if trace.LatencyMs == nil || *trace.LatencyMs != 480 {
		t.Errorf("expected LatencyMs=480, got %v", trace.LatencyMs)
	}
}

func TestTraceBuilder_EvidenceSources(t *testing.T) {
	t.Parallel()
	traceID := "trace-abc-007"
	store := &stubTraceStore{
		run:       makeRun(strPtr(traceID)),
		events:    makeAuditEvents(traceID),
		approvals: []sqlcgen.ApprovalRequest{},
	}

	trace, err := eval.NewTraceBuilder(store).Build(context.Background(), "run-001", "ws-001")
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	if len(trace.EvidenceSources) == 0 {
		t.Error("expected evidence sources from retrieved_evidence_ids")
	}
}

func TestTraceBuilder_NoTraceID_StillBuilds(t *testing.T) {
	t.Parallel()
	store := &stubTraceStore{
		run:       makeRun(nil), // no trace_id
		events:    []sqlcgen.AuditEvent{},
		approvals: []sqlcgen.ApprovalRequest{},
	}

	trace, err := eval.NewTraceBuilder(store).Build(context.Background(), "run-001", "ws-001")
	if err != nil {
		t.Fatalf("Build() unexpected error when trace_id is nil: %v", err)
	}
	if trace.RunID != "run-001" {
		t.Errorf("expected RunID=run-001, got %q", trace.RunID)
	}
	if len(trace.AuditEvents) != 0 {
		t.Error("expected no audit events when trace_id is nil")
	}
}

func TestTraceBuilder_ContractValidation_AllMutatorsTraceable(t *testing.T) {
	t.Parallel()
	traceID := "trace-abc-008"
	store := &stubTraceStore{
		run:       makeRun(strPtr(traceID)),
		events:    makeAuditEvents(traceID),
		approvals: makeApprovals(),
	}

	trace, err := eval.NewTraceBuilder(store).Build(context.Background(), "run-001", "ws-001")
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	// ContractValidation must be present and CheckedAt must be set.
	if trace.ContractValidation.CheckedAt.IsZero() {
		t.Error("ContractValidation.CheckedAt must be set")
	}

	// Every executed tool call must appear in audit events (traceability invariant).
	auditActions := make(map[string]struct{}, len(trace.AuditEvents))
	for _, ae := range trace.AuditEvents {
		auditActions[ae.Action] = struct{}{}
	}
	for _, tc := range trace.ToolCalls {
		if tc.Status == "executed" {
			if _, ok := auditActions["tool.executed"]; !ok {
				t.Errorf("executed tool %q has no corresponding audit event", tc.ToolName)
			}
		}
	}
}
