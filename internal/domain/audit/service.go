package audit

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"io"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite/sqlcgen"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

// AuditService provides audit logging capabilities
// All operations are append-only; no updates or deletes are supported
//
//nolint:revive // servicio de dominio estable y ampliamente referenciado
type AuditService struct {
	db      *sql.DB
	querier sqlcgen.Querier
}

// NewAuditService creates a new audit service
func NewAuditService(db *sql.DB) *AuditService {
	return &AuditService{
		db:      db,
		querier: sqlcgen.New(db),
	}
}

// Log creates a new audit event (append-only, immutable)
// This is the ONLY way to create audit events - no updates, no deletes
func (s *AuditService) Log(ctx context.Context, event *AuditEvent) error {
	details := normalizeJSON(event.Details, []byte("{}"))
	permissionsChecked := normalizeJSON(event.PermissionsChecked, []byte("[]"))

	params := sqlcgen.CreateAuditEventParams{
		ID:                 event.ID,
		WorkspaceID:        event.WorkspaceID,
		ActorID:            event.ActorID,
		ActorType:          string(event.ActorType),
		Action:             event.Action,
		EntityType:         event.EntityType,
		EntityID:           event.EntityID,
		Details:            details,
		PermissionsChecked: permissionsChecked,
		Outcome:            string(event.Outcome),
		TraceID:            event.TraceID,
		IpAddress:          event.IPAddress,
		UserAgent:          event.UserAgent,
		CreatedAt:          event.CreatedAt,
	}

	return s.querier.CreateAuditEvent(ctx, params)
}

// LogWithDetails is a helper for common case with structured details
func (s *AuditService) LogWithDetails(
	ctx context.Context,
	workspaceID string,
	actorID string,
	actorType ActorType,
	action string,
	entityType *string,
	entityID *string,
	details *EventDetails,
	outcome Outcome,
) error {
	var detailsJSON json.RawMessage
	if details != nil {
		var err error
		detailsJSON, err = json.Marshal(details)
		if err != nil {
			return err
		}
	}

	event := &AuditEvent{
		ID:          generateID(),
		WorkspaceID: workspaceID,
		ActorID:     actorID,
		ActorType:   actorType,
		Action:      action,
		EntityType:  entityType,
		EntityID:    entityID,
		Details:     detailsJSON,
		Outcome:     outcome,
		CreatedAt:   time.Now(),
	}

	return s.Log(ctx, event)
}

// GetByID retrieves a single audit event by ID
func (s *AuditService) GetByID(ctx context.Context, id string) (*AuditEvent, error) {
	row, err := s.querier.GetAuditEventByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return rowToAuditEvent(row), nil
}

// ListByWorkspace retrieves audit events for a workspace (with pagination)
// Results are ordered by created_at DESC (newest first)
func (s *AuditService) ListByWorkspace(
	ctx context.Context,
	workspaceID string,
	limit int,
	offset int,
) ([]*AuditEvent, int, error) {
	params := sqlcgen.ListAuditEventsByWorkspaceParams{
		WorkspaceID: workspaceID,
		Limit:       int64(limit),
		Offset:      int64(offset),
	}

	rows, err := s.querier.ListAuditEventsByWorkspace(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.querier.CountAuditEventsByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, 0, err
	}

	events := make([]*AuditEvent, len(rows))
	for i, row := range rows {
		events[i] = rowToAuditEvent(row)
	}

	return events, int(count), nil
}

// ListByActor retrieves audit events for a specific actor
func (s *AuditService) ListByActor(
	ctx context.Context,
	actorID string,
	limit int,
) ([]*AuditEvent, error) {
	params := sqlcgen.ListAuditEventsByActorParams{
		ActorID: actorID,
		Limit:   int64(limit),
	}

	rows, err := s.querier.ListAuditEventsByActor(ctx, params)
	if err != nil {
		return nil, err
	}

	events := make([]*AuditEvent, len(rows))
	for i, row := range rows {
		events[i] = rowToAuditEvent(row)
	}

	return events, nil
}

// ListByEntity retrieves audit events for a specific entity
func (s *AuditService) ListByEntity(
	ctx context.Context,
	entityType string,
	entityID string,
	limit int,
) ([]*AuditEvent, error) {
	params := sqlcgen.ListAuditEventsByEntityParams{
		EntityType: &entityType,
		EntityID:   &entityID,
		Limit:      int64(limit),
	}

	rows, err := s.querier.ListAuditEventsByEntity(ctx, params)
	if err != nil {
		return nil, err
	}

	events := make([]*AuditEvent, len(rows))
	for i, row := range rows {
		events[i] = rowToAuditEvent(row)
	}

	return events, nil
}

// ListByOutcome retrieves audit events filtered by outcome
func (s *AuditService) ListByOutcome(
	ctx context.Context,
	workspaceID string,
	outcome Outcome,
	limit int,
	offset int,
) ([]*AuditEvent, error) {
	params := sqlcgen.ListAuditEventsByOutcomeParams{
		WorkspaceID: workspaceID,
		Outcome:     string(outcome),
		Limit:       int64(limit),
		Offset:      int64(offset),
	}

	rows, err := s.querier.ListAuditEventsByOutcome(ctx, params)
	if err != nil {
		return nil, err
	}

	events := make([]*AuditEvent, len(rows))
	for i, row := range rows {
		events[i] = rowToAuditEvent(row)
	}

	return events, nil
}

// ListByAction retrieves audit events filtered by action type
func (s *AuditService) ListByAction(
	ctx context.Context,
	workspaceID string,
	action string,
	limit int,
	offset int,
) ([]*AuditEvent, error) {
	params := sqlcgen.ListAuditEventsByActionParams{
		WorkspaceID: workspaceID,
		Action:      action,
		Limit:       int64(limit),
		Offset:      int64(offset),
	}

	rows, err := s.querier.ListAuditEventsByAction(ctx, params)
	if err != nil {
		return nil, err
	}

	events := make([]*AuditEvent, len(rows))
	for i, row := range rows {
		events[i] = rowToAuditEvent(row)
	}

	return events, nil
}

// Query filters audit events with optional compound criteria.
// Task 4.6: FR-070 Audit Advanced
func (s *AuditService) Query(ctx context.Context, in QueryInput) ([]*AuditEvent, error) {
	params := sqlcgen.QueryAuditEventsParams{
		WorkspaceID: in.WorkspaceID,
		ActorID:     in.ActorID,
		EntityType:  in.EntityType,
		Action:      in.Action,
		Outcome:     in.Outcome,
		DateFrom:    normalizeDateArg(in.DateFrom),
		DateTo:      normalizeDateArg(in.DateTo),
		Off:         int64(in.Offset),
		Lim:         int64(resolveQueryLimit(in.Limit)),
	}

	rows, err := s.querier.QueryAuditEvents(ctx, params)
	if err != nil {
		return nil, err
	}

	events := make([]*AuditEvent, len(rows))
	for i, row := range rows {
		events[i] = rowToAuditEvent(row)
	}

	return events, nil
}

// Export returns audit events as a streaming CSV reader.
// Task 4.6: FR-071 Audit Export
func (s *AuditService) Export(ctx context.Context, in ExportInput) (io.Reader, error) {
	pr, pw := io.Pipe()
	go s.writeCSVExport(ctx, pw, in)
	return pr, nil
}

func (s *AuditService) writeCSVExport(ctx context.Context, pw *io.PipeWriter, in ExportInput) {
	w := csv.NewWriter(pw)
	if err := writeAuditCSVHeader(w); err != nil {
		_ = pw.CloseWithError(err)
		return
	}
	if err := s.writeAuditCSVRows(ctx, w, in); err != nil {
		_ = pw.CloseWithError(err)
		return
	}
	w.Flush()
	_ = pw.CloseWithError(w.Error())
}

func writeAuditCSVHeader(w *csv.Writer) error {
	return w.Write([]string{
		"id", "workspace_id", "actor_id", "actor_type", "action",
		"entity_type", "entity_id", "outcome", "trace_id", "created_at",
	})
}

func (s *AuditService) writeAuditCSVRows(ctx context.Context, w *csv.Writer, in ExportInput) error {
	offset := 0
	const batchSize = 500
	for {
		events, queryErr := s.Query(ctx, QueryInput{
			WorkspaceID: in.WorkspaceID,
			ActorID:     in.ActorID,
			EntityType:  in.EntityType,
			Action:      in.Action,
			Outcome:     in.Outcome,
			DateFrom:    in.DateFrom,
			DateTo:      in.DateTo,
			Limit:       batchSize,
			Offset:      offset,
		})
		if queryErr != nil {
			return queryErr
		}
		if err := writeAuditCSVBatch(w, events); err != nil {
			return err
		}
		if len(events) < batchSize {
			return nil
		}
		offset += batchSize
	}
}

func writeAuditCSVBatch(w *csv.Writer, events []*AuditEvent) error {
	for _, ev := range events {
		if err := w.Write([]string{
			ev.ID,
			ev.WorkspaceID,
			ev.ActorID,
			string(ev.ActorType),
			ev.Action,
			derefString(ev.EntityType),
			derefString(ev.EntityID),
			string(ev.Outcome),
			derefString(ev.TraceID),
			ev.CreatedAt.UTC().Format(time.RFC3339),
		}); err != nil {
			return err
		}
	}
	return nil
}

// RegisterEventSubscribers wires the audit service to all domain event topics.
// Task 4.6: Completes FR-070 audit trail for agent/tool/policy/approval events.
func (s *AuditService) RegisterEventSubscribers(bus eventbus.EventBus) {
	if bus == nil {
		return
	}
	topics := []string{
		"agent.run.started", "agent.run.completed", "agent.run.failed",
		"tool.executed", "tool.denied",
		"policy.evaluated", "approval.requested", "approval.decided",
	}
	for _, topic := range topics {
		ch := bus.Subscribe(topic)
		go s.consumeEvents(topic, ch)
	}
}

func (s *AuditService) consumeEvents(topic string, ch <-chan eventbus.Event) {
	for ev := range ch {
		workspaceID, actorID, entityType, entityID := extractEventContext(ev.Payload)
		if workspaceID == "" || actorID == "" {
			continue
		}
		_ = s.LogWithDetails(
			context.Background(),
			workspaceID,
			actorID,
			resolveActorType(topic),
			topic,
			entityType,
			entityID,
			&EventDetails{Metadata: map[string]any{"topic": topic, "payload": ev.Payload}},
			resolveOutcome(topic),
		)
	}
}

func resolveQueryLimit(limit int) int {
	if limit <= 0 {
		return 25
	}
	return limit
}

func normalizeDateArg(raw string) any {
	if raw == "" {
		return ""
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return raw
	}
	return parsed.UTC().Format("2006-01-02 15:04:05")
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func extractEventContext(payload any) (string, string, *string, *string) {
	obj, ok := payload.(map[string]any)
	if !ok {
		return "", "", nil, nil
	}
	workspaceID, _ := obj["workspace_id"].(string)
	actorID, _ := obj["actor_id"].(string)
	entityType := optionalString(obj, "entity_type")
	entityID := optionalString(obj, "entity_id")
	return workspaceID, actorID, entityType, entityID
}

func optionalString(obj map[string]any, key string) *string {
	v, ok := obj[key].(string)
	if !ok || v == "" {
		return nil
	}
	return &v
}

func resolveActorType(topic string) ActorType {
	if len(topic) >= 5 && topic[:5] == "agent" {
		return ActorTypeAgent
	}
	return ActorTypeSystem
}

func resolveOutcome(topic string) Outcome {
	if topic == "agent.run.failed" || topic == "tool.denied" {
		return OutcomeDenied
	}
	return OutcomeSuccess
}

// rowToAuditEvent converts a sqlcgen AuditEvent row to domain AuditEvent
func rowToAuditEvent(row sqlcgen.AuditEvent) *AuditEvent {
	return &AuditEvent{
		ID:                 row.ID,
		WorkspaceID:        row.WorkspaceID,
		ActorID:            row.ActorID,
		ActorType:          ActorType(row.ActorType),
		Action:             row.Action,
		EntityType:         row.EntityType,
		EntityID:           row.EntityID,
		Details:            row.Details,
		PermissionsChecked: row.PermissionsChecked,
		Outcome:            Outcome(row.Outcome),
		TraceID:            row.TraceID,
		IPAddress:          row.IpAddress,
		UserAgent:          row.UserAgent,
		CreatedAt:          row.CreatedAt,
	}
}

// generateID generates a new UUID for audit events
func generateID() string {
	// Using UUID v7 for better time-based ordering
	return uuid.NewV7().String()
}

func normalizeJSON(raw json.RawMessage, fallback []byte) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(fallback)
	}
	return raw
}
