package crm

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite/sqlcgen"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

type CaseTicket struct {
	ID          string     `json:"id"`
	WorkspaceID string     `json:"workspaceId"`
	AccountID   *string    `json:"accountId,omitempty"`
	ContactID   *string    `json:"contactId,omitempty"`
	PipelineID  *string    `json:"pipelineId,omitempty"`
	StageID     *string    `json:"stageId,omitempty"`
	OwnerID     string     `json:"ownerId"`
	Subject     string     `json:"subject"`
	Description *string    `json:"description,omitempty"`
	Priority    string     `json:"priority"`
	Status      string     `json:"status"`
	Channel     *string    `json:"channel,omitempty"`
	SLAConfig   *string    `json:"slaConfig,omitempty"`
	SLADeadline *string    `json:"slaDeadline,omitempty"`
	Metadata    *string    `json:"metadata,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty"`
}

type CreateCaseInput struct {
	WorkspaceID string
	AccountID   string
	ContactID   string
	PipelineID  string
	StageID     string
	OwnerID     string
	Subject     string
	Description string
	Priority    string
	Status      string
	Channel     string
	SLAConfig   string
	SLADeadline string
	Metadata    string
}

type UpdateCaseInput struct {
	AccountID   string
	ContactID   string
	PipelineID  string
	StageID     string
	OwnerID     string
	Subject     string
	Description string
	Priority    string
	Status      string
	Channel     string
	SLAConfig   string
	SLADeadline string
	Metadata    string
}

type ListCasesInput struct {
	Limit     int
	Offset    int
	Status    string
	Priority  string
	OwnerID   string
	AccountID string
	Sort      string
}

const (
	caseSortCreatedAtAsc  = "created_at"
	caseSortCreatedAtDesc = "-created_at"
)

type CaseService struct {
	db      *sql.DB
	querier sqlcgen.Querier
	bus     eventbus.EventBus
	audit   auditLogger
}

func NewCaseService(db *sql.DB) *CaseService {
	return &CaseService{db: db, querier: sqlcgen.New(db), audit: newCRMAuditService(db)}
}

func NewCaseServiceWithBus(db *sql.DB, bus eventbus.EventBus) *CaseService {
	return &CaseService{db: db, querier: sqlcgen.New(db), bus: bus, audit: newCRMAuditService(db)}
}

func (s *CaseService) Create(ctx context.Context, input CreateCaseInput) (*CaseTicket, error) {
	id := uuid.NewV7().String()
	now := nowRFC3339()
	priority := input.Priority
	if priority == "" {
		priority = "medium"
	}
	status := input.Status
	if status == "" {
		status = "open"
	}
	input.Priority = priority
	input.Status = status
	if validationErr := validateCaseInput(ctx, s.db, input.WorkspaceID, input); validationErr != nil {
		return nil, validationErr
	}

	err := s.querier.CreateCase(ctx, sqlcgen.CreateCaseParams{
		ID:          id,
		WorkspaceID: input.WorkspaceID,
		AccountID:   nullString(input.AccountID),
		ContactID:   nullString(input.ContactID),
		PipelineID:  nullString(input.PipelineID),
		StageID:     nullString(input.StageID),
		OwnerID:     input.OwnerID,
		Subject:     input.Subject,
		Description: nullString(input.Description),
		Priority:    priority,
		Status:      status,
		Channel:     nullString(input.Channel),
		SlaConfig:   nullString(input.SLAConfig),
		SlaDeadline: nullString(input.SLADeadline),
		Metadata:    nullString(input.Metadata),
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return nil, fmt.Errorf("create case: %w", err)
	}
	if timelineErr := createTimelineEvent(ctx, s.querier, input.WorkspaceID, timelineEntityCase, id, input.OwnerID, timelineActionCreated); timelineErr != nil {
		return nil, fmt.Errorf("create case timeline: %w", timelineErr)
	}
	logCRMAudit(ctx, s.audit, input.WorkspaceID, input.OwnerID, actionCaseCreated, timelineEntityCase, id)
	s.publishRecordChanged(knowledge.ChangeTypeCreated, input.WorkspaceID, id)

	return s.Get(ctx, input.WorkspaceID, id)
}

func (s *CaseService) Get(ctx context.Context, workspaceID, caseID string) (*CaseTicket, error) {
	row, err := s.querier.GetCaseByID(ctx, sqlcgen.GetCaseByIDParams{ID: caseID, WorkspaceID: workspaceID})
	if err != nil {
		return nil, err
	}
	return rowToCaseTicket(row), nil
}

func (s *CaseService) List(ctx context.Context, workspaceID string, input ListCasesInput) ([]*CaseTicket, int, error) {
	input.Sort = firstNonEmpty(input.Sort, caseSortCreatedAtDesc)
	if shouldUseFilteredCaseList(input) {
		filtered, err := s.listFiltered(ctx, workspaceID, input)
		if err != nil {
			return nil, 0, err
		}
		return paginateCases(filtered, input.Offset, input.Limit), len(filtered), nil
	}
	return s.listPage(ctx, workspaceID, input)
}

func (s *CaseService) listPage(ctx context.Context, workspaceID string, input ListCasesInput) ([]*CaseTicket, int, error) {
	total, err := s.countCases(ctx, workspaceID)
	if err != nil {
		return nil, 0, err
	}
	items, err := s.pageCases(ctx, workspaceID, input)
	if err != nil {
		return nil, 0, err
	}
	return items, int(total), nil
}

func (s *CaseService) countCases(ctx context.Context, workspaceID string) (int64, error) {
	n, err := s.querier.CountCasesByWorkspace(ctx, workspaceID)
	if err != nil {
		return 0, fmt.Errorf("count cases: %w", err)
	}
	return n, nil
}

func (s *CaseService) pageCases(ctx context.Context, workspaceID string, input ListCasesInput) ([]*CaseTicket, error) {
	rows, err := s.querier.ListCasesByWorkspace(ctx, sqlcgen.ListCasesByWorkspaceParams{
		WorkspaceID: workspaceID,
		Limit:       int64(input.Limit),
		Offset:      int64(input.Offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list cases: %w", err)
	}
	return mapRows(rows, rowToCaseTicket), nil
}

func shouldUseFilteredCaseList(input ListCasesInput) bool {
	return input.Status != "" || input.Priority != "" || input.OwnerID != "" || input.AccountID != "" || input.Sort != caseSortCreatedAtDesc
}

func (s *CaseService) listFiltered(ctx context.Context, workspaceID string, input ListCasesInput) ([]*CaseTicket, error) {
	rows, err := s.selectCaseRowsByFilter(ctx, workspaceID, input)
	if err != nil {
		return nil, fmt.Errorf("list cases: %w", err)
	}

	out := mapRows(rows, rowToCaseTicket)
	out = filterCasesByPriority(out, input.Priority)
	sortCasesByCreatedAt(out, input.Sort)

	return out, nil
}

func (s *CaseService) selectCaseRowsByFilter(ctx context.Context, workspaceID string, input ListCasesInput) ([]sqlcgen.CaseTicket, error) {
	if input.AccountID != "" {
		return s.querier.ListCasesByAccount(ctx, sqlcgen.ListCasesByAccountParams{WorkspaceID: workspaceID, AccountID: nullString(input.AccountID)})
	}
	if input.OwnerID != "" {
		return s.querier.ListCasesByOwner(ctx, sqlcgen.ListCasesByOwnerParams{WorkspaceID: workspaceID, OwnerID: input.OwnerID})
	}
	if input.Status != "" {
		return s.querier.ListCasesByStatus(ctx, sqlcgen.ListCasesByStatusParams{WorkspaceID: workspaceID, Status: input.Status})
	}

	total, err := s.querier.CountCasesByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("count cases: %w", err)
	}
	return s.querier.ListCasesByWorkspace(ctx, sqlcgen.ListCasesByWorkspaceParams{
		WorkspaceID: workspaceID,
		Limit:       total,
		Offset:      0,
	})
}

func filterCasesByPriority(items []*CaseTicket, priority string) []*CaseTicket {
	if priority == "" {
		return items
	}
	filtered := make([]*CaseTicket, 0, len(items))
	for _, item := range items {
		if item.Priority == priority {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func sortCasesByCreatedAt(items []*CaseTicket, sortBy string) {
	if sortBy == caseSortCreatedAtAsc {
		sort.SliceStable(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
		return
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
}

func paginateCases(items []*CaseTicket, offset, limit int) []*CaseTicket {
	if offset >= len(items) {
		return []*CaseTicket{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
}

func (s *CaseService) Update(ctx context.Context, workspaceID, caseID string, input UpdateCaseInput) (*CaseTicket, error) {
	if validationErr := validateCaseInput(ctx, s.db, workspaceID, CreateCaseInput{
		WorkspaceID: workspaceID,
		AccountID:   input.AccountID,
		ContactID:   input.ContactID,
		PipelineID:  input.PipelineID,
		StageID:     input.StageID,
		OwnerID:     input.OwnerID,
		Subject:     input.Subject,
		Description: input.Description,
		Priority:    input.Priority,
		Status:      input.Status,
		Channel:     input.Channel,
		SLAConfig:   input.SLAConfig,
		SLADeadline: input.SLADeadline,
		Metadata:    input.Metadata,
	}); validationErr != nil {
		return nil, validationErr
	}

	err := s.querier.UpdateCase(ctx, sqlcgen.UpdateCaseParams{
		AccountID:   nullString(input.AccountID),
		ContactID:   nullString(input.ContactID),
		PipelineID:  nullString(input.PipelineID),
		StageID:     nullString(input.StageID),
		OwnerID:     input.OwnerID,
		Subject:     input.Subject,
		Description: nullString(input.Description),
		Priority:    input.Priority,
		Status:      input.Status,
		Channel:     nullString(input.Channel),
		SlaConfig:   nullString(input.SLAConfig),
		SlaDeadline: nullString(input.SLADeadline),
		Metadata:    nullString(input.Metadata),
		UpdatedAt:   nowRFC3339(),
		ID:          caseID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("update case: %w", err)
	}
	if timelineErr := createTimelineEvent(ctx, s.querier, workspaceID, timelineEntityCase, caseID, input.OwnerID, timelineActionUpdated); timelineErr != nil {
		return nil, fmt.Errorf("update case timeline: %w", timelineErr)
	}
	logCRMAudit(ctx, s.audit, workspaceID, input.OwnerID, actionCaseUpdated, timelineEntityCase, caseID)
	s.publishRecordChanged(knowledge.ChangeTypeUpdated, workspaceID, caseID)

	return s.Get(ctx, workspaceID, caseID)
}

func (s *CaseService) Delete(ctx context.Context, workspaceID, caseID string) error {
	existing, err := s.Get(ctx, workspaceID, caseID)
	if err != nil {
		return err
	}

	now := nowRFC3339()
	err = s.querier.SoftDeleteCase(ctx, sqlcgen.SoftDeleteCaseParams{
		DeletedAt:   &now,
		UpdatedAt:   now,
		ID:          caseID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return fmt.Errorf("soft delete case: %w", err)
	}
	if timelineErr := createTimelineEvent(ctx, s.querier, workspaceID, timelineEntityCase, caseID, existing.OwnerID, timelineActionDeleted); timelineErr != nil {
		return fmt.Errorf("delete case timeline: %w", timelineErr)
	}
	logCRMAudit(ctx, s.audit, workspaceID, existing.OwnerID, actionCaseDeleted, timelineEntityCase, caseID)
	s.publishRecordChanged(knowledge.ChangeTypeDeleted, workspaceID, caseID)
	return nil
}

func (s *CaseService) publishRecordChanged(changeType knowledge.ChangeType, workspaceID, caseID string) {
	if s.bus == nil {
		return
	}
	s.bus.Publish(knowledge.TopicForChangeType(changeType), knowledge.RecordChangedEvent{
		EntityType:  knowledge.EntityTypeCaseTicket,
		EntityID:    caseID,
		WorkspaceID: workspaceID,
		ChangeType:  changeType,
		OccurredAt:  time.Now(),
	})
}

func rowToCaseTicket(row sqlcgen.CaseTicket) *CaseTicket {
	createdAt := parseRFC3339Time(row.CreatedAt)
	updatedAt := parseRFC3339Time(row.UpdatedAt)
	deletedAt := parseOptionalRFC3339(row.DeletedAt)

	return &CaseTicket{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceID,
		AccountID:   row.AccountID,
		ContactID:   row.ContactID,
		PipelineID:  row.PipelineID,
		StageID:     row.StageID,
		OwnerID:     row.OwnerID,
		Subject:     row.Subject,
		Description: row.Description,
		Priority:    row.Priority,
		Status:      row.Status,
		Channel:     row.Channel,
		SLAConfig:   row.SlaConfig,
		SLADeadline: row.SlaDeadline,
		Metadata:    row.Metadata,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		DeletedAt:   deletedAt,
	}
}
