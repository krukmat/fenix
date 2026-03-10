package workflow

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

const (
	maxSourceSizeBytes = 64 * 1024
)

var (
	ErrInvalidWorkflowInput    = errors.New("invalid workflow input")
	ErrWorkflowNotEditable     = errors.New("workflow is not editable")
	ErrWorkflowNameConflict    = errors.New("workflow name/version conflict")
	ErrWorkflowActiveConflict  = errors.New("workflow active version conflict")
	ErrInvalidStatusTransition = errors.New("invalid workflow status transition")
	ErrWorkflowVersionInvalid  = errors.New("invalid workflow version operation")
	ErrWorkflowDeleteInvalid   = errors.New("invalid workflow delete operation")
)

type CreateWorkflowInput struct {
	WorkspaceID       string
	AgentDefinitionID *string
	Name              string
	Description       string
	DSLSource         string
	SpecSource        string
	CreatedByUserID   *string
}

type UpdateWorkflowInput struct {
	AgentDefinitionID *string
	Description       string
	DSLSource         string
	SpecSource        string
}

type ListWorkflowsInput struct {
	Status *Status
	Name   string
}

type Service struct {
	repo *Repository
}

func NewService(db *sql.DB) *Service {
	return &Service{repo: NewRepository(db)}
}

func NewServiceWithRepository(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, input CreateWorkflowInput) (*Workflow, error) {
	if err := validateCreateInput(input); err != nil {
		return nil, err
	}

	desc := trimOptionalString(input.Description)
	spec := trimOptionalString(input.SpecSource)

	workflow, err := s.repo.Create(ctx, CreateInput{
		ID:                uuid.NewV7().String(),
		WorkspaceID:       input.WorkspaceID,
		AgentDefinitionID: input.AgentDefinitionID,
		Name:              strings.TrimSpace(input.Name),
		Description:       desc,
		DSLSource:         input.DSLSource,
		SpecSource:        spec,
		Version:           1,
		Status:            StatusDraft,
		CreatedByUserID:   input.CreatedByUserID,
	})
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, ErrWorkflowNameConflict
		}
		return nil, err
	}
	return workflow, nil
}

func (s *Service) Get(ctx context.Context, workspaceID, workflowID string) (*Workflow, error) {
	return s.repo.GetByID(ctx, workspaceID, workflowID)
}

func (s *Service) GetActiveByAgentDefinition(ctx context.Context, workspaceID, agentDefinitionID string) (*Workflow, error) {
	return s.repo.GetActiveByAgentDefinition(ctx, workspaceID, agentDefinitionID)
}

func (s *Service) List(ctx context.Context, workspaceID string, input ListWorkflowsInput) ([]*Workflow, error) {
	workflows, err := s.listBase(ctx, workspaceID, input)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(input.Name) == "" {
		return workflows, nil
	}

	name := strings.TrimSpace(input.Name)
	out := make([]*Workflow, 0, len(workflows))
	for _, workflow := range workflows {
		if workflow.Name == name {
			out = append(out, workflow)
		}
	}
	return out, nil
}

func (s *Service) Update(ctx context.Context, workspaceID, workflowID string, input UpdateWorkflowInput) (*Workflow, error) {
	if err := validateUpdateInput(input); err != nil {
		return nil, err
	}

	existing, err := s.repo.GetByID(ctx, workspaceID, workflowID)
	if err != nil {
		return nil, err
	}
	if existing.Status != StatusDraft {
		return nil, ErrWorkflowNotEditable
	}

	desc := trimOptionalString(input.Description)
	spec := trimOptionalString(input.SpecSource)

	updated, err := s.repo.Update(ctx, workspaceID, workflowID, UpdateInput{
		AgentDefinitionID: input.AgentDefinitionID,
		Description:       desc,
		DSLSource:         input.DSLSource,
		SpecSource:        spec,
		Status:            existing.Status,
		ArchivedAt:        existing.ArchivedAt,
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Service) SetStatus(ctx context.Context, workspaceID, workflowID string, next Status) (*Workflow, error) {
	existing, err := s.repo.GetByID(ctx, workspaceID, workflowID)
	if err != nil {
		return nil, err
	}

	if !isValidStatusTransition(existing.Status, next) {
		return nil, ErrInvalidStatusTransition
	}
	if next == StatusActive {
		if err = s.ensureNoOtherActiveWorkflow(ctx, existing); err != nil {
			return nil, err
		}
	}

	updated, err := s.repo.Update(ctx, workspaceID, workflowID, UpdateInput{
		AgentDefinitionID: existing.AgentDefinitionID,
		Description:       existing.Description,
		DSLSource:         existing.DSLSource,
		SpecSource:        existing.SpecSource,
		Status:            next,
		ArchivedAt:        archivedAtForStatus(next, existing.ArchivedAt),
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Service) MarkTesting(ctx context.Context, workspaceID, workflowID string) (*Workflow, error) {
	return s.SetStatus(ctx, workspaceID, workflowID, StatusTesting)
}

func (s *Service) MarkActive(ctx context.Context, workspaceID, workflowID string) (*Workflow, error) {
	return s.SetStatus(ctx, workspaceID, workflowID, StatusActive)
}

func (s *Service) MarkArchived(ctx context.Context, workspaceID, workflowID string) (*Workflow, error) {
	return s.SetStatus(ctx, workspaceID, workflowID, StatusArchived)
}

func (s *Service) NewVersion(ctx context.Context, workspaceID, workflowID string) (*Workflow, error) {
	existing, err := s.repo.GetByID(ctx, workspaceID, workflowID)
	if err != nil {
		return nil, err
	}
	if existing.Status != StatusActive {
		return nil, ErrWorkflowVersionInvalid
	}

	parentID := existing.ID
	next, err := s.repo.Create(ctx, CreateInput{
		ID:                uuid.NewV7().String(),
		WorkspaceID:       existing.WorkspaceID,
		AgentDefinitionID: existing.AgentDefinitionID,
		ParentVersionID:   &parentID,
		Name:              existing.Name,
		Description:       cloneOptionalString(existing.Description),
		DSLSource:         existing.DSLSource,
		SpecSource:        cloneOptionalString(existing.SpecSource),
		Version:           existing.Version + 1,
		Status:            StatusDraft,
		CreatedByUserID:   existing.CreatedByUserID,
	})
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, ErrWorkflowNameConflict
		}
		return nil, err
	}
	return next, nil
}

func (s *Service) Rollback(ctx context.Context, workspaceID, workflowID string) (*Workflow, error) {
	existing, err := s.repo.GetByID(ctx, workspaceID, workflowID)
	if err != nil {
		return nil, err
	}
	if existing.Status != StatusArchived {
		return nil, ErrWorkflowVersionInvalid
	}
	return s.MarkActive(ctx, workspaceID, workflowID)
}

func (s *Service) DeleteDraft(ctx context.Context, workspaceID, workflowID string) error {
	existing, err := s.repo.GetByID(ctx, workspaceID, workflowID)
	if err != nil {
		return err
	}
	if existing.Status != StatusDraft {
		return ErrWorkflowDeleteInvalid
	}
	return s.repo.Delete(ctx, workspaceID, workflowID)
}

func (s *Service) listBase(ctx context.Context, workspaceID string, input ListWorkflowsInput) ([]*Workflow, error) {
	if input.Status != nil {
		return s.repo.ListByStatus(ctx, workspaceID, *input.Status)
	}
	return s.repo.ListByWorkspace(ctx, workspaceID)
}

func validateCreateInput(input CreateWorkflowInput) error {
	if strings.TrimSpace(input.WorkspaceID) == "" {
		return invalidWorkflowInput("workspace_id is required", nil)
	}
	if strings.TrimSpace(input.Name) == "" {
		return invalidWorkflowInput("name is required", nil)
	}
	if err := validateDSLSource(input.DSLSource); err != nil {
		return err
	}
	if err := validateOptionalSourceSize("spec_source", input.SpecSource); err != nil {
		return err
	}
	return nil
}

func validateUpdateInput(input UpdateWorkflowInput) error {
	if err := validateDSLSource(input.DSLSource); err != nil {
		return err
	}
	if err := validateOptionalSourceSize("spec_source", input.SpecSource); err != nil {
		return err
	}
	return nil
}

func validateDSLSource(source string) error {
	if strings.TrimSpace(source) == "" {
		return invalidWorkflowInput("dsl_source is required", nil)
	}
	if err := validateOptionalSourceSize("dsl_source", source); err != nil {
		return err
	}
	return nil
}

func validateOptionalSourceSize(field, source string) error {
	if len(source) > maxSourceSizeBytes {
		return invalidWorkflowInput(fmt.Sprintf("%s exceeds %d bytes", field, maxSourceSizeBytes), nil)
	}
	return nil
}

func invalidWorkflowInput(reason string, err error) error {
	if err == nil {
		return fmt.Errorf("%w: %s", ErrInvalidWorkflowInput, reason)
	}
	return fmt.Errorf("%w: %s: %w", ErrInvalidWorkflowInput, reason, err)
}

func trimOptionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func cloneOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func (s *Service) ensureNoOtherActiveWorkflow(ctx context.Context, workflow *Workflow) error {
	active, err := s.repo.GetActiveByName(ctx, workflow.WorkspaceID, workflow.Name)
	if err != nil {
		if errors.Is(err, ErrWorkflowNotFound) {
			return nil
		}
		return err
	}
	if active.ID != workflow.ID {
		return ErrWorkflowActiveConflict
	}
	return nil
}

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func isValidStatusTransition(current, next Status) bool {
	if current == next {
		return true
	}

	switch current {
	case StatusDraft:
		return next == StatusTesting
	case StatusTesting:
		return next == StatusDraft || next == StatusActive
	case StatusActive:
		return next == StatusArchived
	case StatusArchived:
		return next == StatusActive
	default:
		return false
	}
}

func archivedAtForStatus(next Status, existing *time.Time) *time.Time {
	if next == StatusArchived {
		now := time.Now().UTC()
		return &now
	}
	return existing
}
