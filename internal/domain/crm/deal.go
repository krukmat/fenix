package crm

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
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
	Limit      int
	Offset     int
	Status     string
	OwnerID    string
	AccountID  string
	PipelineID string
	StageID    string
	Sort       string
}

const (
	sortCreatedAtAsc  = "created_at"
	sortCreatedAtDesc = "-created_at"
)

type DealService struct {
	db      *sql.DB
	querier sqlcgen.Querier
	audit   auditLogger
}

func NewDealService(db *sql.DB) *DealService {
	return &DealService{db: db, querier: sqlcgen.New(db), audit: newCRMAuditService(db)}
}

func (s *DealService) Create(ctx context.Context, input CreateDealInput) (*Deal, error) {
	id := uuid.NewV7().String()
	now := nowRFC3339()
	status := input.Status
	if status == "" {
		status = "open"
	}
	input.Status = status
	if validationErr := validateDealInput(ctx, s.db, input.WorkspaceID, input); validationErr != nil {
		return nil, validationErr
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
	if timelineErr := createTimelineEvent(ctx, s.querier, input.WorkspaceID, timelineEntityDeal, id, input.OwnerID, timelineActionCreated); timelineErr != nil {
		return nil, fmt.Errorf("create deal timeline: %w", timelineErr)
	}
	logCRMAudit(ctx, s.audit, input.WorkspaceID, input.OwnerID, actionDealCreated, timelineEntityDeal, id)

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
	if input.Sort == "" {
		input.Sort = sortCreatedAtDesc
	}
	return listFilteredOrPaged(
		shouldUseFilteredDealList(input),
		func() ([]*Deal, error) { return s.listFiltered(ctx, workspaceID, input) },
		paginateDeals,
		input.Offset, input.Limit,
		func() (int64, error) {
			n, err := s.querier.CountDealsByWorkspace(ctx, workspaceID)
			if err != nil {
				return 0, fmt.Errorf("count deals: %w", err)
			}
			return n, nil
		},
		func() ([]*Deal, error) {
			rows, err := s.querier.ListDealsByWorkspace(ctx, sqlcgen.ListDealsByWorkspaceParams{
				WorkspaceID: workspaceID,
				Limit:       int64(input.Limit),
				Offset:      int64(input.Offset),
			})
			if err != nil {
				return nil, fmt.Errorf("list deals: %w", err)
			}
			return mapRows(rows, rowToDeal), nil
		},
	)
}

func shouldUseFilteredDealList(input ListDealsInput) bool {
	return input.AccountID != "" || input.OwnerID != "" || input.PipelineID != "" || input.StageID != "" || input.Status != "" || input.Sort != sortCreatedAtDesc
}

func (s *DealService) listFiltered(ctx context.Context, workspaceID string, input ListDealsInput) ([]*Deal, error) {
	rows, err := s.selectDealRowsByFilter(ctx, workspaceID, input)
	if err != nil {
		return nil, fmt.Errorf("list deals: %w", err)
	}

	out := mapRows(rows, rowToDeal)
	sortDealsByCreatedAt(out, input.Sort)

	return out, nil
}

func (s *DealService) selectDealRowsByFilter(ctx context.Context, workspaceID string, input ListDealsInput) ([]sqlcgen.Deal, error) {
	if input.StageID != "" {
		return s.querier.ListDealsByStage(ctx, sqlcgen.ListDealsByStageParams{WorkspaceID: workspaceID, StageID: input.StageID})
	}
	if input.PipelineID != "" {
		return s.querier.ListDealsByPipeline(ctx, sqlcgen.ListDealsByPipelineParams{WorkspaceID: workspaceID, PipelineID: input.PipelineID})
	}
	if input.AccountID != "" {
		return s.querier.ListDealsByAccount(ctx, sqlcgen.ListDealsByAccountParams{WorkspaceID: workspaceID, AccountID: input.AccountID})
	}
	if input.OwnerID != "" {
		return s.querier.ListDealsByOwner(ctx, sqlcgen.ListDealsByOwnerParams{WorkspaceID: workspaceID, OwnerID: input.OwnerID})
	}
	if input.Status != "" {
		return s.querier.ListDealsByStatus(ctx, sqlcgen.ListDealsByStatusParams{WorkspaceID: workspaceID, Status: input.Status})
	}

	total, err := s.querier.CountDealsByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("count deals: %w", err)
	}
	return s.querier.ListDealsByWorkspace(ctx, sqlcgen.ListDealsByWorkspaceParams{
		WorkspaceID: workspaceID,
		Limit:       total,
		Offset:      0,
	})
}

func sortDealsByCreatedAt(items []*Deal, sortBy string) {
	if sortBy == sortCreatedAtAsc {
		sort.SliceStable(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
		return
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
}

func paginateDeals(items []*Deal, offset, limit int) []*Deal {
	if offset >= len(items) {
		return []*Deal{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
}

func (s *DealService) Update(ctx context.Context, workspaceID, dealID string, input UpdateDealInput) (*Deal, error) {
	if validationErr := validateDealInput(ctx, s.db, workspaceID, CreateDealInput{
		WorkspaceID:   workspaceID,
		AccountID:     input.AccountID,
		ContactID:     input.ContactID,
		PipelineID:    input.PipelineID,
		StageID:       input.StageID,
		OwnerID:       input.OwnerID,
		Title:         input.Title,
		Amount:        input.Amount,
		Currency:      input.Currency,
		ExpectedClose: input.ExpectedClose,
		Status:        input.Status,
		Metadata:      input.Metadata,
	}); validationErr != nil {
		return nil, validationErr
	}

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
		UpdatedAt:     nowRFC3339(),
		ID:            dealID,
		WorkspaceID:   workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("update deal: %w", err)
	}
	if timelineErr := createTimelineEvent(ctx, s.querier, workspaceID, timelineEntityDeal, dealID, input.OwnerID, timelineActionUpdated); timelineErr != nil {
		return nil, fmt.Errorf("update deal timeline: %w", timelineErr)
	}
	logCRMAudit(ctx, s.audit, workspaceID, input.OwnerID, actionDealUpdated, timelineEntityDeal, dealID)

	return s.Get(ctx, workspaceID, dealID)
}

func (s *DealService) Delete(ctx context.Context, workspaceID, dealID string) error {
	existing, err := s.Get(ctx, workspaceID, dealID)
	if err != nil {
		return err
	}

	now := nowRFC3339()
	err = s.querier.SoftDeleteDeal(ctx, sqlcgen.SoftDeleteDealParams{
		DeletedAt:   &now,
		UpdatedAt:   now,
		ID:          dealID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return fmt.Errorf("soft delete deal: %w", err)
	}
	if timelineErr := createTimelineEvent(ctx, s.querier, workspaceID, timelineEntityDeal, dealID, existing.OwnerID, timelineActionDeleted); timelineErr != nil {
		return fmt.Errorf("delete deal timeline: %w", timelineErr)
	}
	logCRMAudit(ctx, s.audit, workspaceID, existing.OwnerID, actionDealDeleted, timelineEntityDeal, dealID)
	return nil
}

func rowToDeal(row sqlcgen.Deal) *Deal {
	createdAt := parseRFC3339Time(row.CreatedAt)
	updatedAt := parseRFC3339Time(row.UpdatedAt)
	deletedAt := parseOptionalRFC3339(row.DeletedAt)

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
