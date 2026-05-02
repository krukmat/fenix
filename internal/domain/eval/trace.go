package eval

// F2-T1: ActualRunTrace DTO and TraceBuilder — read-side enrichment.
// Assembles what the agent actually did from existing DB joins.
// No modification to agent_run schema or orchestrator.go.

import (
	"context"
	"encoding/json"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite/sqlcgen"
)

// TraceStore is the read-side interface required by TraceBuilder.
// Allows deterministic testing via stub implementations.
type TraceStore interface {
	GetAgentRunByID(ctx context.Context, arg sqlcgen.GetAgentRunByIDParams) (sqlcgen.AgentRun, error)
	ListAuditEventsByTraceID(ctx context.Context, traceID *string) ([]sqlcgen.AuditEvent, error)
	ListApprovalRequestsByIDs(ctx context.Context, ids []string) ([]sqlcgen.ApprovalRequest, error)
}

// ActualRunTrace is the fully enriched read-side record of a single agent execution.
// All fields are derived from existing DB tables — no schema mutations.
type ActualRunTrace struct {
	// Identity
	RunID             string `json:"run_id"`
	WorkspaceID       string `json:"workspace_id"`
	AgentDefinitionID string `json:"agent_definition_id"`
	ScenarioID        string `json:"scenario_id,omitempty"` // when matched against a GoldenScenario

	// Trigger
	TriggerType   string          `json:"trigger_type"`
	InputEvent    json.RawMessage `json:"input_event"`    // from trigger_context
	ContextInputs json.RawMessage `json:"context_inputs"` // from inputs

	// Retrieval
	RetrievalQueries []string `json:"retrieval_queries"`
	EvidenceSources  []string `json:"evidence_sources"` // retrieved_evidence_ids

	// Policy
	PolicyDecisions []TracePolicyDecision `json:"policy_decisions"`

	// Approvals
	ApprovalEvents []TraceApprovalEvent `json:"approval_events"`

	// Tool calls
	ToolCalls []TraceToolCall `json:"tool_calls"`

	// Audit trail
	AuditEvents []TraceAuditEvent `json:"audit_events"`

	// Outcome
	FinalOutcome     string          `json:"final_outcome"` // maps from agent_run.status
	Output           json.RawMessage `json:"output"`
	AbstentionReason *string         `json:"abstention_reason,omitempty"`
	ReasoningTrace   json.RawMessage `json:"reasoning_trace"`

	// Performance / cost signals
	LatencyMs   *int64   `json:"latency_ms,omitempty"`
	TotalTokens *int64   `json:"total_tokens,omitempty"`
	TotalCost   *float64 `json:"total_cost,omitempty"`
	Retries     int      `json:"retries"`

	// Timestamps
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Contract validation results (populated by TraceBuilder)
	ContractValidation TraceContractValidation `json:"contract_validation"`

	// FinalStateRaw is the observed final state of entities after the run.
	// Populated externally (e.g. by the eval runner); compared against GoldenScenario.Expected.FinalState.
	FinalStateRaw json.RawMessage `json:"final_state,omitempty"`
}

// TracePolicyDecision captures a single policy evaluation extracted from audit events.
type TracePolicyDecision struct {
	Action  string `json:"action"`  // e.g. "tool:send_email"
	Outcome string `json:"outcome"` // "allow" | "deny" | "require_approval"
}

// TraceApprovalEvent captures an approval request linked to the run.
type TraceApprovalEvent struct {
	ApprovalID string     `json:"approval_id"`
	Action     string     `json:"action"`
	Status     string     `json:"status"` // "pending" | "approved" | "rejected"
	DecidedAt  *time.Time `json:"decided_at,omitempty"`
}

// TraceToolCall captures a single tool invocation from agent_run.tool_calls.
type TraceToolCall struct {
	ToolName string          `json:"tool_name"`
	Status   string          `json:"status"` // "executed" | "blocked" | "attempted"
	Params   json.RawMessage `json:"params,omitempty"`
}

// TraceAuditEvent is a minimal projection of sqlcgen.AuditEvent for the trace.
type TraceAuditEvent struct {
	ID       string    `json:"id"`
	Action   string    `json:"action"`
	Outcome  string    `json:"outcome"`
	ActorID  string    `json:"actor_id"`
	EntityID *string   `json:"entity_id,omitempty"`
	At       time.Time `json:"at"`
}

// TraceContractValidation records the invariant checks performed at build time.
type TraceContractValidation struct {
	CheckedAt         time.Time `json:"checked_at"`
	MutatorsTraceable bool      `json:"mutators_traceable"` // all executed tools have audit events
	PolicysTraceable  bool      `json:"policys_traceable"`  // all policy decisions have audit trail
}

// TraceBuilder assembles an ActualRunTrace from existing DB joins.
// F2-T1: read-side only, no writes.
type TraceBuilder struct {
	store TraceStore
}

// NewTraceBuilder constructs a TraceBuilder backed by the given store.
func NewTraceBuilder(store TraceStore) *TraceBuilder {
	return &TraceBuilder{store: store}
}

// Build fetches and joins all data for the given run, returning the enriched trace.
func (b *TraceBuilder) Build(ctx context.Context, runID, workspaceID string) (*ActualRunTrace, error) {
	run, err := b.store.GetAgentRunByID(ctx, sqlcgen.GetAgentRunByIDParams{ID: runID, WorkspaceID: workspaceID})
	if err != nil {
		return nil, err
	}

	events, err := b.store.ListAuditEventsByTraceID(ctx, run.TraceID)
	if err != nil {
		return nil, err
	}

	approvalIDs := extractApprovalIDs(events)
	var approvals []sqlcgen.ApprovalRequest
	if len(approvalIDs) > 0 {
		approvals, err = b.store.ListApprovalRequestsByIDs(ctx, approvalIDs)
		if err != nil {
			return nil, err
		}
	}

	trace := &ActualRunTrace{
		RunID:             run.ID,
		WorkspaceID:       run.WorkspaceID,
		AgentDefinitionID: run.AgentDefinitionID,
		TriggerType:       run.TriggerType,
		InputEvent:        run.TriggerContext,
		ContextInputs:     run.Inputs,
		RetrievalQueries:  parseStringSlice(run.RetrievalQueries),
		EvidenceSources:   parseStringSlice(run.RetrievedEvidenceIds),
		FinalOutcome:      run.Status,
		Output:            run.Output,
		AbstentionReason:  run.AbstentionReason,
		ReasoningTrace:    run.ReasoningTrace,
		LatencyMs:         run.LatencyMs,
		TotalTokens:       run.TotalTokens,
		TotalCost:         run.TotalCost,
		StartedAt:         run.StartedAt,
		CompletedAt:       run.CompletedAt,
		ToolCalls:         parseToolCalls(run.ToolCalls),
		AuditEvents:       projectAuditEvents(events),
		PolicyDecisions:   extractPolicyDecisions(events),
		ApprovalEvents:    buildApprovalEvents(approvals),
	}

	trace.ContractValidation = validateContract(trace)
	return trace, nil
}

// parseStringSlice unmarshals a JSON array of strings; returns empty slice on failure.
func parseStringSlice(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

// rawToolCall is the shape stored in agent_run.tool_calls.
type rawToolCall struct {
	Tool   string          `json:"tool"`
	Status string          `json:"status"`
	Params json.RawMessage `json:"params,omitempty"`
}

// parseToolCalls deserializes agent_run.tool_calls into TraceToolCall slice.
func parseToolCalls(raw json.RawMessage) []TraceToolCall {
	if len(raw) == 0 {
		return nil
	}
	var items []rawToolCall
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil
	}
	out := make([]TraceToolCall, 0, len(items))
	for _, item := range items {
		out = append(out, TraceToolCall{
			ToolName: item.Tool,
			Status:   item.Status,
			Params:   item.Params,
		})
	}
	return out
}

// projectAuditEvents converts sqlcgen audit events to trace projections.
func projectAuditEvents(events []sqlcgen.AuditEvent) []TraceAuditEvent {
	out := make([]TraceAuditEvent, 0, len(events))
	for _, e := range events {
		out = append(out, TraceAuditEvent{
			ID:       e.ID,
			Action:   e.Action,
			Outcome:  e.Outcome,
			ActorID:  e.ActorID,
			EntityID: e.EntityID,
			At:       e.CreatedAt,
		})
	}
	return out
}

// policyDetailsPayload is the shape stored in audit_event.details for policy.decision events.
type policyDetailsPayload struct {
	Action  string `json:"action"`
	Outcome string `json:"outcome"`
}

// extractPolicyDecisions scans audit events for action=="policy.decision" and parses details.
func extractPolicyDecisions(events []sqlcgen.AuditEvent) []TracePolicyDecision {
	out := make([]TracePolicyDecision, 0, len(events))
	for _, e := range events {
		if e.Action != "policy.decision" {
			continue
		}
		var p policyDetailsPayload
		if err := json.Unmarshal(e.Details, &p); err != nil || p.Action == "" {
			continue
		}
		out = append(out, TracePolicyDecision(p))
	}
	return out
}

// approvalDetailsPayload is the shape in audit_event.details for events that reference an approval.
type approvalDetailsPayload struct {
	ApprovalID string `json:"approval_id"`
}

// extractApprovalIDs collects approval_id values from audit_event.details.
func extractApprovalIDs(events []sqlcgen.AuditEvent) []string {
	seen := make(map[string]struct{})
	var ids []string
	for _, e := range events {
		var p approvalDetailsPayload
		if err := json.Unmarshal(e.Details, &p); err != nil || p.ApprovalID == "" {
			continue
		}
		if _, ok := seen[p.ApprovalID]; !ok {
			seen[p.ApprovalID] = struct{}{}
			ids = append(ids, p.ApprovalID)
		}
	}
	return ids
}

// buildApprovalEvents converts sqlcgen approval requests to trace events.
func buildApprovalEvents(approvals []sqlcgen.ApprovalRequest) []TraceApprovalEvent {
	out := make([]TraceApprovalEvent, 0, len(approvals))
	for _, a := range approvals {
		out = append(out, TraceApprovalEvent{
			ApprovalID: a.ID,
			Action:     a.Action,
			Status:     a.Status,
			DecidedAt:  a.DecidedAt,
		})
	}
	return out
}

// validateContract checks traceability invariants and records the result.
func validateContract(t *ActualRunTrace) TraceContractValidation {
	auditActions := make(map[string]struct{}, len(t.AuditEvents))
	for _, ae := range t.AuditEvents {
		auditActions[ae.Action] = struct{}{}
	}

	mutatorsTraceable := true
	for _, tc := range t.ToolCalls {
		if tc.Status == "executed" {
			if _, ok := auditActions["tool.executed"]; !ok {
				mutatorsTraceable = false
				break
			}
		}
	}

	policysTraceable := len(t.PolicyDecisions) == 0 || len(t.AuditEvents) > 0

	return TraceContractValidation{
		CheckedAt:         time.Now().UTC(),
		MutatorsTraceable: mutatorsTraceable,
		PolicysTraceable:  policysTraceable,
	}
}
