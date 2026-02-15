package crm

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite/sqlcgen"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

type Pipeline struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspaceId"`
	Name        string    `json:"name"`
	EntityType  string    `json:"entityType"`
	Settings    *string   `json:"settings,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type PipelineStage struct {
	ID             string    `json:"id"`
	PipelineID     string    `json:"pipelineId"`
	Name           string    `json:"name"`
	Position       int64     `json:"position"`
	Probability    *float64  `json:"probability,omitempty"`
	SLAHours       *int64    `json:"slaHours,omitempty"`
	RequiredFields *string   `json:"requiredFields,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type CreatePipelineInput struct {
	WorkspaceID string
	Name        string
	EntityType  string
	Settings    string
}

type UpdatePipelineInput struct {
	Name       string
	EntityType string
	Settings   string
}

type ListPipelinesInput struct {
	Limit  int
	Offset int
}

type CreatePipelineStageInput struct {
	PipelineID     string
	Name           string
	Position       int64
	Probability    *float64
	SLAHours       *int64
	RequiredFields string
}

type UpdatePipelineStageInput struct {
	Name           string
	Position       int64
	Probability    *float64
	SLAHours       *int64
	RequiredFields string
}

type PipelineService struct {
	db      *sql.DB
	querier sqlcgen.Querier
}

func NewPipelineService(db *sql.DB) *PipelineService {
	return &PipelineService{db: db, querier: sqlcgen.New(db)}
}

func (s *PipelineService) Create(ctx context.Context, input CreatePipelineInput) (*Pipeline, error) {
	id := uuid.NewV7().String()
	now := time.Now().UTC().Format(time.RFC3339)
	err := s.querier.CreatePipeline(ctx, sqlcgen.CreatePipelineParams{
		ID:          id,
		WorkspaceID: input.WorkspaceID,
		Name:        input.Name,
		EntityType:  input.EntityType,
		Settings:    nullString(input.Settings),
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return nil, fmt.Errorf("create pipeline: %w", err)
	}
	return s.Get(ctx, input.WorkspaceID, id)
}

func (s *PipelineService) Get(ctx context.Context, workspaceID, pipelineID string) (*Pipeline, error) {
	row, err := s.querier.GetPipelineByID(ctx, sqlcgen.GetPipelineByIDParams{ID: pipelineID, WorkspaceID: workspaceID})
	if err != nil {
		return nil, err
	}
	return rowToPipeline(row), nil
}

func (s *PipelineService) List(ctx context.Context, workspaceID string, input ListPipelinesInput) ([]*Pipeline, int, error) {
	total, err := s.querier.CountPipelinesByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, 0, fmt.Errorf("count pipelines: %w", err)
	}
	rows, err := s.querier.ListPipelinesByWorkspace(ctx, sqlcgen.ListPipelinesByWorkspaceParams{
		WorkspaceID: workspaceID,
		Limit:       int64(input.Limit),
		Offset:      int64(input.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list pipelines: %w", err)
	}
	out := make([]*Pipeline, len(rows))
	for i := range rows {
		out[i] = rowToPipeline(rows[i])
	}
	return out, int(total), nil
}

func (s *PipelineService) Update(ctx context.Context, workspaceID, pipelineID string, input UpdatePipelineInput) (*Pipeline, error) {
	err := s.querier.UpdatePipeline(ctx, sqlcgen.UpdatePipelineParams{
		Name:        input.Name,
		EntityType:  input.EntityType,
		Settings:    nullString(input.Settings),
		UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
		ID:          pipelineID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("update pipeline: %w", err)
	}
	return s.Get(ctx, workspaceID, pipelineID)
}

func (s *PipelineService) Delete(ctx context.Context, workspaceID, pipelineID string) error {
	if err := s.querier.DeletePipeline(ctx, sqlcgen.DeletePipelineParams{ID: pipelineID, WorkspaceID: workspaceID}); err != nil {
		return fmt.Errorf("delete pipeline: %w", err)
	}
	return nil
}

func (s *PipelineService) CreateStage(ctx context.Context, input CreatePipelineStageInput) (*PipelineStage, error) {
	id := uuid.NewV7().String()
	now := time.Now().UTC().Format(time.RFC3339)
	err := s.querier.CreatePipelineStage(ctx, sqlcgen.CreatePipelineStageParams{
		ID:             id,
		PipelineID:     input.PipelineID,
		Name:           input.Name,
		Position:       input.Position,
		Probability:    input.Probability,
		SlaHours:       input.SLAHours,
		RequiredFields: nullString(input.RequiredFields),
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	if err != nil {
		return nil, fmt.Errorf("create pipeline stage: %w", err)
	}
	return s.GetStage(ctx, id)
}

func (s *PipelineService) GetStage(ctx context.Context, stageID string) (*PipelineStage, error) {
	row, err := s.querier.GetPipelineStageByID(ctx, stageID)
	if err != nil {
		return nil, err
	}
	return rowToPipelineStage(row), nil
}

func (s *PipelineService) ListStages(ctx context.Context, pipelineID string) ([]*PipelineStage, error) {
	rows, err := s.querier.ListPipelineStagesByPipeline(ctx, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("list pipeline stages: %w", err)
	}
	out := make([]*PipelineStage, len(rows))
	for i := range rows {
		out[i] = rowToPipelineStage(rows[i])
	}
	return out, nil
}

func (s *PipelineService) UpdateStage(ctx context.Context, stageID string, input UpdatePipelineStageInput) (*PipelineStage, error) {
	err := s.querier.UpdatePipelineStage(ctx, sqlcgen.UpdatePipelineStageParams{
		Name:           input.Name,
		Position:       input.Position,
		Probability:    input.Probability,
		SlaHours:       input.SLAHours,
		RequiredFields: nullString(input.RequiredFields),
		UpdatedAt:      time.Now().UTC().Format(time.RFC3339),
		ID:             stageID,
	})
	if err != nil {
		return nil, fmt.Errorf("update pipeline stage: %w", err)
	}
	return s.GetStage(ctx, stageID)
}

func (s *PipelineService) DeleteStage(ctx context.Context, stageID string) error {
	if err := s.querier.DeletePipelineStage(ctx, stageID); err != nil {
		return fmt.Errorf("delete pipeline stage: %w", err)
	}
	return nil
}

func rowToPipeline(row sqlcgen.Pipeline) *Pipeline {
	createdAt, _ := time.Parse(time.RFC3339, row.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, row.UpdatedAt)
	return &Pipeline{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceID,
		Name:        row.Name,
		EntityType:  row.EntityType,
		Settings:    row.Settings,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

func rowToPipelineStage(row sqlcgen.PipelineStage) *PipelineStage {
	createdAt, _ := time.Parse(time.RFC3339, row.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, row.UpdatedAt)
	return &PipelineStage{
		ID:             row.ID,
		PipelineID:     row.PipelineID,
		Name:           row.Name,
		Position:       row.Position,
		Probability:    row.Probability,
		SLAHours:       row.SlaHours,
		RequiredFields: row.RequiredFields,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}
}
