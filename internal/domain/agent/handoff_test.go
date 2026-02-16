// Package agent — Handoff Manager tests.
// Task 3.8: Human handoff with evidence context.
package agent

import (
	"context"
	"database/sql"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
)

// ── Test helpers ────────────────────────────────────────────────────────────

// insertHandoffTestAgentDef inserts a minimal agent_definition row for handoff tests.
func insertHandoffTestAgentDef(t *testing.T, db *sql.DB, id, workspaceID string) {
	t.Helper()
	ctx := context.Background()
	_, err := db.ExecContext(ctx,
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES (?, ?, 'Handoff Test Agent', 'support', 'active')`,
		id, workspaceID,
	)
	if err != nil {
		t.Fatalf("insertHandoffTestAgentDef: %v", err)
	}
}

// insertHandoffTestRun inserts an agent_run row with status=escalated.
func insertHandoffTestRun(t *testing.T, db *sql.DB, runID, workspaceID, agentDefID string) {
	t.Helper()
	ctx := context.Background()
	_, err := db.ExecContext(ctx, `
		INSERT INTO agent_run (
			id, workspace_id, agent_definition_id, trigger_type, status,
			retrieval_queries, retrieved_evidence_ids, reasoning_trace, tool_calls,
			output, started_at, created_at
		) VALUES (
			?, ?, ?, 'manual', 'escalated',
			'["q1"]', '["ev1"]', '[{"step":"think"}]', '[{"tool":"search"}]',
			'{}', datetime('now'), datetime('now')
		)`, runID, workspaceID, agentDefID)
	if err != nil {
		t.Fatalf("insertHandoffTestRun: %v", err)
	}
}

// insertHandoffTestCase inserts a minimal case_ticket row.
func insertHandoffTestCase(t *testing.T, db *sql.DB, caseID, workspaceID string) {
	t.Helper()
	ctx := context.Background()
	_, err := db.ExecContext(ctx, `
		INSERT INTO case_ticket (id, workspace_id, owner_id, subject, priority, status, created_at, updated_at)
		VALUES (?, ?, 'user-1', 'Test Case Subject', 'medium', 'open', datetime('now'), datetime('now'))
		`, caseID, workspaceID)
	if err != nil {
		t.Fatalf("insertHandoffTestCase: %v", err)
	}
}

// newHandoffSvc creates a HandoffService backed by a real DB (no mocks).
func newHandoffSvc(t *testing.T) (*HandoffService, *sql.DB) {
	t.Helper()
	db := setupTestDB(t)
	cs := crm.NewCaseService(db)
	bus := eventbus.New()
	svc := NewHandoffService(db, cs, bus)
	return svc, db
}

// ── Tests ────────────────────────────────────────────────────────────────────

// TestInitiateHandoff_Success verifies the happy path: run escalated, case updated, package returned.
// Traces: FR-232
func TestInitiateHandoff_Success(t *testing.T) {
	svc, db := newHandoffSvc(t)
	defer db.Close()

	ctx := context.Background()
	const wsID = "ws-handoff-1"
	const runID = "run-handoff-1"
	const agentDefID = "agent-handoff-1"
	const caseID = "case-handoff-1"

	insertHandoffTestAgentDef(t, db, agentDefID, wsID)
	insertHandoffTestRun(t, db, runID, wsID, agentDefID)
	insertHandoffTestCase(t, db, caseID, wsID)

	pkg, err := svc.InitiateHandoff(ctx, wsID, runID, caseID, "no solution found")
	if err != nil {
		t.Fatalf("InitiateHandoff: %v", err)
	}

	if pkg.RunID != runID {
		t.Errorf("RunID: got %q, want %q", pkg.RunID, runID)
	}
	if pkg.WorkspaceID != wsID {
		t.Errorf("WorkspaceID: got %q, want %q", pkg.WorkspaceID, wsID)
	}
	if pkg.CaseID != caseID {
		t.Errorf("CaseID: got %q, want %q", pkg.CaseID, caseID)
	}
	if pkg.CaseSubject != "Test Case Subject" {
		t.Errorf("CaseSubject: got %q, want %q", pkg.CaseSubject, "Test Case Subject")
	}
	if pkg.CaseStatus != StatusEscalated {
		t.Errorf("CaseStatus: got %q, want %q", pkg.CaseStatus, StatusEscalated)
	}
	if pkg.Reason != "no solution found" {
		t.Errorf("Reason: got %q, want %q", pkg.Reason, "no solution found")
	}
}

// TestInitiateHandoff_RunNotFound returns ErrAgentRunNotFound for unknown run.
// Traces: FR-232
func TestInitiateHandoff_RunNotFound(t *testing.T) {
	svc, db := newHandoffSvc(t)
	defer db.Close()

	_, err := svc.InitiateHandoff(context.Background(), "ws-1", "nonexistent-run", "case-1", "reason")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != ErrAgentRunNotFound {
		t.Errorf("expected ErrAgentRunNotFound, got: %v", err)
	}
}

// TestInitiateHandoff_CaseNotFound returns ErrHandoffCaseNotFound when the case doesn't exist.
// Traces: FR-232
func TestInitiateHandoff_CaseNotFound(t *testing.T) {
	svc, db := newHandoffSvc(t)
	defer db.Close()

	ctx := context.Background()
	const wsID = "ws-handoff-2"
	const runID = "run-handoff-2"
	const agentDefID = "agent-handoff-2"

	insertHandoffTestAgentDef(t, db, agentDefID, wsID)
	insertHandoffTestRun(t, db, runID, wsID, agentDefID)
	// Do NOT insert case

	_, err := svc.InitiateHandoff(ctx, wsID, runID, "nonexistent-case", "reason")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != ErrHandoffCaseNotFound {
		t.Errorf("expected ErrHandoffCaseNotFound, got: %v", err)
	}
}

// TestGetHandoffPackage_ContainsAllContext verifies all fields are populated correctly.
// Traces: FR-232
func TestGetHandoffPackage_ContainsAllContext(t *testing.T) {
	svc, db := newHandoffSvc(t)
	defer db.Close()

	ctx := context.Background()
	const wsID = "ws-handoff-3"
	const runID = "run-handoff-3"
	const agentDefID = "agent-handoff-3"
	const caseID = "case-handoff-3"

	insertHandoffTestAgentDef(t, db, agentDefID, wsID)
	insertHandoffTestRun(t, db, runID, wsID, agentDefID)
	insertHandoffTestCase(t, db, caseID, wsID)

	pkg, err := svc.GetHandoffPackage(ctx, wsID, runID, caseID)
	if err != nil {
		t.Fatalf("GetHandoffPackage: %v", err)
	}

	if pkg.RunID != runID {
		t.Errorf("RunID: got %q, want %q", pkg.RunID, runID)
	}
	if len(pkg.ReasoningTrace) == 0 {
		t.Error("ReasoningTrace should not be empty")
	}
	if len(pkg.ToolCalls) == 0 {
		t.Error("ToolCalls should not be empty")
	}
	if len(pkg.EvidenceIDs) == 0 {
		t.Error("EvidenceIDs should not be empty")
	}
	if pkg.CaseSubject != "Test Case Subject" {
		t.Errorf("CaseSubject: got %q, want %q", pkg.CaseSubject, "Test Case Subject")
	}
}

// TestInitiateHandoff_EmitsEvent verifies the agent.handoff event is published.
// Traces: FR-232
func TestInitiateHandoff_EmitsEvent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	const wsID = "ws-handoff-4"
	const runID = "run-handoff-4"
	const agentDefID = "agent-handoff-4"
	const caseID = "case-handoff-4"

	insertHandoffTestAgentDef(t, db, agentDefID, wsID)
	insertHandoffTestRun(t, db, runID, wsID, agentDefID)
	insertHandoffTestCase(t, db, caseID, wsID)

	bus := eventbus.New()
	sub := bus.Subscribe("agent.handoff")

	cs := crm.NewCaseService(db)
	svc := NewHandoffService(db, cs, bus)

	_, err := svc.InitiateHandoff(ctx, wsID, runID, caseID, "escalating")
	if err != nil {
		t.Fatalf("InitiateHandoff: %v", err)
	}

	// Verify event was published
	select {
	case evt := <-sub:
		if evt.Topic != "agent.handoff" {
			t.Errorf("expected topic agent.handoff, got %s", evt.Topic)
		}
	default:
		t.Error("expected agent.handoff event to be published, but channel was empty")
	}
}
