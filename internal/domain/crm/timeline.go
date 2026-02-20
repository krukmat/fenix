package crm

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite/sqlcgen"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

type TimelineEvent struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspaceId"`
	EntityType  string    `json:"entityType"`
	EntityID    string    `json:"entityId"`
	ActorID     *string   `json:"actorId,omitempty"`
	EventType   string    `json:"eventType"`
	OldValue    *string   `json:"oldValue,omitempty"`
	NewValue    *string   `json:"newValue,omitempty"`
	Context     *string   `json:"context,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

type CreateTimelineEventInput struct {
	WorkspaceID string
	EntityType  string
	EntityID    string
	ActorID     string
	EventType   string
	OldValue    string
	NewValue    string
	Context     string
}

type ListTimelineInput struct {
	Limit  int
	Offset int
}

type TimelineService struct {
	db      *sql.DB
	querier sqlcgen.Querier
}

func NewTimelineService(db *sql.DB) *TimelineService {
	return &TimelineService{db: db, querier: sqlcgen.New(db)}
}

func (s *TimelineService) Create(ctx context.Context, input CreateTimelineEventInput) (*TimelineEvent, error) {
	id := uuid.NewV7().String()
	now := time.Now().UTC().Format(time.RFC3339)
	err := s.querier.CreateTimelineEvent(ctx, sqlcgen.CreateTimelineEventParams{
		ID:          id,
		WorkspaceID: input.WorkspaceID,
		EntityType:  input.EntityType,
		EntityID:    input.EntityID,
		ActorID:     nullString(input.ActorID),
		EventType:   input.EventType,
		OldValue:    nullString(input.OldValue),
		NewValue:    nullString(input.NewValue),
		Context:     nullString(input.Context),
		CreatedAt:   now,
	})
	if err != nil {
		return nil, fmt.Errorf("create timeline event: %w", err)
	}
	return s.Get(ctx, input.WorkspaceID, id)
}

func (s *TimelineService) Get(ctx context.Context, workspaceID, eventID string) (*TimelineEvent, error) {
	row, err := s.querier.GetTimelineEventByID(ctx, sqlcgen.GetTimelineEventByIDParams{ID: eventID, WorkspaceID: workspaceID})
	if err != nil {
		return nil, err
	}
	return rowToTimelineEvent(row), nil
}

func (s *TimelineService) List(ctx context.Context, workspaceID string, input ListTimelineInput) ([]*TimelineEvent, int, error) {
	total, err := s.querier.CountTimelineEventsByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, 0, fmt.Errorf("count timeline events: %w", err)
	}

	rows, err := s.querier.ListTimelineEventsByWorkspace(ctx, sqlcgen.ListTimelineEventsByWorkspaceParams{
		WorkspaceID: workspaceID,
		Limit:       int64(input.Limit),
		Offset:      int64(input.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list timeline events: %w", err)
	}

	return mapRows(rows, rowToTimelineEvent), int(total), nil
}

func (s *TimelineService) ListByEntity(ctx context.Context, workspaceID, entityType, entityID string, input ListTimelineInput) ([]*TimelineEvent, error) {
	rows, err := s.querier.ListTimelineEventsByEntity(ctx, sqlcgen.ListTimelineEventsByEntityParams{
		WorkspaceID: workspaceID,
		EntityType:  entityType,
		EntityID:    entityID,
		Limit:       int64(input.Limit),
		Offset:      int64(input.Offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list timeline events by entity: %w", err)
	}

	return mapRows(rows, rowToTimelineEvent), nil
}

func rowToTimelineEvent(row sqlcgen.TimelineEvent) *TimelineEvent {
	createdAt, _ := time.Parse(time.RFC3339, row.CreatedAt)
	return &TimelineEvent{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceID,
		EntityType:  row.EntityType,
		EntityID:    row.EntityID,
		ActorID:     row.ActorID,
		EventType:   row.EventType,
		OldValue:    row.OldValue,
		NewValue:    row.NewValue,
		Context:     row.Context,
		CreatedAt:   createdAt,
	}
}
