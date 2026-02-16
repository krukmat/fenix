// Package agent — Handoff Manager.
// Task 3.8: Human handoff with evidence context (FR-232).
package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
)

// ErrHandoffCaseNotFound is returned when the requested case does not exist.
var ErrHandoffCaseNotFound = errors.New("case not found for handoff")

// topicHandoff is the event bus topic published when a handoff is initiated.
const topicHandoff = "agent.handoff"

// CaseServiceInterface allows HandoffService to load and update cases
// without creating a circular import between domain/agent and domain/crm.
// crm.CaseService satisfies this interface at the routes.go wiring layer.
type CaseServiceInterface interface {
	Get(ctx context.Context, workspaceID, caseID string) (*crm.CaseTicket, error)
	Update(ctx context.Context, workspaceID, caseID string, input crm.UpdateCaseInput) (*crm.CaseTicket, error)
}

// HandoffPackage is the structured context delivered to a human agent
// when an AI agent cannot resolve a case and escalates.
type HandoffPackage struct {
	RunID             string          `json:"runId"`
	WorkspaceID       string          `json:"workspaceId"`
	AgentDefinitionID string          `json:"agentDefinitionId"`
	Status            string          `json:"status"`
	Reason            string          `json:"reason"`
	CaseID            string          `json:"caseId"`
	CaseSubject       string          `json:"caseSubject"`
	CaseStatus        string          `json:"caseStatus"`
	ReasoningTrace    json.RawMessage `json:"reasoningTrace"`
	ToolCalls         json.RawMessage `json:"toolCalls"`
	EvidenceIDs       json.RawMessage `json:"evidenceIds"`
	StartedAt         time.Time       `json:"startedAt"`
	CompletedAt       *time.Time      `json:"completedAt,omitempty"`
}

// HandoffService handles agent-to-human escalation packaging.
type HandoffService struct {
	db          *sql.DB
	caseService CaseServiceInterface
	bus         eventbus.EventBus
}

// NewHandoffService creates a new HandoffService.
func NewHandoffService(db *sql.DB, cs CaseServiceInterface, bus eventbus.EventBus) *HandoffService {
	return &HandoffService{db: db, caseService: cs, bus: bus}
}

// InitiateHandoff builds the handoff package, updates the case status to "escalated",
// and publishes an agent.handoff event.
func (s *HandoffService) InitiateHandoff(ctx context.Context, workspaceID, runID, caseID, reason string) (*HandoffPackage, error) {
	run, err := s.loadRun(ctx, workspaceID, runID)
	if err != nil {
		return nil, err
	}

	cs, err := s.loadAndEscalateCase(ctx, workspaceID, caseID, run)
	if err != nil {
		return nil, err
	}

	pkg := buildHandoffPackage(run, cs, reason)
	s.publishHandoffEvent(pkg)
	return pkg, nil
}

// GetHandoffPackage loads the handoff context for an escalated run (read-only, no side effects).
func (s *HandoffService) GetHandoffPackage(ctx context.Context, workspaceID, runID, caseID string) (*HandoffPackage, error) {
	run, err := s.loadRun(ctx, workspaceID, runID)
	if err != nil {
		return nil, err
	}

	cs, err := s.caseService.Get(ctx, workspaceID, caseID)
	if err != nil {
		return nil, ErrHandoffCaseNotFound
	}

	return buildHandoffPackage(run, cs, ""), nil
}

// ── Private helpers ──────────────────────────────────────────────────────────

// loadRun fetches the agent_run by ID and workspace, returning ErrAgentRunNotFound when missing.
func (s *HandoffService) loadRun(ctx context.Context, workspaceID, runID string) (*Run, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, agent_definition_id, triggered_by_user_id,
		       trigger_type, trigger_context, status, inputs,
		       retrieval_queries, retrieved_evidence_ids, reasoning_trace,
		       tool_calls, output, abstention_reason,
		       total_tokens, total_cost, latency_ms, trace_id,
		       started_at, completed_at, created_at
		FROM agent_run
		WHERE id = ? AND workspace_id = ?
	`, runID, workspaceID)

	run, err := scanAgentRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAgentRunNotFound
	}
	return run, err
}

// loadAndEscalateCase fetches the case and updates its status to "escalated".
func (s *HandoffService) loadAndEscalateCase(ctx context.Context, workspaceID, caseID string, _ *Run) (*crm.CaseTicket, error) {
	existing, err := s.caseService.Get(ctx, workspaceID, caseID)
	if err != nil {
		return nil, ErrHandoffCaseNotFound
	}

	updated, err := s.caseService.Update(ctx, workspaceID, caseID, crm.UpdateCaseInput{
		OwnerID:  existing.OwnerID,
		Subject:  existing.Subject,
		Priority: existing.Priority,
		Status:   StatusEscalated,
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// publishHandoffEvent emits the agent.handoff event on the bus (nil-safe).
func (s *HandoffService) publishHandoffEvent(pkg *HandoffPackage) {
	if s.bus == nil {
		return
	}
	s.bus.Publish(topicHandoff, pkg)
}

// buildHandoffPackage assembles a HandoffPackage from a Run and a CaseTicket.
// Extracted as a standalone helper to keep InitiateHandoff and GetHandoffPackage
// within cyclomatic complexity ≤ 4 each.
func buildHandoffPackage(run *Run, cs *crm.CaseTicket, reason string) *HandoffPackage {
	return &HandoffPackage{
		RunID:             run.ID,
		WorkspaceID:       run.WorkspaceID,
		AgentDefinitionID: run.DefinitionID,
		Status:            run.Status,
		Reason:            reason,
		CaseID:            cs.ID,
		CaseSubject:       cs.Subject,
		CaseStatus:        cs.Status,
		ReasoningTrace:    run.ReasoningTrace,
		ToolCalls:         run.ToolCalls,
		EvidenceIDs:       run.RetrievedEvidenceIDs,
		StartedAt:         run.StartedAt,
		CompletedAt:       run.CompletedAt,
	}
}
