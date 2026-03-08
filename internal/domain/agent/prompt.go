// Task 3.9: Prompt Versioning
package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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

const (
	evalStatusPassed           = "passed"
	systemActorID              = "system"
	entityTypePrompt           = "prompt_version"
	actionPromptActivated      = "prompt.activated"
	actionPromptRollback       = "prompt.rollback"
	actionPromptPromoteBlocked = "prompt.promote_blocked"
)

var (
	ErrPromptVersionNotFound      = errors.New("prompt version not found")
	ErrPromptVersionArchived      = errors.New("cannot promote archived prompt")
	ErrPromptRollbackInvalid      = errors.New("rollback requires an archived prompt version")
	ErrPromptPromotionEvalMissing = errors.New("prompt promotion requires a passing eval")
	ErrPromptPromotionEvalFailed  = errors.New("prompt promotion blocked by failed eval")
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
		if errors.Is(err, sql.ErrNoRows) {
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
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPromptVersionNotFound
		}
		return nil, fmt.Errorf("get by id: %w", err)
	}

	return rowToPromptVersion(&row), nil
}

// PromotePrompt activa una versión (status=active), archiva la anterior activa
func (s *PromptService) PromotePrompt(ctx context.Context, workspaceID, promptVersionID string) error {
	pv, err := s.preparePromptPromotion(ctx, workspaceID, promptVersionID)
	if err != nil {
		return err
	}
	if err = s.activatePromptVersion(ctx, workspaceID, pv.AgentDefinitionID, promptVersionID); err != nil {
		return err
	}
	s.logPromptActivation(ctx, workspaceID, promptVersionID, pv.AgentDefinitionID)
	return nil
}

// RollbackPrompt reactiva la versión archivada más reciente del agente
func (s *PromptService) RollbackPrompt(ctx context.Context, workspaceID, promptVersionID string) error {
	pv, err := s.preparePromptRollback(ctx, workspaceID, promptVersionID)
	if err != nil {
		return err
	}
	if err = s.activatePromptVersion(ctx, workspaceID, pv.AgentDefinitionID, promptVersionID); err != nil {
		return err
	}
	s.logPromptRollback(ctx, workspaceID, promptVersionID, pv.AgentDefinitionID)
	return nil
}

// Helper functions to reduce complexity

func (s *PromptService) getPromptVersionRow(
	ctx context.Context,
	queries sqlcgen.Querier,
	workspaceID, promptVersionID string,
) (*sqlcgen.PromptVersion, error) {
	pv, err := queries.GetPromptVersionByID(ctx, sqlcgen.GetPromptVersionByIDParams{
		ID:          promptVersionID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPromptVersionNotFound
		}
		return nil, fmt.Errorf("get version: %w", err)
	}
	return &pv, nil
}

func validatePromotionStatus(status string) error {
	if status == string(PromptStatusArchived) {
		return ErrPromptVersionArchived
	}
	return nil
}

func (s *PromptService) requirePassingEval(ctx context.Context, workspaceID, promptVersionID string) error {
	row := s.db.QueryRowContext(ctx, `
		SELECT status
		FROM eval_run
		WHERE workspace_id = ? AND prompt_version_id = ?
		ORDER BY COALESCE(completed_at, created_at) DESC, created_at DESC
		LIMIT 1
	`, workspaceID, promptVersionID)

	var status string
	err := row.Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrPromptPromotionEvalMissing
		}
		return fmt.Errorf("get latest eval result: %w", err)
	}
	if status != evalStatusPassed {
		return ErrPromptPromotionEvalFailed
	}
	return nil
}

func (s *PromptService) preparePromptPromotion(ctx context.Context, workspaceID, promptVersionID string) (*sqlcgen.PromptVersion, error) {
	queries := sqlcgen.New(s.db)
	pv, err := s.getPromptVersionRow(ctx, queries, workspaceID, promptVersionID)
	if err != nil {
		return nil, err
	}
	if err = validatePromotionStatus(pv.Status); err != nil {
		return nil, err
	}
	if err = s.requirePassingEval(ctx, workspaceID, promptVersionID); err != nil {
		s.logPromptBlockedAudit(ctx, workspaceID, promptVersionID, pv.AgentDefinitionID, err)
		return nil, err
	}
	return pv, nil
}

func (s *PromptService) preparePromptRollback(ctx context.Context, workspaceID, promptVersionID string) (*sqlcgen.PromptVersion, error) {
	queries := sqlcgen.New(s.db)
	pv, err := s.getPromptVersionRow(ctx, queries, workspaceID, promptVersionID)
	if err != nil {
		return nil, err
	}
	if PromptStatus(pv.Status) != PromptStatusArchived {
		return nil, ErrPromptRollbackInvalid
	}
	return pv, nil
}

func (s *PromptService) beginPromptTx(ctx context.Context) (*sql.Tx, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	return tx, nil
}

func (s *PromptService) activatePromptVersion(
	ctx context.Context,
	workspaceID, agentDefinitionID, promptVersionID string,
) error {
	tx, err := s.beginPromptTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	queries := sqlcgen.New(s.db)
	qtx := queries.WithTx(tx)
	if err = archiveOtherActivePrompts(ctx, qtx, agentDefinitionID, workspaceID, promptVersionID); err != nil {
		return err
	}
	if err = setPromptVersionStatus(ctx, qtx, promptVersionID, workspaceID, PromptStatusActive); err != nil {
		return err
	}
	if err = s.syncActivePromptVersion(ctx, tx, workspaceID, agentDefinitionID, promptVersionID); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func archiveOtherActivePrompts(
	ctx context.Context,
	qtx *sqlcgen.Queries,
	agentDefinitionID, workspaceID, promptVersionID string,
) error {
	err := qtx.ArchivePreviousActivePrompts(ctx, sqlcgen.ArchivePreviousActivePromptsParams{
		AgentDefinitionID: agentDefinitionID,
		WorkspaceID:       workspaceID,
		ID:                promptVersionID,
	})
	if err != nil {
		return fmt.Errorf("archive previous: %w", err)
	}
	return nil
}

func setPromptVersionStatus(
	ctx context.Context,
	qtx *sqlcgen.Queries,
	promptVersionID, workspaceID string,
	status PromptStatus,
) error {
	err := qtx.SetPromptStatus(ctx, sqlcgen.SetPromptStatusParams{
		Status:      string(status),
		ID:          promptVersionID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return fmt.Errorf("set status: %w", err)
	}
	return nil
}

func (s *PromptService) syncActivePromptVersion(
	ctx context.Context,
	tx *sql.Tx,
	workspaceID, agentDefinitionID, promptVersionID string,
) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE agent_definition
		SET active_prompt_version_id = ?, updated_at = datetime('now')
		WHERE id = ? AND workspace_id = ?
	`, promptVersionID, agentDefinitionID, workspaceID)
	if err != nil {
		return fmt.Errorf("sync active prompt version: %w", err)
	}
	return nil
}

func (s *PromptService) logPromptActivation(ctx context.Context, workspaceID, promptVersionID, agentID string) {
	if s.audit != nil {
		_ = s.audit.LogWithDetails(ctx, workspaceID, systemActorID, audit.ActorTypeSystem, actionPromptActivated, stringPtr(entityTypePrompt), &promptVersionID, &audit.EventDetails{
			Metadata: map[string]interface{}{"agent_id": agentID},
		}, audit.OutcomeSuccess)
	}
}

func (s *PromptService) logPromptRollback(ctx context.Context, workspaceID, promptVersionID, agentID string) {
	if s.audit != nil {
		_ = s.audit.LogWithDetails(ctx, workspaceID, systemActorID, audit.ActorTypeSystem, actionPromptRollback, stringPtr(entityTypePrompt), &promptVersionID, &audit.EventDetails{
			Metadata: map[string]interface{}{"agent_id": agentID},
		}, audit.OutcomeSuccess)
	}
}

func (s *PromptService) logPromptBlockedAudit(ctx context.Context, workspaceID, promptVersionID, agentID string, reason error) {
	if s.audit != nil {
		_ = s.audit.LogWithDetails(ctx, workspaceID, systemActorID, audit.ActorTypeSystem, actionPromptPromoteBlocked, stringPtr(entityTypePrompt), &promptVersionID, &audit.EventDetails{
			Metadata: map[string]interface{}{
				"agent_id": agentID,
				"reason":   reason.Error(),
			},
		}, audit.OutcomeDenied)
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
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
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
		_ = s.audit.LogWithDetails(ctx, workspaceID, userID, audit.ActorTypeUser, action, stringPtr(entityTypePrompt), &resourceID, &audit.EventDetails{
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
