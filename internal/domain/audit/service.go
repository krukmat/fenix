package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

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
