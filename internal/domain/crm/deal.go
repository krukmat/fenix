package crm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite/sqlcgen"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

type Deal struct {
	ID                string     `json:"id"`
	WorkspaceID       string     `json:"workspaceId"`
	AccountID         string     `json:"accountId"`
	ContactID         *string    `json:"contactId,omitempty"`
	PipelineID        string     `json:"pipelineId"`
	StageID           string     `json:"stageId"`
	OwnerID           string     `json:"ownerId"`
	Title             string     `json:"title"`
	Amount            *float64   `json:"amount,omitempty"`
	Currency          *string    `json:"currency,omitempty"`
	ExpectedClose     *string    `json:"expectedClose,omitempty"`
	Status            string     `json:"status"`
	Metadata          *string    `json:"metadata,omitempty"`
	ActiveSignalCount *int       `json:"active_signal_count,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
	DeletedAt         *time.Time `json:"deletedAt,omitempty"`
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
	bus     eventbus.EventBus
}

func NewDealService(db *sql.DB) *DealService {
	return &DealService{db: db, querier: sqlcgen.New(db), audit: newCRMAuditService(db)}
}

func NewDealServiceWithBus(db *sql.DB, bus eventbus.EventBus) *DealService {
	return &DealService{db: db, querier: sqlcgen.New(db), audit: newCRMAuditService(db), bus: bus}
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
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("get deal by id: %w", err)
	}
	return rowToDeal(row), nil
}

func (s *DealService) List(ctx context.Context, workspaceID string, input ListDealsInput) ([]*Deal, int, error) {
	return listInputFilteredOrPaged(
		&input,
		func(in *ListDealsInput) string { return in.Sort },
		func(in *ListDealsInput, sort string) { in.Sort = sort },
		sortCreatedAtDesc,
		shouldUseFilteredDealList,
		func() ([]*Deal, error) { return s.listFiltered(ctx, workspaceID, input) },
		paginateDeals,
		input.Offset, input.Limit,
		func() (int64, error) { return s.countDeals(ctx, workspaceID) },
		func() ([]*Deal, error) { return s.pageDeals(ctx, workspaceID, input) },
	)
}

func (s *DealService) countDeals(ctx context.Context, workspaceID string) (int64, error) {
	n, err := s.querier.CountDealsByWorkspace(ctx, workspaceID)
	if err != nil {
		return 0, fmt.Errorf("count deals: %w", err)
	}
	return n, nil
}

func (s *DealService) pageDeals(ctx context.Context, workspaceID string, input ListDealsInput) ([]*Deal, error) {
	rows, err := s.querier.ListDealsByWorkspace(ctx, sqlcgen.ListDealsByWorkspaceParams{
		WorkspaceID: workspaceID,
		Limit:       int64(input.Limit),
		Offset:      int64(input.Offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list deals: %w", err)
	}
	return mapRows(rows, rowToDeal), nil
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
	switch {
	case input.StageID != "":
		return s.listDealsByStage(ctx, workspaceID, input.StageID)
	case input.PipelineID != "":
		return s.listDealsByPipeline(ctx, workspaceID, input.PipelineID)
	case input.AccountID != "":
		return s.listDealsByAccount(ctx, workspaceID, input.AccountID)
	case input.OwnerID != "":
		return s.listDealsByOwner(ctx, workspaceID, input.OwnerID)
	case input.Status != "":
		return s.listDealsByStatus(ctx, workspaceID, input.Status)
	default:
		return s.listDealsByWorkspaceAll(ctx, workspaceID)
	}
}

func (s *DealService) listDealsByStage(ctx context.Context, workspaceID, stageID string) ([]sqlcgen.Deal, error) {
	rows, err := s.querier.ListDealsByStage(ctx, sqlcgen.ListDealsByStageParams{WorkspaceID: workspaceID, StageID: stageID})
	if err != nil {
		return nil, fmt.Errorf("list deals by stage: %w", err)
	}
	return rows, nil
}

func (s *DealService) listDealsByPipeline(ctx context.Context, workspaceID, pipelineID string) ([]sqlcgen.Deal, error) {
	rows, err := s.querier.ListDealsByPipeline(ctx, sqlcgen.ListDealsByPipelineParams{WorkspaceID: workspaceID, PipelineID: pipelineID})
	if err != nil {
		return nil, fmt.Errorf("list deals by pipeline: %w", err)
	}
	return rows, nil
}

func (s *DealService) listDealsByAccount(ctx context.Context, workspaceID, accountID string) ([]sqlcgen.Deal, error) {
	rows, err := s.querier.ListDealsByAccount(ctx, sqlcgen.ListDealsByAccountParams{WorkspaceID: workspaceID, AccountID: accountID})
	if err != nil {
		return nil, fmt.Errorf("list deals by account: %w", err)
	}
	return rows, nil
}

func (s *DealService) listDealsByOwner(ctx context.Context, workspaceID, ownerID string) ([]sqlcgen.Deal, error) {
	rows, err := s.querier.ListDealsByOwner(ctx, sqlcgen.ListDealsByOwnerParams{WorkspaceID: workspaceID, OwnerID: ownerID})
	if err != nil {
		return nil, fmt.Errorf("list deals by owner: %w", err)
	}
	return rows, nil
}

func (s *DealService) listDealsByStatus(ctx context.Context, workspaceID, status string) ([]sqlcgen.Deal, error) {
	rows, err := s.querier.ListDealsByStatus(ctx, sqlcgen.ListDealsByStatusParams{WorkspaceID: workspaceID, Status: status})
	if err != nil {
		return nil, fmt.Errorf("list deals by status: %w", err)
	}
	return rows, nil
}

func (s *DealService) listDealsByWorkspaceAll(ctx context.Context, workspaceID string) ([]sqlcgen.Deal, error) {
	total, err := s.querier.CountDealsByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("count deals: %w", err)
	}
	rows, err := s.querier.ListDealsByWorkspace(ctx, sqlcgen.ListDealsByWorkspaceParams{
		WorkspaceID: workspaceID,
		Limit:       total,
		Offset:      0,
	})
	if err != nil {
		return nil, fmt.Errorf("list deals by workspace: %w", err)
	}
	return rows, nil
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
	deal, getErr := s.Get(ctx, workspaceID, dealID)
	if getErr != nil {
		return nil, getErr
	}
	publishDealUpdated(s.bus, deal)
	return deal, nil
}

func (s *DealService) Delete(ctx context.Context, workspaceID, dealID string) error {
	existing, err := s.Get(ctx, workspaceID, dealID)
	if err != nil {
		return err
	}
	now := nowRFC3339()
	return softDeleteWithSideEffects(ctx, s.querier, s.audit, workspaceID, timelineEntityDeal, dealID, existing.OwnerID, actionDealDeleted,
		func() error {
			return s.querier.SoftDeleteDeal(ctx, sqlcgen.SoftDeleteDealParams{
				DeletedAt: &now, UpdatedAt: now, ID: dealID, WorkspaceID: workspaceID,
			})
		})
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
