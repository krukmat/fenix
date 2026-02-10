package crm

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite/sqlcgen"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

type Activity struct {
	ID           string     `json:"id"`
	WorkspaceID  string     `json:"workspaceId"`
	ActivityType string     `json:"activityType"`
	EntityType   string     `json:"entityType"`
	EntityID     string     `json:"entityId"`
	OwnerID      string     `json:"ownerId"`
	AssignedTo   *string    `json:"assignedTo,omitempty"`
	Subject      string     `json:"subject"`
	Body         *string    `json:"body,omitempty"`
	Status       string     `json:"status"`
	DueAt        *time.Time `json:"dueAt,omitempty"`
	CompletedAt  *time.Time `json:"completedAt,omitempty"`
	Metadata     *string    `json:"metadata,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

type CreateActivityInput struct {
	WorkspaceID  string
	ActivityType string
	EntityType   string
	EntityID     string
	OwnerID      string
	AssignedTo   string
	Subject      string
	Body         string
	Status       string
	DueAt        string
	CompletedAt  string
	Metadata     string
}

type UpdateActivityInput struct {
	ActivityType string
	EntityType   string
	EntityID     string
	OwnerID      string
	AssignedTo   string
	Subject      string
	Body         string
	Status       string
	DueAt        string
	CompletedAt  string
	Metadata     string
}

type ListActivitiesInput struct {
	Limit  int
	Offset int
}

type ActivityService struct {
	db      *sql.DB
	querier sqlcgen.Querier
}

func NewActivityService(db *sql.DB) *ActivityService {
	return &ActivityService{db: db, querier: sqlcgen.New(db)}
}

func (s *ActivityService) Create(ctx context.Context, input CreateActivityInput) (*Activity, error) {
	id := uuid.NewV7().String()
	now := time.Now().UTC().Format(time.RFC3339)
	status := input.Status
	if status == "" {
		status = "pending"
	}

	err := s.querier.CreateActivity(ctx, sqlcgen.CreateActivityParams{
		ID:           id,
		WorkspaceID:  input.WorkspaceID,
		ActivityType: input.ActivityType,
		EntityType:   input.EntityType,
		EntityID:     input.EntityID,
		OwnerID:      input.OwnerID,
		AssignedTo:   nullString(input.AssignedTo),
		Subject:      input.Subject,
		Body:         nullString(input.Body),
		Status:       status,
		DueAt:        nullString(input.DueAt),
		CompletedAt:  nullString(input.CompletedAt),
		Metadata:     nullString(input.Metadata),
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		return nil, fmt.Errorf("create activity: %w", err)
	}
	if err := createTimelineEvent(ctx, s.querier, input.WorkspaceID, input.EntityType, input.EntityID, input.OwnerID, "activity_created"); err != nil {
		return nil, fmt.Errorf("create activity timeline: %w", err)
	}

	return s.Get(ctx, input.WorkspaceID, id)
}

func (s *ActivityService) Get(ctx context.Context, workspaceID, activityID string) (*Activity, error) {
	row, err := s.querier.GetActivityByID(ctx, sqlcgen.GetActivityByIDParams{ID: activityID, WorkspaceID: workspaceID})
	if err != nil {
		return nil, err
	}
	return rowToActivity(row), nil
}

func (s *ActivityService) List(ctx context.Context, workspaceID string, input ListActivitiesInput) ([]*Activity, int, error) {
	total, err := s.querier.CountActivitiesByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, 0, fmt.Errorf("count activities: %w", err)
	}
	rows, err := s.querier.ListActivitiesByWorkspace(ctx, sqlcgen.ListActivitiesByWorkspaceParams{
		WorkspaceID: workspaceID,
		Limit:       int64(input.Limit),
		Offset:      int64(input.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list activities: %w", err)
	}
	out := make([]*Activity, len(rows))
	for i := range rows {
		out[i] = rowToActivity(rows[i])
	}
	return out, int(total), nil
}

func (s *ActivityService) Update(ctx context.Context, workspaceID, activityID string, input UpdateActivityInput) (*Activity, error) {
	err := s.querier.UpdateActivity(ctx, sqlcgen.UpdateActivityParams{
		ActivityType: input.ActivityType,
		EntityType:   input.EntityType,
		EntityID:     input.EntityID,
		OwnerID:      input.OwnerID,
		AssignedTo:   nullString(input.AssignedTo),
		Subject:      input.Subject,
		Body:         nullString(input.Body),
		Status:       input.Status,
		DueAt:        nullString(input.DueAt),
		CompletedAt:  nullString(input.CompletedAt),
		Metadata:     nullString(input.Metadata),
		UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
		ID:           activityID,
		WorkspaceID:  workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("update activity: %w", err)
	}
	if err := createTimelineEvent(ctx, s.querier, workspaceID, input.EntityType, input.EntityID, input.OwnerID, "activity_updated"); err != nil {
		return nil, fmt.Errorf("update activity timeline: %w", err)
	}

	return s.Get(ctx, workspaceID, activityID)
}

func (s *ActivityService) Delete(ctx context.Context, workspaceID, activityID string) error {
	err := s.querier.DeleteActivity(ctx, sqlcgen.DeleteActivityParams{ID: activityID, WorkspaceID: workspaceID})
	if err != nil {
		return fmt.Errorf("delete activity: %w", err)
	}
	if err := createTimelineEvent(ctx, s.querier, workspaceID, "activity", activityID, "", "activity_deleted"); err != nil {
		return fmt.Errorf("delete activity timeline: %w", err)
	}
	return nil
}

func rowToActivity(row sqlcgen.Activity) *Activity {
	createdAt, _ := time.Parse(time.RFC3339, row.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, row.UpdatedAt)

	var dueAt *time.Time
	if row.DueAt != nil {
		t, _ := time.Parse(time.RFC3339, *row.DueAt)
		dueAt = &t
	}

	var completedAt *time.Time
	if row.CompletedAt != nil {
		t, _ := time.Parse(time.RFC3339, *row.CompletedAt)
		completedAt = &t
	}

	return &Activity{
		ID:           row.ID,
		WorkspaceID:  row.WorkspaceID,
		ActivityType: row.ActivityType,
		EntityType:   row.EntityType,
		EntityID:     row.EntityID,
		OwnerID:      row.OwnerID,
		AssignedTo:   row.AssignedTo,
		Subject:      row.Subject,
		Body:         row.Body,
		Status:       row.Status,
		DueAt:        dueAt,
		CompletedAt:  completedAt,
		Metadata:     row.Metadata,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
}
