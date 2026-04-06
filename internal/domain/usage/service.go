package usage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

var ErrQuotaStateNotFound = errors.New("quota state not found")

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) RecordEvent(ctx context.Context, input RecordEventInput) (*Event, error) {
	now := time.Now().UTC()
	event := &Event{
		ID:            uuid.NewV7().String(),
		WorkspaceID:   input.WorkspaceID,
		ActorID:       input.ActorID,
		ActorType:     input.ActorType,
		RunID:         input.RunID,
		ToolName:      input.ToolName,
		ModelName:     input.ModelName,
		InputUnits:    input.InputUnits,
		OutputUnits:   input.OutputUnits,
		EstimatedCost: input.EstimatedCost,
		LatencyMs:     input.LatencyMs,
		CreatedAt:     now,
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO usage_event (
			id, workspace_id, actor_id, actor_type, run_id, tool_name, model_name,
			input_units, output_units, estimated_cost, latency_ms, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		event.ID,
		event.WorkspaceID,
		event.ActorID,
		event.ActorType,
		event.RunID,
		event.ToolName,
		event.ModelName,
		event.InputUnits,
		event.OutputUnits,
		event.EstimatedCost,
		event.LatencyMs,
		event.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return event, nil
}

func (s *Service) ListEvents(ctx context.Context, workspaceID string, runID *string, limit int) ([]*Event, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, workspace_id, actor_id, actor_type, run_id, tool_name, model_name,
		       input_units, output_units, estimated_cost, latency_ms, created_at
		FROM usage_event
		WHERE workspace_id = ? AND (? IS NULL OR run_id = ?)
		ORDER BY created_at DESC
		LIMIT ?
	`, workspaceID, runID, runID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]*Event, 0)
	for rows.Next() {
		event, scanErr := scanEvent(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func (s *Service) CreatePolicy(ctx context.Context, input CreatePolicyInput) (*Policy, error) {
	now := time.Now().UTC()
	scopeType := input.ScopeType
	if scopeType == "" {
		scopeType = "workspace"
	}
	enforcementMode := input.EnforcementMode
	if enforcementMode == "" {
		enforcementMode = "soft"
	}

	policy := &Policy{
		ID:              uuid.NewV7().String(),
		WorkspaceID:     input.WorkspaceID,
		PolicyType:      input.PolicyType,
		ScopeType:       scopeType,
		ScopeID:         input.ScopeID,
		MetricName:      input.MetricName,
		LimitValue:      input.LimitValue,
		ResetPeriod:     input.ResetPeriod,
		EnforcementMode: enforcementMode,
		IsActive:        input.IsActive,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO quota_policy (
			id, workspace_id, policy_type, scope_type, scope_id, metric_name,
			limit_value, reset_period, enforcement_mode, is_active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		policy.ID,
		policy.WorkspaceID,
		policy.PolicyType,
		policy.ScopeType,
		policy.ScopeID,
		policy.MetricName,
		policy.LimitValue,
		policy.ResetPeriod,
		policy.EnforcementMode,
		boolToInt(policy.IsActive),
		policy.CreatedAt,
		policy.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return policy, nil
}

func (s *Service) UpsertState(ctx context.Context, input UpsertStateInput) (*State, error) {
	now := time.Now().UTC()
	state := &State{
		ID:            uuid.NewV7().String(),
		WorkspaceID:   input.WorkspaceID,
		QuotaPolicyID: input.QuotaPolicyID,
		CurrentValue:  input.CurrentValue,
		PeriodStart:   input.PeriodStart.UTC(),
		PeriodEnd:     input.PeriodEnd.UTC(),
		LastEventAt:   input.LastEventAt,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO quota_state (
			id, workspace_id, quota_policy_id, current_value, period_start,
			period_end, last_event_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(quota_policy_id, period_start, period_end) DO UPDATE SET
			current_value = excluded.current_value,
			last_event_at = excluded.last_event_at,
			updated_at = excluded.updated_at
	`,
		state.ID,
		state.WorkspaceID,
		state.QuotaPolicyID,
		state.CurrentValue,
		state.PeriodStart,
		state.PeriodEnd,
		state.LastEventAt,
		state.CreatedAt,
		state.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return s.GetState(ctx, input.WorkspaceID, input.QuotaPolicyID, input.PeriodStart.UTC(), input.PeriodEnd.UTC())
}

func (s *Service) GetState(ctx context.Context, workspaceID, quotaPolicyID string, periodStart, periodEnd time.Time) (*State, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, quota_policy_id, current_value, period_start,
		       period_end, last_event_at, created_at, updated_at
		FROM quota_state
		WHERE workspace_id = ? AND quota_policy_id = ? AND period_start = ? AND period_end = ?
	`, workspaceID, quotaPolicyID, periodStart.UTC(), periodEnd.UTC())

	state, err := scanState(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrQuotaStateNotFound
	}
	if err != nil {
		return nil, err
	}
	return state, nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

type eventScanner interface {
	Scan(dest ...any) error
}

func scanEvent(scan eventScanner) (*Event, error) {
	var (
		event     Event
		runID     sql.NullString
		toolName  sql.NullString
		modelName sql.NullString
		latencyMs sql.NullInt64
	)

	if err := scan.Scan(
		&event.ID,
		&event.WorkspaceID,
		&event.ActorID,
		&event.ActorType,
		&runID,
		&toolName,
		&modelName,
		&event.InputUnits,
		&event.OutputUnits,
		&event.EstimatedCost,
		&latencyMs,
		&event.CreatedAt,
	); err != nil {
		return nil, err
	}

	if runID.Valid {
		event.RunID = &runID.String
	}
	if toolName.Valid {
		event.ToolName = &toolName.String
	}
	if modelName.Valid {
		event.ModelName = &modelName.String
	}
	if latencyMs.Valid {
		event.LatencyMs = &latencyMs.Int64
	}

	return &event, nil
}

type stateScanner interface {
	Scan(dest ...any) error
}

func scanState(scan stateScanner) (*State, error) {
	var (
		state       State
		lastEventAt sql.NullTime
	)

	if err := scan.Scan(
		&state.ID,
		&state.WorkspaceID,
		&state.QuotaPolicyID,
		&state.CurrentValue,
		&state.PeriodStart,
		&state.PeriodEnd,
		&lastEventAt,
		&state.CreatedAt,
		&state.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if lastEventAt.Valid {
		state.LastEventAt = &lastEventAt.Time
	}
	return &state, nil
}
