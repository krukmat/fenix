package agent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/audit"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite/sqlcgen"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

type PromptExperimentStatus string

const (
	PromptExperimentStatusDraft     PromptExperimentStatus = "draft"
	PromptExperimentStatusRunning   PromptExperimentStatus = "running"
	PromptExperimentStatusCompleted PromptExperimentStatus = "completed"
	PromptExperimentStatusCancelled PromptExperimentStatus = "cancelled"
)

var (
	ErrPromptExperimentInvalidSplit   = errors.New("prompt experiment traffic split must sum to 100")
	ErrPromptExperimentSameVersion    = errors.New("prompt experiment requires distinct versions")
	ErrPromptExperimentAgentMismatch  = errors.New("prompt experiment versions must belong to the same agent")
	ErrPromptExperimentAlreadyRunning = errors.New("prompt experiment already running for agent")
	ErrPromptExperimentNotFound       = errors.New("prompt experiment not found")
)

type PromptExperiment struct {
	ID                       string
	WorkspaceID              string
	AgentDefinitionID        string
	ControlPromptVersionID   string
	CandidatePromptVersionID string
	ControlTrafficPercent    int
	CandidateTrafficPercent  int
	Status                   PromptExperimentStatus
	WinnerPromptVersionID    *string
	CreatedBy                *string
	StartedAt                *time.Time
	CompletedAt              *time.Time
	CreatedAt                time.Time
}

type StartPromptExperimentInput struct {
	WorkspaceID              string
	ControlPromptVersionID   string
	CandidatePromptVersionID string
	ControlTrafficPercent    int
	CandidateTrafficPercent  int
	CreatedBy                *string
}

type StopPromptExperimentInput struct {
	WorkspaceID           string
	ExperimentID          string
	WinnerPromptVersionID *string
}

func (s *PromptService) StartPromptExperiment(ctx context.Context, input StartPromptExperimentInput) (*PromptExperiment, error) {
	if err := validatePromptExperimentSplit(input.ControlTrafficPercent, input.CandidateTrafficPercent); err != nil {
		return nil, err
	}
	if input.ControlPromptVersionID == input.CandidatePromptVersionID {
		return nil, ErrPromptExperimentSameVersion
	}

	queries := sqlcgen.New(s.db)
	control, err := s.getPromptVersionRow(ctx, queries, input.WorkspaceID, input.ControlPromptVersionID)
	if err != nil {
		return nil, err
	}
	candidate, err := s.getPromptVersionRow(ctx, queries, input.WorkspaceID, input.CandidatePromptVersionID)
	if err != nil {
		return nil, err
	}
	if control.AgentDefinitionID != candidate.AgentDefinitionID {
		return nil, ErrPromptExperimentAgentMismatch
	}
	if err := s.ensureNoRunningExperiment(ctx, input.WorkspaceID, control.AgentDefinitionID); err != nil {
		return nil, err
	}

	experimentID := uuid.NewV7().String()
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO prompt_experiment (
			id, workspace_id, agent_definition_id, control_prompt_version_id, candidate_prompt_version_id,
			control_traffic_percent, candidate_traffic_percent, status, created_by, started_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		RETURNING id, workspace_id, agent_definition_id, control_prompt_version_id, candidate_prompt_version_id,
		          control_traffic_percent, candidate_traffic_percent, status, winner_prompt_version_id,
		          created_by, started_at, completed_at, created_at
	`,
		experimentID,
		input.WorkspaceID,
		control.AgentDefinitionID,
		input.ControlPromptVersionID,
		input.CandidatePromptVersionID,
		input.ControlTrafficPercent,
		input.CandidateTrafficPercent,
		PromptExperimentStatusRunning,
		input.CreatedBy,
	)

	experiment, err := scanPromptExperiment(row)
	if err != nil {
		return nil, fmt.Errorf("create prompt experiment: %w", err)
	}
	if PromptStatus(candidate.Status) == PromptStatusDraft {
		if err := s.setPromptStatus(ctx, input.WorkspaceID, input.CandidatePromptVersionID, PromptStatusTesting); err != nil {
			return nil, err
		}
	}
	s.logPromptExperimentAudit(ctx, input.WorkspaceID, "prompt.experiment_started", experiment.ID, control.AgentDefinitionID, map[string]interface{}{
		"control_prompt_version_id":   experiment.ControlPromptVersionID,
		"candidate_prompt_version_id": experiment.CandidatePromptVersionID,
		"control_traffic_percent":     experiment.ControlTrafficPercent,
		"candidate_traffic_percent":   experiment.CandidateTrafficPercent,
	})
	return experiment, nil
}

func (s *PromptService) ListPromptExperiments(ctx context.Context, workspaceID, agentID string) ([]*PromptExperiment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, workspace_id, agent_definition_id, control_prompt_version_id, candidate_prompt_version_id,
		       control_traffic_percent, candidate_traffic_percent, status, winner_prompt_version_id,
		       created_by, started_at, completed_at, created_at
		FROM prompt_experiment
		WHERE workspace_id = ? AND agent_definition_id = ?
		ORDER BY created_at DESC
	`, workspaceID, agentID)
	if err != nil {
		return nil, fmt.Errorf("list prompt experiments: %w", err)
	}
	defer rows.Close()

	var experiments []*PromptExperiment
	for rows.Next() {
		experiment, scanErr := scanPromptExperiment(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		experiments = append(experiments, experiment)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("list prompt experiments rows: %w", err)
	}
	return experiments, nil
}

func (s *PromptService) StopPromptExperiment(ctx context.Context, input StopPromptExperimentInput) (*PromptExperiment, error) {
	experiment, err := s.getPromptExperimentByID(ctx, input.WorkspaceID, input.ExperimentID)
	if err != nil {
		return nil, err
	}

	status := PromptExperimentStatusCancelled
	if input.WinnerPromptVersionID != nil {
		status = PromptExperimentStatusCompleted
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE prompt_experiment
		SET status = ?, winner_prompt_version_id = ?, completed_at = CURRENT_TIMESTAMP
		WHERE id = ? AND workspace_id = ?
	`, status, input.WinnerPromptVersionID, input.ExperimentID, input.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("stop prompt experiment: %w", err)
	}

	if shouldResetCandidateToDraft(experiment, input.WinnerPromptVersionID) {
		if err = s.setPromptStatus(ctx, input.WorkspaceID, experiment.CandidatePromptVersionID, PromptStatusDraft); err != nil {
			return nil, err
		}
	}

	updated, err := s.getPromptExperimentByID(ctx, input.WorkspaceID, input.ExperimentID)
	if err != nil {
		return nil, err
	}
	s.logPromptExperimentAudit(ctx, input.WorkspaceID, "prompt.experiment_stopped", updated.ID, updated.AgentDefinitionID, map[string]interface{}{
		"winner_prompt_version_id": updated.WinnerPromptVersionID,
		"status":                   updated.Status,
	})
	return updated, nil
}

func validatePromptExperimentSplit(controlPercent, candidatePercent int) error {
	if controlPercent+candidatePercent != 100 {
		return ErrPromptExperimentInvalidSplit
	}
	return nil
}

func (s *PromptService) ensureNoRunningExperiment(ctx context.Context, workspaceID, agentID string) error {
	row := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1)
		FROM prompt_experiment
		WHERE workspace_id = ? AND agent_definition_id = ? AND status = ?
	`, workspaceID, agentID, PromptExperimentStatusRunning)

	var count int
	if err := row.Scan(&count); err != nil {
		return fmt.Errorf("check running experiment: %w", err)
	}
	if count > 0 {
		return ErrPromptExperimentAlreadyRunning
	}
	return nil
}

func (s *PromptService) getPromptExperimentByID(ctx context.Context, workspaceID, experimentID string) (*PromptExperiment, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, agent_definition_id, control_prompt_version_id, candidate_prompt_version_id,
		       control_traffic_percent, candidate_traffic_percent, status, winner_prompt_version_id,
		       created_by, started_at, completed_at, created_at
		FROM prompt_experiment
		WHERE id = ? AND workspace_id = ?
	`, experimentID, workspaceID)
	experiment, err := scanPromptExperiment(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPromptExperimentNotFound
		}
		return nil, err
	}
	return experiment, nil
}

func shouldResetCandidateToDraft(experiment *PromptExperiment, winnerPromptVersionID *string) bool {
	if winnerPromptVersionID == nil {
		return true
	}
	return *winnerPromptVersionID != experiment.CandidatePromptVersionID
}

func (s *PromptService) setPromptStatus(ctx context.Context, workspaceID, promptVersionID string, status PromptStatus) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE prompt_version
		SET status = ?
		WHERE id = ? AND workspace_id = ?
	`, status, promptVersionID, workspaceID)
	if err != nil {
		return fmt.Errorf("set prompt status: %w", err)
	}
	return nil
}

func (s *PromptService) logPromptExperimentAudit(
	ctx context.Context,
	workspaceID, action, experimentID, agentID string,
	metadata map[string]interface{},
) {
	if s.audit == nil {
		return
	}
	entityType := "prompt_experiment"
	_ = s.audit.LogWithDetails(ctx, workspaceID, systemActorID, audit.ActorTypeSystem, action, &entityType, &experimentID, &audit.EventDetails{
		Metadata: mergePromptExperimentMetadata(metadata, agentID),
	}, audit.OutcomeSuccess)
}

func mergePromptExperimentMetadata(metadata map[string]interface{}, agentID string) map[string]interface{} {
	merged := map[string]interface{}{"agent_id": agentID}
	for key, value := range metadata {
		merged[key] = value
	}
	return merged
}

type promptExperimentRowScanner interface {
	Scan(dest ...any) error
}

func scanPromptExperiment(scanner promptExperimentRowScanner) (*PromptExperiment, error) {
	var experiment PromptExperiment
	var winnerPromptVersionID sql.NullString
	var createdBy sql.NullString
	var startedAt sql.NullTime
	var completedAt sql.NullTime

	err := scanner.Scan(
		&experiment.ID,
		&experiment.WorkspaceID,
		&experiment.AgentDefinitionID,
		&experiment.ControlPromptVersionID,
		&experiment.CandidatePromptVersionID,
		&experiment.ControlTrafficPercent,
		&experiment.CandidateTrafficPercent,
		&experiment.Status,
		&winnerPromptVersionID,
		&createdBy,
		&startedAt,
		&completedAt,
		&experiment.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if winnerPromptVersionID.Valid {
		experiment.WinnerPromptVersionID = &winnerPromptVersionID.String
	}
	if createdBy.Valid {
		experiment.CreatedBy = &createdBy.String
	}
	if startedAt.Valid {
		experiment.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		experiment.CompletedAt = &completedAt.Time
	}
	return &experiment, nil
}
