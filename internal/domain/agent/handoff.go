// Package agent — Handoff Manager.
// Task 3.8: Human handoff with evidence context (FR-232).
package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite/sqlcgen"
)

// ErrHandoffCaseNotFound is returned when the requested case does not exist.
var ErrHandoffCaseNotFound = errors.New("case not found for handoff")

// topicHandoff is the event bus topic published when a handoff is initiated.
const topicHandoff = "agent.handoff"
const handoffContractVersion = "v1"
const handoffPayloadEntityTypeKey = "entity_type"
const handoffPayloadEntityIDKey = "entity_id"

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
	ContractVersion   string              `json:"contractVersion"`
	RunID             string              `json:"runId"`
	WorkspaceID       string              `json:"workspaceId"`
	AgentDefinitionID string              `json:"agentDefinitionId"`
	Status            string              `json:"status"`
	RuntimeStatus     string              `json:"runtimeStatus,omitempty"`
	Reason            string              `json:"reason"`
	AbstentionReason  *string             `json:"abstentionReason,omitempty"`
	CaseID            string              `json:"caseId"`
	CaseSubject       string              `json:"caseSubject"`
	CaseStatus        string              `json:"caseStatus"`
	CasePriority      string              `json:"casePriority"`
	CaseOwnerID       string              `json:"caseOwnerId"`
	TriggerContext    json.RawMessage     `json:"triggerContext"`
	FinalOutput       json.RawMessage     `json:"finalOutput"`
	ReasoningTrace    json.RawMessage     `json:"reasoningTrace"`
	ToolCalls         json.RawMessage     `json:"toolCalls"`
	EvidenceIDs       json.RawMessage     `json:"evidenceIds"`
	EvidencePack      HandoffEvidencePack `json:"evidencePack"`
	StartedAt         time.Time           `json:"startedAt"`
	CompletedAt       *time.Time          `json:"completedAt,omitempty"`
}

type HandoffEvidencePack struct {
	SchemaVersion        string                  `json:"schema_version"`
	Query                string                  `json:"query"`
	Sources              []HandoffEvidenceSource `json:"sources"`
	SourceCount          int                     `json:"source_count"`
	DedupCount           int                     `json:"dedup_count"`
	FilteredCount        int                     `json:"filtered_count"`
	Confidence           string                  `json:"confidence"`
	Warnings             []string                `json:"warnings"`
	RetrievalMethodsUsed []string                `json:"retrieval_methods_used"`
	BuiltAt              string                  `json:"built_at"`
}

type HandoffEvidenceSource struct {
	EvidenceID      string  `json:"evidence_id"`
	KnowledgeItemID string  `json:"knowledge_item_id"`
	Method          string  `json:"method"`
	Score           float64 `json:"score"`
	Snippet         *string `json:"snippet,omitempty"`
	PiiRedacted     bool    `json:"pii_redacted"`
	Metadata        *string `json:"metadata,omitempty"`
	CreatedAt       string  `json:"created_at"`
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
	reason = resolveHandoffReason(reason, run)
	err = s.persistHandoffReason(ctx, workspaceID, runID, reason)
	if err != nil {
		return nil, err
	}
	run.AbstentionReason = stringPtr(reason)

	cs, err := s.loadAndEscalateCase(ctx, workspaceID, caseID, run)
	if err != nil {
		return nil, err
	}

	pkg := s.buildHandoffPackage(ctx, run, cs, reason)
	s.publishHandoffEvent(pkg)
	return pkg, nil
}

// GetHandoffPackage loads the handoff context for an escalated run (read-only, no side effects).
func (s *HandoffService) GetHandoffPackage(ctx context.Context, workspaceID, runID, caseID string) (*HandoffPackage, error) {
	run, err := s.loadRun(ctx, workspaceID, runID)
	if err != nil {
		return nil, err
	}

	caseID = resolveHandoffCaseID(run, caseID)
	cs, err := s.caseService.Get(ctx, workspaceID, caseID)
	if err != nil {
		return nil, ErrHandoffCaseNotFound
	}

	return s.buildHandoffPackage(ctx, run, cs, resolveHandoffReason("", run)), nil
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

func (s *HandoffService) persistHandoffReason(ctx context.Context, workspaceID, runID, reason string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE agent_run
		SET abstention_reason = COALESCE(?, abstention_reason), updated_at = datetime('now')
		WHERE id = ? AND workspace_id = ?
	`, nullableHandoffReason(reason), runID, workspaceID)
	return err
}

func resolveHandoffReason(requested string, run *Run) string {
	if requested != "" {
		return requested
	}
	if run.AbstentionReason != nil && *run.AbstentionReason != "" {
		return *run.AbstentionReason
	}
	return ""
}

func nullableHandoffReason(reason string) any {
	if reason == "" {
		return nil
	}
	return reason
}

func resolveHandoffCaseID(run *Run, requestedCaseID string) string {
	if strings.TrimSpace(requestedCaseID) != "" {
		return requestedCaseID
	}

	for _, payload := range []json.RawMessage{run.TriggerContext, run.Output, run.Inputs} {
		caseID := extractCaseIDFromPayload(payload)
		if caseID != "" {
			return caseID
		}
	}

	return ""
}

func extractCaseIDFromPayload(payload json.RawMessage) string {
	if len(payload) == 0 {
		return ""
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return ""
	}

	if caseID := firstNonEmptyString(decoded, "case_id", "caseId"); caseID != "" {
		return caseID
	}

	entityType := firstNonEmptyString(decoded, handoffPayloadEntityTypeKey, "entityType")
	entityID := firstNonEmptyString(decoded, handoffPayloadEntityIDKey, "entityId")
	if strings.TrimSpace(entityType) == bridgeEntityCase && strings.TrimSpace(entityID) != "" {
		return entityID
	}

	return ""
}

func firstNonEmptyString(decoded map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := decoded[key].(string)
		if ok && strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// buildHandoffPackage assembles a HandoffPackage from a Run and a CaseTicket.
// Extracted as a standalone helper to keep InitiateHandoff and GetHandoffPackage
// within cyclomatic complexity ≤ 4 each.
func (s *HandoffService) buildHandoffPackage(ctx context.Context, run *Run, cs *crm.CaseTicket, reason string) *HandoffPackage {
	evidencePack := s.reconstructEvidencePack(ctx, run)
	return &HandoffPackage{
		ContractVersion:   handoffContractVersion,
		RunID:             run.ID,
		WorkspaceID:       run.WorkspaceID,
		AgentDefinitionID: run.DefinitionID,
		Status:            PublicRunOutcome(run),
		RuntimeStatus:     run.Status,
		Reason:            reason,
		AbstentionReason:  run.AbstentionReason,
		CaseID:            cs.ID,
		CaseSubject:       cs.Subject,
		CaseStatus:        cs.Status,
		CasePriority:      cs.Priority,
		CaseOwnerID:       cs.OwnerID,
		TriggerContext:    run.TriggerContext,
		FinalOutput:       run.Output,
		ReasoningTrace:    run.ReasoningTrace,
		ToolCalls:         run.ToolCalls,
		EvidenceIDs:       run.RetrievedEvidenceIDs,
		EvidencePack:      evidencePack,
		StartedAt:         run.StartedAt,
		CompletedAt:       run.CompletedAt,
	}
}

func (s *HandoffService) reconstructEvidencePack(ctx context.Context, run *Run) HandoffEvidencePack {
	query := firstHandoffStringArrayValue(run.RetrievalQueries)
	sources := s.loadHandoffEvidenceSources(ctx, run.WorkspaceID, handoffEvidenceIDs(run.RetrievedEvidenceIDs))
	return newHandoffEvidencePack(query, sources, run.StartedAt)
}

func (s *HandoffService) loadHandoffEvidenceSources(ctx context.Context, workspaceID string, evidenceIDs []string) []knowledge.Evidence {
	if len(evidenceIDs) == 0 {
		return []knowledge.Evidence{}
	}
	q := sqlcgen.New(s.db)
	sources := make([]knowledge.Evidence, 0, len(evidenceIDs))
	for _, evidenceID := range evidenceIDs {
		row, err := q.GetEvidenceByID(ctx, sqlcgen.GetEvidenceByIDParams{ID: evidenceID, WorkspaceID: workspaceID})
		if err != nil {
			continue
		}
		sources = append(sources, knowledge.Evidence{
			ID:              row.ID,
			KnowledgeItemID: row.KnowledgeItemID,
			WorkspaceID:     row.WorkspaceID,
			Method:          knowledge.EvidenceMethod(row.Method),
			Score:           row.Score,
			Snippet:         row.Snippet,
			PiiRedacted:     row.PiiRedacted,
			Metadata:        row.Metadata,
			CreatedAt:       row.CreatedAt,
		})
	}
	return sources
}

func newHandoffEvidencePack(query string, sources []knowledge.Evidence, fallbackBuiltAt time.Time) HandoffEvidencePack {
	methods := make([]string, 0, len(sources))
	seenMethods := make(map[string]struct{}, len(sources))
	builtAt := fallbackBuiltAt.UTC()
	for _, source := range sources {
		method := string(source.Method)
		if _, ok := seenMethods[method]; !ok && method != "" {
			seenMethods[method] = struct{}{}
			methods = append(methods, method)
		}
		if source.CreatedAt.After(builtAt) {
			builtAt = source.CreatedAt.UTC()
		}
	}
	return HandoffEvidencePack{
		SchemaVersion:        knowledge.EvidencePackSchemaVersion,
		Query:                query,
		Sources:              handoffEvidenceSources(sources),
		SourceCount:          len(sources),
		DedupCount:           0,
		FilteredCount:        0,
		Confidence:           string(handoffEvidenceConfidence(sources)),
		Warnings:             []string{},
		RetrievalMethodsUsed: methods,
		BuiltAt:              builtAt.Format(time.RFC3339),
	}
}

func handoffEvidenceSources(sources []knowledge.Evidence) []HandoffEvidenceSource {
	items := make([]HandoffEvidenceSource, 0, len(sources))
	for _, source := range sources {
		items = append(items, HandoffEvidenceSource{
			EvidenceID:      source.ID,
			KnowledgeItemID: source.KnowledgeItemID,
			Method:          string(source.Method),
			Score:           source.Score,
			Snippet:         source.Snippet,
			PiiRedacted:     source.PiiRedacted,
			Metadata:        source.Metadata,
			CreatedAt:       source.CreatedAt.Format(time.RFC3339),
		})
	}
	return items
}

func handoffEvidenceIDs(raw json.RawMessage) []string {
	var ids []string
	if len(raw) == 0 || json.Unmarshal(raw, &ids) != nil {
		return []string{}
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if trimmed := strings.TrimSpace(id); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func firstHandoffStringArrayValue(raw json.RawMessage) string {
	values := handoffEvidenceIDs(raw)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func handoffEvidenceConfidence(sources []knowledge.Evidence) knowledge.ConfidenceLevel {
	if len(sources) == 0 {
		return knowledge.ConfidenceLow
	}
	topScore := sources[0].Score
	switch {
	case topScore >= 0.8:
		return knowledge.ConfidenceHigh
	case topScore >= 0.5:
		return knowledge.ConfidenceMedium
	default:
		return knowledge.ConfidenceLow
	}
}
