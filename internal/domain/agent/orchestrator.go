// Package agent provides the Agent Orchestrator and related functionality.
// Task 3.7: Agent Runtime - Orchestrator + Support Agent UC-C1
package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

var (
	ErrAgentNotFound       = errors.New("agent definition not found")
	ErrAgentRunNotFound    = errors.New("agent run not found")
	ErrAgentNotActive      = errors.New("agent is not active")
	ErrInvalidTriggerType  = errors.New("invalid trigger type")
	ErrAgentAlreadyRunning = errors.New("agent run already in progress")
)

// Agent status constants
const (
	StatusRunning   = "running"
	StatusSuccess   = "success"
	StatusPartial   = "partial"
	StatusAbstained = "abstained"
	StatusFailed    = "failed"
	StatusEscalated = "escalated"
)

// emptyJSONArray is the default value for JSON array fields in a new agent run.
const emptyJSONArray = `[]`

// Trigger type constants
const (
	TriggerTypeEvent    = "event"
	TriggerTypeSchedule = "schedule"
	TriggerTypeManual   = "manual"
	TriggerTypeCopilot  = "copilot"
)

// Domain models

// Definition defines an agent (formerly AgentDefinition — renamed to avoid agent.AgentDefinition stutter)
type Definition struct {
	ID                    string
	WorkspaceID           string
	Name                  string
	Description           *string
	AgentType             string
	Objective             json.RawMessage
	AllowedTools          []string
	Limits                map[string]any
	TriggerConfig         map[string]any
	PolicySetID           *string
	ActivePromptVersionID *string
	Status                string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// Run holds the state of an agent execution (formerly AgentRun — renamed to avoid agent.AgentRun stutter)
type Run struct {
	ID                   string
	WorkspaceID          string
	DefinitionID         string
	TriggeredByUserID    *string
	TriggerType          string
	TriggerContext       json.RawMessage
	Status               string
	Inputs               json.RawMessage
	RetrievalQueries     json.RawMessage
	RetrievedEvidenceIDs json.RawMessage
	ReasoningTrace       json.RawMessage
	ToolCalls            json.RawMessage
	Output               json.RawMessage
	AbstentionReason     *string
	TotalTokens          *int64
	TotalCost            *float64
	LatencyMs            *int64
	TraceID              *string
	StartedAt            time.Time
	CompletedAt          *time.Time
	CreatedAt            time.Time
}

type SkillDefinition struct {
	ID           string
	WorkspaceID  string
	Name         string
	Description  *string
	Steps        json.RawMessage
	DefinitionID *string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Input/Output types

type TriggerAgentInput struct {
	AgentID        string
	WorkspaceID    string
	TriggeredBy    *string
	TriggerType    string
	TriggerContext json.RawMessage
	Inputs         json.RawMessage
}

type ToolCall struct {
	ToolName   string          `json:"tool_name"`
	Params     json.RawMessage `json:"params"`
	Result     json.RawMessage `json:"result,omitempty"`
	Error      string          `json:"error,omitempty"`
	ExecutedAt *time.Time      `json:"executed_at,omitempty"`
}

// Orchestrator service

type Orchestrator struct {
	db *sql.DB
}

func NewOrchestrator(db *sql.DB) *Orchestrator {
	return &Orchestrator{db: db}
}

// TriggerAgent creates a new agent run and returns it
func (o *Orchestrator) TriggerAgent(ctx context.Context, in TriggerAgentInput) (*Run, error) {
	// Validate trigger type
	if !isValidTriggerType(in.TriggerType) {
		return nil, ErrInvalidTriggerType
	}

	// Get agent definition
	agent, err := o.getAgentDefinition(ctx, in.AgentID, in.WorkspaceID)
	if err != nil {
		return nil, err
	}

	// Check agent is active
	if agent.Status != "active" {
		return nil, ErrAgentNotActive
	}

	// Create agent run
	run := &Run{
		ID:                   uuid.NewV7().String(),
		WorkspaceID:          in.WorkspaceID,
		DefinitionID:         in.AgentID,
		TriggeredByUserID:    in.TriggeredBy,
		TriggerType:          in.TriggerType,
		TriggerContext:       in.TriggerContext,
		Status:               StatusRunning,
		Inputs:               in.Inputs,
		RetrievalQueries:     json.RawMessage(emptyJSONArray),
		RetrievedEvidenceIDs: json.RawMessage(emptyJSONArray),
		ReasoningTrace:       json.RawMessage(emptyJSONArray),
		ToolCalls:            json.RawMessage(emptyJSONArray),
		Output:               json.RawMessage(`{}`),
		TraceID:              stringPtr(uuid.NewV7().String()),
		StartedAt:            time.Now().UTC(),
		CreatedAt:            time.Now().UTC(),
	}

	_, err = o.db.ExecContext(ctx, `
		INSERT INTO agent_run (
			id, workspace_id, agent_definition_id, triggered_by_user_id,
			trigger_type, trigger_context, status, inputs,
			retrieval_queries, retrieved_evidence_ids, reasoning_trace,
			tool_calls, output, abstention_reason,
			total_tokens, total_cost, latency_ms, trace_id,
			started_at, completed_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, ?)
	`,
		run.ID,
		run.WorkspaceID,
		run.DefinitionID,
		run.TriggeredByUserID,
		run.TriggerType,
		run.TriggerContext,
		run.Status,
		run.Inputs,
		run.RetrievalQueries,
		run.RetrievedEvidenceIDs,
		run.ReasoningTrace,
		run.ToolCalls,
		run.Output,
		run.AbstentionReason,
		run.TotalTokens,
		run.TotalCost,
		run.LatencyMs,
		run.TraceID,
		run.StartedAt,
		run.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return run, nil
}

// GetAgentRun retrieves an agent run by ID
func (o *Orchestrator) GetAgentRun(ctx context.Context, workspaceID, runID string) (*Run, error) {
	row := o.db.QueryRowContext(ctx, `
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
	if err != nil {
		return nil, err
	}
	return run, nil
}

// ListAgentRuns lists agent runs with pagination
func (o *Orchestrator) ListAgentRuns(ctx context.Context, workspaceID string, limit, offset int64) ([]*Run, int64, error) {
	if limit <= 0 {
		limit = 25
	}

	rows, err := o.db.QueryContext(ctx, `
		SELECT id, workspace_id, agent_definition_id, triggered_by_user_id,
		       trigger_type, trigger_context, status, inputs,
		       retrieval_queries, retrieved_evidence_ids, reasoning_trace,
		       tool_calls, output, abstention_reason,
		       total_tokens, total_cost, latency_ms, trace_id,
		       started_at, completed_at, created_at
		FROM agent_run
		WHERE workspace_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, workspaceID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	runs := make([]*Run, 0)
	for rows.Next() {
		run, scanErr := scanAgentRun(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		runs = append(runs, run)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, 0, rowsErr
	}

	// Get total count
	var total int64
	countRow := o.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM agent_run WHERE workspace_id = ?
	`, workspaceID)
	if scanErr := countRow.Scan(&total); scanErr != nil {
		return nil, 0, scanErr
	}

	return runs, total, nil
}

// UpdateAgentRunStatus updates the status of an agent run
func (o *Orchestrator) UpdateAgentRunStatus(ctx context.Context, workspaceID, runID, status string) (*Run, error) {
	now := time.Now().UTC()
	_, err := o.db.ExecContext(ctx, `
		UPDATE agent_run
		SET status = ?, completed_at = ?, updated_at = ?
		WHERE id = ? AND workspace_id = ?
	`, status, now, now, runID, workspaceID)
	if err != nil {
		return nil, err
	}

	return o.GetAgentRun(ctx, workspaceID, runID)
}

// UpdateAgentRun updates an agent run with full data
func (o *Orchestrator) UpdateAgentRun(ctx context.Context, workspaceID, runID string, updates RunUpdates) (*Run, error) {
	now := time.Now().UTC()
	var completedAt *time.Time
	if updates.Completed {
		completedAt = &now
	}

	_, err := o.db.ExecContext(ctx, `
		UPDATE agent_run
		SET status = ?, inputs = ?, retrieval_queries = ?, retrieved_evidence_ids = ?,
		    reasoning_trace = ?, tool_calls = ?, output = ?, abstention_reason = ?,
		    total_tokens = ?, total_cost = ?, latency_ms = ?,
		    completed_at = COALESCE(?, completed_at), updated_at = ?
		WHERE id = ? AND workspace_id = ?
	`,
		updates.Status,
		updates.Inputs,
		updates.RetrievalQueries,
		updates.RetrievedEvidenceIDs,
		updates.ReasoningTrace,
		updates.ToolCalls,
		updates.Output,
		updates.AbstentionReason,
		updates.TotalTokens,
		updates.TotalCost,
		updates.LatencyMs,
		completedAt,
		now,
		runID,
		workspaceID,
	)
	if err != nil {
		return nil, err
	}

	return o.GetAgentRun(ctx, workspaceID, runID)
}

// RunningUpdates holds the fields that can be updated
type RunUpdates struct {
	Status               string
	Inputs               json.RawMessage
	RetrievalQueries     json.RawMessage
	RetrievedEvidenceIDs json.RawMessage
	ReasoningTrace       json.RawMessage
	ToolCalls            json.RawMessage
	Output               json.RawMessage
	AbstentionReason     *string
	TotalTokens          *int64
	TotalCost            *float64
	LatencyMs            *int64
	Completed            bool
}

// ListAgentDefinitions lists all agent definitions for a workspace
func (o *Orchestrator) ListAgentDefinitions(ctx context.Context, workspaceID string) ([]*Definition, error) {
	rows, err := o.db.QueryContext(ctx, `
		SELECT id, workspace_id, name, description, agent_type, objective,
		       allowed_tools, limits, trigger_config, policy_set_id,
		       active_prompt_version_id, status, created_at, updated_at
		FROM agent_definition
		WHERE workspace_id = ?
		ORDER BY created_at DESC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	definitions := make([]*Definition, 0)
	for rows.Next() {
		def, scanErr := scanAgentDefinition(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		definitions = append(definitions, def)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, rowsErr
	}

	return definitions, nil
}

// GetAgentDefinition retrieves an agent definition by ID
func (o *Orchestrator) GetAgentDefinition(ctx context.Context, workspaceID, agentID string) (*Definition, error) {
	return o.getAgentDefinition(ctx, agentID, workspaceID)
}

// Helper functions

func (o *Orchestrator) getAgentDefinition(ctx context.Context, id, workspaceID string) (*Definition, error) {
	row := o.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, name, description, agent_type, objective,
		       allowed_tools, limits, trigger_config, policy_set_id,
		       active_prompt_version_id, status, created_at, updated_at
		FROM agent_definition
		WHERE id = ? AND workspace_id = ?
	`, id, workspaceID)

	def, err := scanAgentDefinition(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAgentNotFound
	}
	if err != nil {
		return nil, err
	}
	return def, nil
}

func isValidTriggerType(t string) bool {
	switch t {
	case TriggerTypeEvent, TriggerTypeSchedule, TriggerTypeManual, TriggerTypeCopilot:
		return true
	default:
		return false
	}
}

type agentRunScanner interface {
	Scan(dest ...any) error
}

type agentRunNullable struct {
	triggeredByUserID sql.NullString
	triggerContext    sql.NullString
	inputs            sql.NullString
	retrievalQueries  sql.NullString
	retrievedEvidence sql.NullString
	reasoningTrace    sql.NullString
	toolCalls         sql.NullString
	output            sql.NullString
	abstentionReason  sql.NullString
	totalTokens       sql.NullInt64
	totalCost         sql.NullFloat64
	latencyMs         sql.NullInt64
	traceID           sql.NullString
	completedAt       sql.NullTime
}

func scanAgentRun(scan agentRunScanner) (*Run, error) {
	var r Run
	var n agentRunNullable

	err := scan.Scan(
		&r.ID, &r.WorkspaceID, &r.DefinitionID, &n.triggeredByUserID,
		&r.TriggerType, &n.triggerContext, &r.Status, &n.inputs,
		&n.retrievalQueries, &n.retrievedEvidence, &n.reasoningTrace,
		&n.toolCalls, &n.output, &n.abstentionReason,
		&n.totalTokens, &n.totalCost, &n.latencyMs, &n.traceID,
		&r.StartedAt, &n.completedAt, &r.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	applyRunNullables(&r, &n)
	return &r, nil
}

func applyRunNullables(r *Run, n *agentRunNullable) {
	applyRunStringFields(r, n)
	applyRunMetricFields(r, n)
}

// applyRunStringFields maps nullable string/JSON fields onto Run.
func applyRunStringFields(r *Run, n *agentRunNullable) {
	applyRunContextFields(r, n)
	applyRunPayloadFields(r, n)
}

// applyRunContextFields maps identity/context nullable fields.
func applyRunContextFields(r *Run, n *agentRunNullable) {
	if n.triggeredByUserID.Valid {
		r.TriggeredByUserID = &n.triggeredByUserID.String
	}
	if n.triggerContext.Valid {
		r.TriggerContext = json.RawMessage(n.triggerContext.String)
	}
	if n.inputs.Valid {
		r.Inputs = json.RawMessage(n.inputs.String)
	}
	if n.retrievalQueries.Valid {
		r.RetrievalQueries = json.RawMessage(n.retrievalQueries.String)
	}
}

// applyRunPayloadFields maps evidence/reasoning/output nullable fields.
func applyRunPayloadFields(r *Run, n *agentRunNullable) {
	if n.retrievedEvidence.Valid {
		r.RetrievedEvidenceIDs = json.RawMessage(n.retrievedEvidence.String)
	}
	if n.reasoningTrace.Valid {
		r.ReasoningTrace = json.RawMessage(n.reasoningTrace.String)
	}
	if n.toolCalls.Valid {
		r.ToolCalls = json.RawMessage(n.toolCalls.String)
	}
	if n.output.Valid {
		r.Output = json.RawMessage(n.output.String)
	}
	if n.abstentionReason.Valid {
		r.AbstentionReason = &n.abstentionReason.String
	}
}

// applyRunMetricFields maps nullable numeric/time fields onto Run.
func applyRunMetricFields(r *Run, n *agentRunNullable) {
	if n.totalTokens.Valid {
		r.TotalTokens = &n.totalTokens.Int64
	}
	if n.totalCost.Valid {
		r.TotalCost = &n.totalCost.Float64
	}
	if n.latencyMs.Valid {
		r.LatencyMs = &n.latencyMs.Int64
	}
	if n.traceID.Valid {
		r.TraceID = &n.traceID.String
	}
	if n.completedAt.Valid {
		r.CompletedAt = &n.completedAt.Time
	}
}

type agentDefScanner interface {
	Scan(dest ...any) error
}

func scanAgentDefinition(scan agentDefScanner) (*Definition, error) {
	var d Definition
	var (
		description    sql.NullString
		objective      sql.NullString
		allowedTools   sql.NullString
		limits         sql.NullString
		triggerConfig  sql.NullString
		policySetID    sql.NullString
		activePromptID sql.NullString
	)

	err := scan.Scan(
		&d.ID, &d.WorkspaceID, &d.Name, &description,
		&d.AgentType, &objective, &allowedTools, &limits,
		&triggerConfig, &policySetID, &activePromptID,
		&d.Status, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	applyDefinitionNullables(&d, description, objective, allowedTools, limits, triggerConfig, policySetID, activePromptID)
	return &d, nil
}

func applyDefinitionNullables(d *Definition, description, objective, allowedTools, limits, triggerConfig, policySetID, activePromptID sql.NullString) {
	applyDefinitionTextFields(d, description, objective, policySetID, activePromptID)
	applyDefinitionJSONFields(d, allowedTools, limits, triggerConfig)
}

// applyDefinitionTextFields maps nullable plain-text fields onto Definition.
func applyDefinitionTextFields(d *Definition, description, objective, policySetID, activePromptID sql.NullString) {
	if description.Valid {
		d.Description = &description.String
	}
	if objective.Valid {
		d.Objective = json.RawMessage(objective.String)
	}
	if policySetID.Valid {
		d.PolicySetID = &policySetID.String
	}
	if activePromptID.Valid {
		d.ActivePromptVersionID = &activePromptID.String
	}
}

// applyDefinitionJSONFields unmarshals nullable JSON columns onto Definition.
func applyDefinitionJSONFields(d *Definition, allowedTools, limits, triggerConfig sql.NullString) {
	if allowedTools.Valid {
		_ = json.Unmarshal([]byte(allowedTools.String), &d.AllowedTools)
	}
	if limits.Valid {
		_ = json.Unmarshal([]byte(limits.String), &d.Limits)
	}
	if triggerConfig.Valid {
		_ = json.Unmarshal([]byte(triggerConfig.String), &d.TriggerConfig)
	}
}

func stringPtr(s string) *string {
	return &s
}
