package crm

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite/sqlcgen"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

type Deal struct {
	ID            string     `json:"id"`
	WorkspaceID   string     `json:"workspaceId"`
	AccountID     string     `json:"accountId"`
	ContactID     *string    `json:"contactId,omitempty"`
	PipelineID    string     `json:"pipelineId"`
	StageID       string     `json:"stageId"`
	OwnerID       string     `json:"ownerId"`
	Title         string     `json:"title"`
	Amount        *float64   `json:"amount,omitempty"`
	Currency      *string    `json:"currency,omitempty"`
	ExpectedClose *string    `json:"expectedClose,omitempty"`
	Status        string     `json:"status"`
	Metadata      *string    `json:"metadata,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	DeletedAt     *time.Time `json:"deletedAt,omitempty"`
}

type CreateDealInput struct {
	WorkspaceID   string
	AccountID     string
	ContactID     string
	PipelineID    string
	StageID       string
	OwnerID       string
	Title         string
	Amount        *float64
	Currency      string
	ExpectedClose string
	Status        string
	Metadata      string
}

type UpdateDealInput struct {
	AccountID     string
	ContactID     string
	PipelineID    string
	StageID       string
	OwnerID       string
	Title         string
	Amount        *float64
	Currency      string
	ExpectedClose string
	Status        string
	Metadata      string
}

type ListDealsInput struct {
	Limit  int
	Offset int
}

type DealService struct {
	db      *sql.DB
	querier sqlcgen.Querier
}

func NewDealService(db *sql.DB) *DealService {
	return &DealService{db: db, querier: sqlcgen.New(db)}
}

func (s *DealService) Create(ctx context.Context, input CreateDealInput) (*Deal, error) {
	id := uuid.NewV7().String()
	now := time.Now().UTC().Format(time.RFC3339)
	status := input.Status
	if status == "" {
		status = "open"
	}

	err := s.querier.CreateDeal(ctx, sqlcgen.CreateDealParams{
		ID:            id,
		WorkspaceID:   input.WorkspaceID,
		AccountID:     input.AccountID,
		ContactID:     nullString(input.ContactID),
		PipelineID:    input.PipelineID,
		StageID:       input.StageID,
		OwnerID:       input.OwnerID,
		Title:         input.Title,
		Amount:        input.Amount,
		Currency:      nullString(input.Currency),
		ExpectedClose: nullString(input.ExpectedClose),
		Status:        status,
		Metadata:      nullString(input.Metadata),
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		return nil, fmt.Errorf("create deal: %w", err)
	}
	if timelineErr := createTimelineEvent(ctx, s.querier, input.WorkspaceID, "deal", id, input.OwnerID, "created"); timelineErr != nil {
		return nil, fmt.Errorf("create deal timeline: %w", timelineErr)
	}

	return s.Get(ctx, input.WorkspaceID, id)
}

func (s *DealService) Get(ctx context.Context, workspaceID, dealID string) (*Deal, error) {
	row, err := s.querier.GetDealByID(ctx, sqlcgen.GetDealByIDParams{ID: dealID, WorkspaceID: workspaceID})
	if err != nil {
		return nil, err
	}
	return rowToDeal(row), nil
}

func (s *DealService) List(ctx context.Context, workspaceID string, input ListDealsInput) ([]*Deal, int, error) {
	total, err := s.querier.CountDealsByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, 0, fmt.Errorf("count deals: %w", err)
	}

	rows, err := s.querier.ListDealsByWorkspace(ctx, sqlcgen.ListDealsByWorkspaceParams{
		WorkspaceID: workspaceID,
		Limit:       int64(input.Limit),
		Offset:      int64(input.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list deals: %w", err)
	}

	out := make([]*Deal, len(rows))
	for i := range rows {
		out[i] = rowToDeal(rows[i])
	}

	return out, int(total), nil
}

func (s *DealService) Update(ctx context.Context, workspaceID, dealID string, input UpdateDealInput) (*Deal, error) {
	err := s.querier.UpdateDeal(ctx, sqlcgen.UpdateDealParams{
		AccountID:     input.AccountID,
		ContactID:     nullString(input.ContactID),
		PipelineID:    input.PipelineID,
		StageID:       input.StageID,
		OwnerID:       input.OwnerID,
		Title:         input.Title,
		Amount:        input.Amount,
		Currency:      nullString(input.Currency),
		ExpectedClose: nullString(input.ExpectedClose),
		Status:        input.Status,
		Metadata:      nullString(input.Metadata),
		UpdatedAt:     time.Now().UTC().Format(time.RFC3339),
		ID:            dealID,
		WorkspaceID:   workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("update deal: %w", err)
	}
	if timelineErr := createTimelineEvent(ctx, s.querier, workspaceID, "deal", dealID, input.OwnerID, "updated"); timelineErr != nil {
		return nil, fmt.Errorf("update deal timeline: %w", timelineErr)
	}

	return s.Get(ctx, workspaceID, dealID)
}

func (s *DealService) Delete(ctx context.Context, workspaceID, dealID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	err := s.querier.SoftDeleteDeal(ctx, sqlcgen.SoftDeleteDealParams{
		DeletedAt:   &now,
		UpdatedAt:   now,
		ID:          dealID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return fmt.Errorf("soft delete deal: %w", err)
	}
	if timelineErr := createTimelineEvent(ctx, s.querier, workspaceID, "deal", dealID, "", "deleted"); timelineErr != nil {
		return fmt.Errorf("delete deal timeline: %w", timelineErr)
	}
	return nil
}

func rowToDeal(row sqlcgen.Deal) *Deal {
	createdAt, _ := time.Parse(time.RFC3339, row.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, row.UpdatedAt)

	var deletedAt *time.Time
	if row.DeletedAt != nil {
		t, _ := time.Parse(time.RFC3339, *row.DeletedAt)
		deletedAt = &t
	}

	return &Deal{
		ID:            row.ID,
		WorkspaceID:   row.WorkspaceID,
		AccountID:     row.AccountID,
		ContactID:     row.ContactID,
		PipelineID:    row.PipelineID,
		StageID:       row.StageID,
		OwnerID:       row.OwnerID,
		Title:         row.Title,
		Amount:        row.Amount,
		Currency:      row.Currency,
		ExpectedClose: row.ExpectedClose,
		Status:        row.Status,
		Metadata:      row.Metadata,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
		DeletedAt:     deletedAt,
	}
}
