package workflow

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrWorkflowNotFound = errors.New("workflow not found")
)

type Status string

const (
	StatusDraft    Status = "draft"
	StatusTesting  Status = "testing"
	StatusActive   Status = "active"
	StatusArchived Status = "archived"
)

type Workflow struct {
	ID                string     `json:"id"`
	WorkspaceID       string     `json:"workspaceId"`
	AgentDefinitionID *string    `json:"agentDefinitionId,omitempty"`
	ParentVersionID   *string    `json:"parentVersionId,omitempty"`
	Name              string     `json:"name"`
	Description       *string    `json:"description,omitempty"`
	DSLSource         string     `json:"dslSource"`
	SpecSource        *string    `json:"specSource,omitempty"`
	Version           int        `json:"version"`
	Status            Status     `json:"status"`
	CreatedByUserID   *string    `json:"createdByUserId,omitempty"`
	ArchivedAt        *time.Time `json:"archivedAt,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

type CreateInput struct {
	ID                string
	WorkspaceID       string
	AgentDefinitionID *string
	ParentVersionID   *string
	Name              string
	Description       *string
	DSLSource         string
	SpecSource        *string
	Version           int
	Status            Status
	CreatedByUserID   *string
	ArchivedAt        *time.Time
}

type UpdateInput struct {
	AgentDefinitionID *string
	Description       *string
	DSLSource         string
	SpecSource        *string
	Status            Status
	ArchivedAt        *time.Time
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, input CreateInput) (*Workflow, error) {
	now := nowRFC3339()
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO workflow (
			id, workspace_id, agent_definition_id, parent_version_id, name, description,
			dsl_source, spec_source, version, status, created_by_user_id, archived_at,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING
			id, workspace_id, agent_definition_id, parent_version_id, name, description,
			dsl_source, spec_source, version, status, created_by_user_id, archived_at,
			created_at, updated_at
	`,
		input.ID,
		input.WorkspaceID,
		input.AgentDefinitionID,
		input.ParentVersionID,
		input.Name,
		input.Description,
		input.DSLSource,
		input.SpecSource,
		input.Version,
		string(input.Status),
		input.CreatedByUserID,
		formatOptionalTime(input.ArchivedAt),
		now,
		now,
	)

	out, err := scanWorkflow(row)
	if err != nil {
		return nil, fmt.Errorf("create workflow: %w", err)
	}
	return out, nil
}

func (r *Repository) GetByID(ctx context.Context, workspaceID, workflowID string) (*Workflow, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			id, workspace_id, agent_definition_id, parent_version_id, name, description,
			dsl_source, spec_source, version, status, created_by_user_id, archived_at,
			created_at, updated_at
		FROM workflow
		WHERE id = ? AND workspace_id = ?
		LIMIT 1
	`, workflowID, workspaceID)

	out, err := scanWorkflow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWorkflowNotFound
		}
		return nil, fmt.Errorf("get workflow by id: %w", err)
	}

	return out, nil
}

func (r *Repository) GetByNameAndVersion(ctx context.Context, workspaceID, name string, version int) (*Workflow, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			id, workspace_id, agent_definition_id, parent_version_id, name, description,
			dsl_source, spec_source, version, status, created_by_user_id, archived_at,
			created_at, updated_at
		FROM workflow
		WHERE workspace_id = ? AND name = ? AND version = ?
		LIMIT 1
	`, workspaceID, name, version)

	out, err := scanWorkflow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWorkflowNotFound
		}
		return nil, fmt.Errorf("get workflow by name and version: %w", err)
	}

	return out, nil
}

func (r *Repository) GetActiveByName(ctx context.Context, workspaceID, name string) (*Workflow, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			id, workspace_id, agent_definition_id, parent_version_id, name, description,
			dsl_source, spec_source, version, status, created_by_user_id, archived_at,
			created_at, updated_at
		FROM workflow
		WHERE workspace_id = ? AND name = ? AND status = 'active'
		LIMIT 1
	`, workspaceID, name)

	out, err := scanWorkflow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWorkflowNotFound
		}
		return nil, fmt.Errorf("get active workflow by name: %w", err)
	}

	return out, nil
}

func (r *Repository) GetActiveByAgentDefinition(ctx context.Context, workspaceID, agentDefinitionID string) (*Workflow, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			id, workspace_id, agent_definition_id, parent_version_id, name, description,
			dsl_source, spec_source, version, status, created_by_user_id, archived_at,
			created_at, updated_at
		FROM workflow
		WHERE workspace_id = ? AND agent_definition_id = ? AND status = 'active'
		ORDER BY updated_at DESC, created_at DESC
		LIMIT 1
	`, workspaceID, agentDefinitionID)

	out, err := scanWorkflow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWorkflowNotFound
		}
		return nil, fmt.Errorf("get active workflow by agent definition: %w", err)
	}

	return out, nil
}

func (r *Repository) ListByWorkspace(ctx context.Context, workspaceID string) ([]*Workflow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id, workspace_id, agent_definition_id, parent_version_id, name, description,
			dsl_source, spec_source, version, status, created_by_user_id, archived_at,
			created_at, updated_at
		FROM workflow
		WHERE workspace_id = ?
		ORDER BY created_at DESC
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list workflows by workspace: %w", err)
	}
	defer rows.Close()

	return scanWorkflowRows(rows)
}

func (r *Repository) ListByStatus(ctx context.Context, workspaceID string, status Status) ([]*Workflow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id, workspace_id, agent_definition_id, parent_version_id, name, description,
			dsl_source, spec_source, version, status, created_by_user_id, archived_at,
			created_at, updated_at
		FROM workflow
		WHERE workspace_id = ? AND status = ?
		ORDER BY created_at DESC
	`, workspaceID, string(status))
	if err != nil {
		return nil, fmt.Errorf("list workflows by status: %w", err)
	}
	defer rows.Close()

	return scanWorkflowRows(rows)
}

func (r *Repository) ListVersionsByName(ctx context.Context, workspaceID, name string) ([]*Workflow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id, workspace_id, agent_definition_id, parent_version_id, name, description,
			dsl_source, spec_source, version, status, created_by_user_id, archived_at,
			created_at, updated_at
		FROM workflow
		WHERE workspace_id = ? AND name = ?
		ORDER BY version DESC, created_at DESC
	`, workspaceID, name)
	if err != nil {
		return nil, fmt.Errorf("list workflow versions: %w", err)
	}
	defer rows.Close()

	return scanWorkflowRows(rows)
}

func (r *Repository) Update(ctx context.Context, workspaceID, workflowID string, input UpdateInput) (*Workflow, error) {
	row := r.db.QueryRowContext(ctx, `
		UPDATE workflow
		SET agent_definition_id = ?, description = ?, dsl_source = ?, spec_source = ?,
			status = ?, archived_at = ?, updated_at = ?
		WHERE id = ? AND workspace_id = ?
		RETURNING
			id, workspace_id, agent_definition_id, parent_version_id, name, description,
			dsl_source, spec_source, version, status, created_by_user_id, archived_at,
			created_at, updated_at
	`,
		input.AgentDefinitionID,
		input.Description,
		input.DSLSource,
		input.SpecSource,
		string(input.Status),
		formatOptionalTime(input.ArchivedAt),
		nowRFC3339(),
		workflowID,
		workspaceID,
	)

	out, err := scanWorkflow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWorkflowNotFound
		}
		return nil, fmt.Errorf("update workflow: %w", err)
	}

	return out, nil
}

func (r *Repository) Delete(ctx context.Context, workspaceID, workflowID string) error {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM workflow
		WHERE id = ? AND workspace_id = ?
	`, workflowID, workspaceID)
	if err != nil {
		return fmt.Errorf("delete workflow: %w", err)
	}
	rows, err := res.RowsAffected()
	if err == nil && rows == 0 {
		return ErrWorkflowNotFound
	}
	return nil
}
