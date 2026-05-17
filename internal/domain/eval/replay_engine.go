package eval

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/blackboard"
)

var (
	// ErrReplayProvenanceMissing indicates that an eval run lacks the structured
	// provenance required to reconstruct a replay-backed execution.
	ErrReplayProvenanceMissing = errors.New("eval replay provenance missing")
	// ErrReplaySourceRunMissing indicates that the source agent_run cannot be found.
	ErrReplaySourceRunMissing = errors.New("eval replay source agent run missing")
	// ErrReplayTraceMissing indicates that the source audit trace cannot be found.
	ErrReplayTraceMissing = errors.New("eval replay audit trace missing")
	// ErrReplayTimelineMissing indicates that the reasoning timeline cannot be found.
	ErrReplayTimelineMissing = errors.New("eval replay reasoning timeline missing")
)

// ReplaySourceErrorKind classifies replay reconstruction failures.
type ReplaySourceErrorKind string

const (
	ReplaySourceErrorProvenance ReplaySourceErrorKind = "provenance_missing"
	ReplaySourceErrorRun        ReplaySourceErrorKind = "source_run_missing"
	ReplaySourceErrorTrace      ReplaySourceErrorKind = "trace_missing"
	ReplaySourceErrorTimeline   ReplaySourceErrorKind = "timeline_missing"
)

// ReplaySourceError captures the missing replay primitive plus stable identifiers
// that callers can surface or inspect.
type ReplaySourceError struct {
	Kind        ReplaySourceErrorKind `json:"kind"`
	WorkspaceID string                `json:"workspaceId,omitempty"`
	EvalRunID   string                `json:"evalRunId,omitempty"`
	SourceID    string                `json:"sourceId,omitempty"`
	Err         error                 `json:"-"`
}

func (e *ReplaySourceError) Error() string {
	if e == nil {
		return ""
	}
	base := replaySourceKindError(e.Kind)
	if e.SourceID == "" {
		return base.Error()
	}
	return fmt.Sprintf("%s: %s", base.Error(), e.SourceID)
}

// Unwrap exposes the sentinel error so callers can use errors.Is.
func (e *ReplaySourceError) Unwrap() error {
	if e == nil {
		return nil
	}
	if e.Err != nil {
		return e.Err
	}
	return replaySourceKindError(e.Kind)
}

// ReplayEngine defines the service boundary for deterministic replay-backed evals.
// Later subtasks will implement this contract.
type ReplayEngine interface {
	BuildReplay(ctx context.Context, in ReplayRequest) (*ReplayArtifact, error)
}

// ReplaySourceLoader resolves persisted provenance into deterministic replay source truth.
type ReplaySourceLoader interface {
	LoadSource(ctx context.Context, in ReplayRequest) (*ReplaySource, error)
}

// ReplayRequest is the normalized contract that identifies which eval run should
// be reconstructed into a replay artifact.
type ReplayRequest struct {
	EvalRunID   string            `json:"evalRunId"`
	WorkspaceID string            `json:"workspaceId"`
	Provenance  *ReplayProvenance `json:"provenance,omitempty"`
	ScenarioID  string            `json:"scenarioId,omitempty"`
	RequestedBy string            `json:"requestedBy,omitempty"`
	RequestedAt time.Time         `json:"requestedAt,omitempty"`
}

// ReplaySource aggregates the persisted truth required to reconstruct a replay.
type ReplaySource struct {
	Provenance      ReplayProvenance       `json:"provenance"`
	SourceRun       *ReplaySourceRun       `json:"sourceRun,omitempty"`
	ReasoningEvents []ReplayReasoningEvent `json:"reasoningEvents,omitempty"`
	AuditEvents     []TraceAuditEvent      `json:"auditEvents,omitempty"`
}

// ReplaySourceRun is the stable projection of the original agent execution.
type ReplaySourceRun struct {
	RunID                string          `json:"runId"`
	WorkspaceID          string          `json:"workspaceId"`
	AgentDefinitionID    string          `json:"agentDefinitionId"`
	TriggerType          string          `json:"triggerType"`
	Status               string          `json:"status"`
	TraceID              *string         `json:"traceId,omitempty"`
	CognitiveWorkspaceID *string         `json:"cognitiveWorkspaceId,omitempty"`
	TriggerContext       json.RawMessage `json:"triggerContext,omitempty"`
	Inputs               json.RawMessage `json:"inputs,omitempty"`
	RetrievedEvidenceIDs []string        `json:"retrievedEvidenceIds,omitempty"`
	ReasoningTrace       json.RawMessage `json:"reasoningTrace,omitempty"`
	ToolCalls            []TraceToolCall `json:"toolCalls,omitempty"`
	Output               json.RawMessage `json:"output,omitempty"`
	AbstentionReason     *string         `json:"abstentionReason,omitempty"`
	StartedAt            time.Time       `json:"startedAt"`
	CompletedAt          *time.Time      `json:"completedAt,omitempty"`
}

// ReplayReasoningEvent is the replay-local projection of a reasoning timeline row.
type ReplayReasoningEvent struct {
	ID                   string          `json:"id"`
	CognitiveWorkspaceID string          `json:"cognitiveWorkspaceId"`
	ActorAgentID         *string         `json:"actorAgentId,omitempty"`
	EventType            string          `json:"eventType"`
	Payload              json.RawMessage `json:"payload"`
	CreatedAt            time.Time       `json:"createdAt"`
}

// ReplayInput is the deterministic, normalized input assembled from persisted
// replay source truth. C.2.3 will build and consume this DTO.
type ReplayInput struct {
	Request         ReplayRequest         `json:"request"`
	Source          ReplaySource          `json:"source"`
	ContextInputs   json.RawMessage       `json:"contextInputs,omitempty"`
	InputEvent      json.RawMessage       `json:"inputEvent,omitempty"`
	EvidenceSources []string              `json:"evidenceSources,omitempty"`
	ToolCalls       []TraceToolCall       `json:"toolCalls,omitempty"`
	PolicyDecisions []TracePolicyDecision `json:"policyDecisions,omitempty"`
	ApprovalEvents  []TraceApprovalEvent  `json:"approvalEvents,omitempty"`
}

// ReplayArtifact is the canonical replay output consumed by downstream scoring
// and benchmark registry work.
type ReplayArtifact struct {
	Request         ReplayRequest          `json:"request"`
	Source          ReplaySource           `json:"source"`
	Input           ReplayInput            `json:"input"`
	FinalOutcome    string                 `json:"finalOutcome"`
	Output          json.RawMessage        `json:"output,omitempty"`
	ReasoningEvents []ReplayReasoningEvent `json:"reasoningEvents,omitempty"`
	ToolCalls       []TraceToolCall        `json:"toolCalls,omitempty"`
	EvidenceSources []string               `json:"evidenceSources,omitempty"`
	PolicyDecisions []TracePolicyDecision  `json:"policyDecisions,omitempty"`
	ApprovalEvents  []TraceApprovalEvent   `json:"approvalEvents,omitempty"`
	AuditEvents     []TraceAuditEvent      `json:"auditEvents,omitempty"`
	BuiltAt         time.Time              `json:"builtAt"`
}

// SQLiteReplayEngine is the read-side SQLite-backed replay implementation.
// C.2.2 wires the persisted source loaders; replay execution remains for C.2.3.
type SQLiteReplayEngine struct {
	db       *sql.DB
	timeline blackboard.ReasoningTimeline
}

// NewSQLiteReplayEngine constructs a SQLite-backed replay engine.
func NewSQLiteReplayEngine(db *sql.DB) *SQLiteReplayEngine {
	return &SQLiteReplayEngine{
		db:       db,
		timeline: blackboard.NewReasoningTimeline(db),
	}
}

// BuildReplay loads persisted source truth and assembles a normalized replay
// artifact without mutating source CRM state or original runtime records. (task-C2.3)
func (e *SQLiteReplayEngine) BuildReplay(ctx context.Context, in ReplayRequest) (*ReplayArtifact, error) {
	source, err := e.LoadSource(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("build replay for eval run %s: %w", in.EvalRunID, err)
	}

	input := assembleReplayInput(in, source)

	artifact := &ReplayArtifact{
		Request:         in,
		Source:          *source,
		Input:           input,
		FinalOutcome:    replayFinalOutcome(source),
		Output:          replayOutput(source),
		ReasoningEvents: source.ReasoningEvents,
		ToolCalls:       input.ToolCalls,
		EvidenceSources: input.EvidenceSources,
		PolicyDecisions: input.PolicyDecisions,
		ApprovalEvents:  input.ApprovalEvents,
		AuditEvents:     source.AuditEvents,
		BuiltAt:         time.Now().UTC(),
	}
	return artifact, nil
}

// assembleReplayInput builds the deterministic replay input from persisted source
// truth. It is a pure function: no DB access, no side effects. (task-C2.3)
func assembleReplayInput(req ReplayRequest, source *ReplaySource) ReplayInput {
	input := ReplayInput{
		Request: req,
		Source:  *source,
	}
	if source.SourceRun == nil {
		return input
	}
	run := source.SourceRun
	input.ContextInputs = run.Inputs
	input.InputEvent = run.TriggerContext
	input.EvidenceSources = run.RetrievedEvidenceIDs
	input.ToolCalls = run.ToolCalls
	return input
}

func replayFinalOutcome(source *ReplaySource) string {
	if source.SourceRun != nil && source.SourceRun.Status != "" {
		return source.SourceRun.Status
	}
	return "unknown"
}

func replayOutput(source *ReplaySource) json.RawMessage {
	if source.SourceRun != nil {
		return source.SourceRun.Output
	}
	return nil
}

// LoadSource resolves replay provenance into persisted source truth.
func (e *SQLiteReplayEngine) LoadSource(ctx context.Context, in ReplayRequest) (*ReplaySource, error) {
	provenance, err := e.resolveReplayProvenance(ctx, in)
	if err != nil {
		return nil, err
	}
	source := &ReplaySource{Provenance: *provenance}
	err = e.attachSourceRun(ctx, in.WorkspaceID, provenance, source)
	if err != nil {
		return nil, err
	}
	err = e.attachReasoningEvents(ctx, in, provenance, source)
	if err != nil {
		return nil, err
	}
	err = e.attachAuditEvents(ctx, in, provenance, source)
	if err != nil {
		return nil, err
	}
	return source, nil
}

func (e *SQLiteReplayEngine) attachSourceRun(
	ctx context.Context, workspaceID string, provenance *ReplayProvenance, source *ReplaySource,
) error {
	if provenance.SourceAgentRunID == nil {
		return nil
	}
	run, err := e.loadSourceRun(ctx, workspaceID, *provenance.SourceAgentRunID)
	if err != nil {
		return err
	}
	source.SourceRun = run
	return nil
}

func (e *SQLiteReplayEngine) attachReasoningEvents(
	ctx context.Context, in ReplayRequest, provenance *ReplayProvenance, source *ReplaySource,
) error {
	if provenance.SourceCognitiveWorkspaceID == nil {
		return nil
	}
	events, err := e.loadReasoningEvents(ctx, in, *provenance.SourceCognitiveWorkspaceID)
	if err != nil {
		return err
	}
	source.ReasoningEvents = events
	return nil
}

func (e *SQLiteReplayEngine) attachAuditEvents(
	ctx context.Context, in ReplayRequest, provenance *ReplayProvenance, source *ReplaySource,
) error {
	if provenance.SourceTraceID == nil {
		return nil
	}
	events, err := e.loadAuditEvents(ctx, in, *provenance.SourceTraceID)
	if err != nil {
		return err
	}
	source.AuditEvents = events
	return nil
}

func replaySourceKindError(kind ReplaySourceErrorKind) error {
	switch kind {
	case ReplaySourceErrorRun:
		return ErrReplaySourceRunMissing
	case ReplaySourceErrorTrace:
		return ErrReplayTraceMissing
	case ReplaySourceErrorTimeline:
		return ErrReplayTimelineMissing
	case ReplaySourceErrorProvenance:
		fallthrough
	default:
		return ErrReplayProvenanceMissing
	}
}

func (e *SQLiteReplayEngine) resolveReplayProvenance(
	ctx context.Context,
	in ReplayRequest,
) (*ReplayProvenance, error) {
	if hasReplaySourceReferences(in.Provenance) {
		return in.Provenance, nil
	}
	if in.EvalRunID == "" || in.WorkspaceID == "" {
		return nil, newReplaySourceError(ReplaySourceErrorProvenance, in, "")
	}

	var (
		mode                       string
		benchmarkCaseID            sql.NullString
		syntheticOrgID             sql.NullString
		sourceAgentRunID           sql.NullString
		sourceCognitiveWorkspaceID sql.NullString
		sourceTraceID              sql.NullString
	)
	err := e.db.QueryRowContext(ctx, `
		SELECT replay_mode, benchmark_case_id, synthetic_org_id, source_agent_run_id,
		       source_cognitive_workspace_id, source_trace_id
		FROM eval_run
		WHERE id = ? AND workspace_id = ?`,
		in.EvalRunID,
		in.WorkspaceID,
	).Scan(
		&mode,
		&benchmarkCaseID,
		&syntheticOrgID,
		&sourceAgentRunID,
		&sourceCognitiveWorkspaceID,
		&sourceTraceID,
	)
	if err != nil {
		return nil, fmt.Errorf("load eval replay provenance: %w", err)
	}

	provenance := &ReplayProvenance{
		Mode:                       ReplayMode(mode),
		BenchmarkCaseID:            nullStringPtr(benchmarkCaseID),
		SyntheticOrgID:             nullStringPtr(syntheticOrgID),
		SourceAgentRunID:           nullStringPtr(sourceAgentRunID),
		SourceCognitiveWorkspaceID: nullStringPtr(sourceCognitiveWorkspaceID),
		SourceTraceID:              nullStringPtr(sourceTraceID),
	}
	if !hasReplaySourceReferences(provenance) {
		return nil, newReplaySourceError(ReplaySourceErrorProvenance, in, "")
	}
	return provenance, nil
}

func hasReplaySourceReferences(provenance *ReplayProvenance) bool {
	if provenance == nil {
		return false
	}
	return provenance.SourceAgentRunID != nil ||
		provenance.SourceCognitiveWorkspaceID != nil ||
		provenance.SourceTraceID != nil
}

func (e *SQLiteReplayEngine) loadSourceRun(
	ctx context.Context,
	workspaceID, runID string,
) (*ReplaySourceRun, error) {
	var (
		row                  ReplaySourceRun
		traceID              sql.NullString
		cognitiveWorkspaceID sql.NullString
		triggerContext       sql.NullString
		inputs               sql.NullString
		retrievedEvidenceIDs sql.NullString
		reasoningTrace       sql.NullString
		toolCalls            sql.NullString
		output               sql.NullString
		abstentionReason     sql.NullString
	)
	err := e.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, agent_definition_id, trigger_type, status, trace_id,
		       cognitive_workspace_id, trigger_context, inputs, retrieved_evidence_ids,
		       reasoning_trace, tool_calls, output, abstention_reason, started_at, completed_at
		FROM agent_run
		WHERE id = ? AND workspace_id = ?`,
		runID,
		workspaceID,
	).Scan(
		&row.RunID,
		&row.WorkspaceID,
		&row.AgentDefinitionID,
		&row.TriggerType,
		&row.Status,
		&traceID,
		&cognitiveWorkspaceID,
		&triggerContext,
		&inputs,
		&retrievedEvidenceIDs,
		&reasoningTrace,
		&toolCalls,
		&output,
		&abstentionReason,
		&row.StartedAt,
		&row.CompletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &ReplaySourceError{
				Kind:        ReplaySourceErrorRun,
				WorkspaceID: workspaceID,
				SourceID:    runID,
			}
		}
		return nil, fmt.Errorf("load replay source run: %w", err)
	}

	row.TraceID = nullStringPtr(traceID)
	row.CognitiveWorkspaceID = nullStringPtr(cognitiveWorkspaceID)
	row.TriggerContext = rawMessageFromNullString(triggerContext)
	row.Inputs = rawMessageFromNullString(inputs)
	row.RetrievedEvidenceIDs = parseStringSlice(rawMessageFromNullString(retrievedEvidenceIDs))
	row.ReasoningTrace = rawMessageFromNullString(reasoningTrace)
	row.ToolCalls = parseToolCalls(rawMessageFromNullString(toolCalls))
	row.Output = rawMessageFromNullString(output)
	row.AbstentionReason = nullStringPtr(abstentionReason)
	return &row, nil
}

func (e *SQLiteReplayEngine) loadReasoningEvents(
	ctx context.Context,
	in ReplayRequest,
	cognitiveWorkspaceID string,
) ([]ReplayReasoningEvent, error) {
	events, err := e.timeline.List(ctx, cognitiveWorkspaceID, blackboard.TimelineFilter{})
	if err != nil {
		return nil, fmt.Errorf("load replay reasoning events: %w", err)
	}
	if len(events) == 0 {
		return nil, newReplaySourceError(ReplaySourceErrorTimeline, in, cognitiveWorkspaceID)
	}

	out := make([]ReplayReasoningEvent, 0, len(events))
	for _, event := range events {
		out = append(out, ReplayReasoningEvent{
			ID:                   event.ID,
			CognitiveWorkspaceID: event.CognitiveWorkspaceID,
			ActorAgentID:         event.ActorAgentID,
			EventType:            string(event.EventType),
			Payload:              append(json.RawMessage(nil), event.Payload...),
			CreatedAt:            event.CreatedAt,
		})
	}
	return out, nil
}

func (e *SQLiteReplayEngine) loadAuditEvents(
	ctx context.Context,
	in ReplayRequest,
	traceID string,
) ([]TraceAuditEvent, error) {
	rows, err := e.db.QueryContext(ctx, `
		SELECT id, actor_id, action, entity_id, details, outcome, created_at
		FROM audit_event
		WHERE workspace_id = ? AND trace_id = ?
		ORDER BY created_at ASC, id ASC`,
		in.WorkspaceID,
		traceID,
	)
	if err != nil {
		return nil, fmt.Errorf("load replay audit events: %w", err)
	}
	defer rows.Close()

	filtered := make([]TraceAuditEvent, 0)
	for rows.Next() {
		var (
			event    TraceAuditEvent
			entityID sql.NullString
			details  sql.NullString
		)
		if scanErr := rows.Scan(
			&event.ID,
			&event.ActorID,
			&event.Action,
			&entityID,
			&details,
			&event.Outcome,
			&event.At,
		); scanErr != nil {
			return nil, fmt.Errorf("scan replay audit event: %w", scanErr)
		}
		event.EntityID = nullStringPtr(entityID)
		event.Details = rawMessageFromNullString(details)
		filtered = append(filtered, event)
	}
	if iterErr := rows.Err(); iterErr != nil {
		return nil, fmt.Errorf("iterate replay audit events: %w", iterErr)
	}
	if len(filtered) == 0 {
		return nil, newReplaySourceError(ReplaySourceErrorTrace, in, traceID)
	}
	return filtered, nil
}

func newReplaySourceError(kind ReplaySourceErrorKind, in ReplayRequest, sourceID string) *ReplaySourceError {
	return &ReplaySourceError{
		Kind:        kind,
		WorkspaceID: in.WorkspaceID,
		EvalRunID:   in.EvalRunID,
		SourceID:    sourceID,
	}
}

func rawMessageFromNullString(in sql.NullString) json.RawMessage {
	if !in.Valid || in.String == "" {
		return nil
	}
	return json.RawMessage(in.String)
}
