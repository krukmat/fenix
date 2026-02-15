package crm

import (
	"context"
	"database/sql"
	"fmt"
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
	Limit  int
	Offset int
}

type CaseService struct {
	db      *sql.DB
	querier sqlcgen.Querier
	bus     eventbus.EventBus
}

func NewCaseService(db *sql.DB) *CaseService {
	return &CaseService{db: db, querier: sqlcgen.New(db)}
}

func NewCaseServiceWithBus(db *sql.DB, bus eventbus.EventBus) *CaseService {
	return &CaseService{db: db, querier: sqlcgen.New(db), bus: bus}
}

func (s *CaseService) Create(ctx context.Context, input CreateCaseInput) (*CaseTicket, error) {
	id := uuid.NewV7().String()
	now := time.Now().UTC().Format(time.RFC3339)
	priority := input.Priority
	if priority == "" {
		priority = "medium"
	}
	status := input.Status
	if status == "" {
		status = "open"
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
	if err := createTimelineEvent(ctx, s.querier, input.WorkspaceID, "case_ticket", id, input.OwnerID, "created"); err != nil {
		return nil, fmt.Errorf("create case timeline: %w", err)
	}
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
	total, err := s.querier.CountCasesByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, 0, fmt.Errorf("count cases: %w", err)
	}

	rows, err := s.querier.ListCasesByWorkspace(ctx, sqlcgen.ListCasesByWorkspaceParams{
		WorkspaceID: workspaceID,
		Limit:       int64(input.Limit),
		Offset:      int64(input.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list cases: %w", err)
	}

	out := make([]*CaseTicket, len(rows))
	for i := range rows {
		out[i] = rowToCaseTicket(rows[i])
	}

	return out, int(total), nil
}

func (s *CaseService) Update(ctx context.Context, workspaceID, caseID string, input UpdateCaseInput) (*CaseTicket, error) {
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
		UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
		ID:          caseID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("update case: %w", err)
	}
	if err := createTimelineEvent(ctx, s.querier, workspaceID, "case_ticket", caseID, input.OwnerID, "updated"); err != nil {
		return nil, fmt.Errorf("update case timeline: %w", err)
	}
	s.publishRecordChanged(knowledge.ChangeTypeUpdated, workspaceID, caseID)

	return s.Get(ctx, workspaceID, caseID)
}

func (s *CaseService) Delete(ctx context.Context, workspaceID, caseID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	err := s.querier.SoftDeleteCase(ctx, sqlcgen.SoftDeleteCaseParams{
		DeletedAt:   &now,
		UpdatedAt:   now,
		ID:          caseID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return fmt.Errorf("soft delete case: %w", err)
	}
	if err := createTimelineEvent(ctx, s.querier, workspaceID, "case_ticket", caseID, "", "deleted"); err != nil {
		return fmt.Errorf("delete case timeline: %w", err)
	}
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
	createdAt, _ := time.Parse(time.RFC3339, row.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, row.UpdatedAt)

	var deletedAt *time.Time
	if row.DeletedAt != nil {
		t, _ := time.Parse(time.RFC3339, *row.DeletedAt)
		deletedAt = &t
	}

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
