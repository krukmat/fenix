package crm

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite/sqlcgen"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

type Lead struct {
	ID          string     `json:"id"`
	WorkspaceID string     `json:"workspaceId"`
	ContactID   *string    `json:"contactId,omitempty"`
	AccountID   *string    `json:"accountId,omitempty"`
	Source      *string    `json:"source,omitempty"`
	Status      string     `json:"status"`
	OwnerID     string     `json:"ownerId"`
	Score       *float64   `json:"score,omitempty"`
	Metadata    *string    `json:"metadata,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty"`
}

type CreateLeadInput struct {
	WorkspaceID string
	ContactID   string
	AccountID   string
	Source      string
	Status      string
	OwnerID     string
	Score       *float64
	Metadata    string
}

type UpdateLeadInput struct {
	ContactID string
	AccountID string
	Source    string
	Status    string
	OwnerID   string
	Score     *float64
	Metadata  string
}

type ListLeadsInput struct {
	Limit  int
	Offset int
}

type LeadService struct {
	db      *sql.DB
	querier sqlcgen.Querier
}

func NewLeadService(db *sql.DB) *LeadService {
	return &LeadService{db: db, querier: sqlcgen.New(db)}
}

func (s *LeadService) Create(ctx context.Context, input CreateLeadInput) (*Lead, error) {
	id := uuid.NewV7().String()
	now := time.Now().UTC().Format(time.RFC3339)
	status := input.Status
	if status == "" {
		status = "new"
	}

	err := s.querier.CreateLead(ctx, sqlcgen.CreateLeadParams{
		ID:          id,
		WorkspaceID: input.WorkspaceID,
		ContactID:   nullString(input.ContactID),
		AccountID:   nullString(input.AccountID),
		Source:      nullString(input.Source),
		Status:      status,
		OwnerID:     input.OwnerID,
		Score:       input.Score,
		Metadata:    nullString(input.Metadata),
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return nil, fmt.Errorf("create lead: %w", err)
	}
	if timelineErr := createTimelineEvent(ctx, s.querier, input.WorkspaceID, timelineEntityLead, id, input.OwnerID, timelineActionCreated); timelineErr != nil {
		return nil, fmt.Errorf("create lead timeline: %w", timelineErr)
	}

	return s.Get(ctx, input.WorkspaceID, id)
}

func (s *LeadService) Get(ctx context.Context, workspaceID, leadID string) (*Lead, error) {
	row, err := s.querier.GetLeadByID(ctx, sqlcgen.GetLeadByIDParams{ID: leadID, WorkspaceID: workspaceID})
	if err != nil {
		return nil, err
	}
	return rowToLead(row), nil
}

func (s *LeadService) List(ctx context.Context, workspaceID string, input ListLeadsInput) ([]*Lead, int, error) {
	total, err := s.querier.CountLeadsByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, 0, fmt.Errorf("count leads: %w", err)
	}

	rows, err := s.querier.ListLeadsByWorkspace(ctx, sqlcgen.ListLeadsByWorkspaceParams{
		WorkspaceID: workspaceID,
		Limit:       int64(input.Limit),
		Offset:      int64(input.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list leads: %w", err)
	}

	out := make([]*Lead, len(rows))
	for i := range rows {
		out[i] = rowToLead(rows[i])
	}

	return out, int(total), nil
}

func (s *LeadService) ListByOwner(ctx context.Context, workspaceID, ownerID string) ([]*Lead, error) {
	rows, err := s.querier.ListLeadsByOwner(ctx, sqlcgen.ListLeadsByOwnerParams{WorkspaceID: workspaceID, OwnerID: ownerID})
	if err != nil {
		return nil, fmt.Errorf("list leads by owner: %w", err)
	}
	out := make([]*Lead, len(rows))
	for i := range rows {
		out[i] = rowToLead(rows[i])
	}
	return out, nil
}

func (s *LeadService) Update(ctx context.Context, workspaceID, leadID string, input UpdateLeadInput) (*Lead, error) {
	err := s.querier.UpdateLead(ctx, sqlcgen.UpdateLeadParams{
		ContactID:   nullString(input.ContactID),
		AccountID:   nullString(input.AccountID),
		Source:      nullString(input.Source),
		Status:      input.Status,
		OwnerID:     input.OwnerID,
		Score:       input.Score,
		Metadata:    nullString(input.Metadata),
		UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
		ID:          leadID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("update lead: %w", err)
	}
	if timelineErr := createTimelineEvent(ctx, s.querier, workspaceID, timelineEntityLead, leadID, input.OwnerID, "updated"); timelineErr != nil {
		return nil, fmt.Errorf("update lead timeline: %w", timelineErr)
	}

	return s.Get(ctx, workspaceID, leadID)
}

func (s *LeadService) Delete(ctx context.Context, workspaceID, leadID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	err := s.querier.SoftDeleteLead(ctx, sqlcgen.SoftDeleteLeadParams{
		DeletedAt:   &now,
		UpdatedAt:   now,
		ID:          leadID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return fmt.Errorf("soft delete lead: %w", err)
	}
	if timelineErr := createTimelineEvent(ctx, s.querier, workspaceID, timelineEntityLead, leadID, "", "deleted"); timelineErr != nil {
		return fmt.Errorf("delete lead timeline: %w", timelineErr)
	}
	return nil
}

func rowToLead(row sqlcgen.Lead) *Lead {
	createdAt, _ := time.Parse(time.RFC3339, row.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, row.UpdatedAt)

	var deletedAt *time.Time
	if row.DeletedAt != nil {
		t, _ := time.Parse(time.RFC3339, *row.DeletedAt)
		deletedAt = &t
	}

	return &Lead{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceID,
		ContactID:   row.ContactID,
		AccountID:   row.AccountID,
		Source:      row.Source,
		Status:      row.Status,
		OwnerID:     row.OwnerID,
		Score:       row.Score,
		Metadata:    row.Metadata,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		DeletedAt:   deletedAt,
	}
}
