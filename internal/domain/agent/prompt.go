// Task 3.9: Prompt Versioning
package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/audit"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite/sqlcgen"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

// PromptVersion representa una versión del prompt de un agente
type PromptVersion struct {
	ID                 string
	WorkspaceID        string
	AgentDefinitionID  string
	VersionNumber      int
	SystemPrompt       string
	UserPromptTemplate *string
	Config             PromptConfig
	Status             PromptStatus
	CreatedBy          *string
	CreatedAt          time.Time
}

// PromptConfig almacena configuración del LLM
type PromptConfig struct {
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
}

// PromptStatus representa el estado de una versión
type PromptStatus string

const (
	PromptStatusDraft    PromptStatus = "draft"
	PromptStatusTesting  PromptStatus = "testing"
	PromptStatusActive   PromptStatus = "active"
	PromptStatusArchived PromptStatus = "archived"
)

// CreatePromptVersionInput es la entrada para crear una versión
type CreatePromptVersionInput struct {
	WorkspaceID        string
	AgentDefinitionID  string
	SystemPrompt       string
	UserPromptTemplate *string
	Config             string // JSON string
	CreatedBy          *string
}

// PromptService gestiona versiones de prompts
type PromptService struct {
	db    *sql.DB
	audit *audit.AuditService
}

// NewPromptService crea un nuevo PromptService
func NewPromptService(db *sql.DB, audit *audit.AuditService) *PromptService {
	return &PromptService{db: db, audit: audit}
}

// CreatePromptVersion crea una nueva versión de prompt (status=draft)
// Auto-incrementa version_number
func (s *PromptService) CreatePromptVersion(ctx context.Context, input CreatePromptVersionInput) (*PromptVersion, error) {
	workspaceID := input.WorkspaceID
	userID := getUserID(ctx, input.CreatedBy)
	queries := sqlcgen.New(s.db)

	nextVersion, err := s.getNextVersionNumber(ctx, queries, workspaceID, input.AgentDefinitionID)
	if err != nil {
		return nil, err
	}

	id := uuid.NewV7().String()
	row, err := queries.CreatePromptVersion(ctx, sqlcgen.CreatePromptVersionParams{
		ID:                 id,
		WorkspaceID:        workspaceID,
		AgentDefinitionID:  input.AgentDefinitionID,
		VersionNumber:      nextVersion,
		SystemPrompt:       input.SystemPrompt,
		UserPromptTemplate: input.UserPromptTemplate,
		Config:             input.Config,
		CreatedBy:          &userID,
	})
	if err != nil {
		return nil, fmt.Errorf("create: %w", err)
	}

	s.logPromptAudit(ctx, workspaceID, userID, "prompt.created", row.ID, map[string]interface{}{
		"version_number":      row.VersionNumber,
		"agent_definition_id": row.AgentDefinitionID,
	})

	return rowToPromptVersion(&row), nil
}

// GetActivePrompt retorna la versión activa de un agente
// Error si no existe versión activa
func (s *PromptService) GetActivePrompt(ctx context.Context, workspaceID, agentID string) (*PromptVersion, error) {
	queries := sqlcgen.New(s.db)

	row, err := queries.GetActivePrompt(ctx, sqlcgen.GetActivePromptParams{
		AgentDefinitionID: agentID,
		WorkspaceID:       workspaceID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no active prompt for agent %s", agentID)
		}
		return nil, fmt.Errorf("get active: %w", err)
	}

	return rowToPromptVersion(&row), nil
}

// ListPromptVersions lista todas las versiones del agente (descendente por version_number)
func (s *PromptService) ListPromptVersions(ctx context.Context, workspaceID, agentID string) ([]*PromptVersion, error) {
	queries := sqlcgen.New(s.db)

	rows, err := queries.ListPromptVersionsByAgent(ctx, sqlcgen.ListPromptVersionsByAgentParams{
		AgentDefinitionID: agentID,
		WorkspaceID:       workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("list: %w", err)
	}

	var results []*PromptVersion
	for _, row := range rows {
		results = append(results, rowToPromptVersion(&row))
	}
	return results, nil
}

// GetPromptVersionByID obtiene una versión específica
func (s *PromptService) GetPromptVersionByID(ctx context.Context, workspaceID, promptVersionID string) (*PromptVersion, error) {
	queries := sqlcgen.New(s.db)

	row, err := queries.GetPromptVersionByID(ctx, sqlcgen.GetPromptVersionByIDParams{
		ID:          promptVersionID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("prompt version not found")
		}
		return nil, fmt.Errorf("get by id: %w", err)
	}

	return rowToPromptVersion(&row), nil
}

// PromotePrompt activa una versión (status=active), archiva la anterior activa
func (s *PromptService) PromotePrompt(ctx context.Context, workspaceID, promptVersionID string) error {
	queries := sqlcgen.New(s.db)
	pv, err := queries.GetPromptVersionByID(ctx, sqlcgen.GetPromptVersionByIDParams{
		ID:          promptVersionID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return fmt.Errorf("get version: %w", err)
	}
	if pv.Status == string(PromptStatusArchived) {
		return fmt.Errorf("cannot promote archived prompt")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	qtx := queries.WithTx(tx)
	err = qtx.ArchivePreviousActivePrompts(ctx, sqlcgen.ArchivePreviousActivePromptsParams{
		AgentDefinitionID: pv.AgentDefinitionID,
		WorkspaceID:       workspaceID,
		ID:                promptVersionID,
	})
	if err != nil {
		return fmt.Errorf("archive previous: %w", err)
	}
	err = qtx.SetPromptStatus(ctx, sqlcgen.SetPromptStatusParams{
		Status:      string(PromptStatusActive),
		ID:          promptVersionID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return fmt.Errorf("set status: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	s.logSystemAudit(ctx, workspaceID, promptVersionID, pv.AgentDefinitionID)
	return nil
}

// RollbackPrompt reactiva la versión archivada más reciente del agente
func (s *PromptService) RollbackPrompt(ctx context.Context, workspaceID, agentID string) error {
	queries := sqlcgen.New(s.db)
	prev, err := queries.GetPreviousArchivedPrompt(ctx, sqlcgen.GetPreviousArchivedPromptParams{
		AgentDefinitionID: agentID,
		WorkspaceID:       workspaceID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no archived prompt to rollback to")
		}
		return fmt.Errorf("get previous: %w", err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	qtx := queries.WithTx(tx)
	err = qtx.ArchivePreviousActivePrompts(ctx, sqlcgen.ArchivePreviousActivePromptsParams{
		AgentDefinitionID: agentID,
		WorkspaceID:       workspaceID,
		ID:                prev.ID,
	})
	if err != nil {
		return fmt.Errorf("archive current: %w", err)
	}
	err = qtx.SetPromptStatus(ctx, sqlcgen.SetPromptStatusParams{
		Status:      string(PromptStatusActive),
		ID:          prev.ID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return fmt.Errorf("set status: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	s.logSystemAudit(ctx, workspaceID, prev.ID, agentID)
	return nil
}

// Helper functions to reduce complexity

func (s *PromptService) logSystemAudit(ctx context.Context, workspaceID, promptVersionID, agentID string) {
	if s.audit != nil {
		systemActor := "system"
		entityType := "prompt_version"
		_ = s.audit.LogWithDetails(ctx, workspaceID, systemActor, audit.ActorTypeSystem, "prompt.activated", &entityType, &promptVersionID, &audit.EventDetails{
			Metadata: map[string]interface{}{"agent_id": agentID},
		}, audit.OutcomeSuccess)
	}
}

func getUserID(ctx context.Context, createdBy *string) string {
	userID, _ := ctx.Value("user_id").(string)
	if userID == "" && createdBy != nil {
		userID = *createdBy
	}
	return userID
}

func (s *PromptService) getNextVersionNumber(ctx context.Context, queries sqlcgen.Querier, workspaceID, agentID string) (int64, error) {
	maxRow, err := queries.GetLatestPromptVersionNumber(ctx, sqlcgen.GetLatestPromptVersionNumberParams{
		AgentDefinitionID: agentID,
		WorkspaceID:       workspaceID,
	})
	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("get latest version: %w", err)
	}

	nextVersionNum := int64(0)
	if maxRow != nil {
		if v, ok := maxRow.(float64); ok {
			nextVersionNum = int64(v)
		} else if v2, ok2 := maxRow.(int64); ok2 {
			nextVersionNum = v2
		}
	}
	return nextVersionNum + 1, nil
}

func (s *PromptService) logPromptAudit(ctx context.Context, workspaceID, userID, action, resourceID string, metadata map[string]interface{}) {
	if s.audit != nil {
		entityType := "prompt_version"
		_ = s.audit.LogWithDetails(ctx, workspaceID, userID, audit.ActorTypeUser, action, &entityType, &resourceID, &audit.EventDetails{
			Metadata: metadata,
		}, audit.OutcomeSuccess)
	}
}

// Helper: rowToPromptVersion convierte una fila SQLC a PromptVersion
func rowToPromptVersion(row *sqlcgen.PromptVersion) *PromptVersion {
	var config PromptConfig
	if row.Config != "" {
		_ = json.Unmarshal([]byte(row.Config), &config)
	}

	return &PromptVersion{
		ID:                 row.ID,
		WorkspaceID:        row.WorkspaceID,
		AgentDefinitionID:  row.AgentDefinitionID,
		VersionNumber:      int(row.VersionNumber),
		SystemPrompt:       row.SystemPrompt,
		UserPromptTemplate: row.UserPromptTemplate,
		Config:             config,
		Status:             PromptStatus(row.Status),
		CreatedBy:          row.CreatedBy,
		CreatedAt:          row.CreatedAt,
	}
}
