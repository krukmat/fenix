package blackboard

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"
)

const deferralReasonNoHypotheses = "no_hypotheses"

var (
	ErrPipelineAlreadyRunning     = errors.New("blackboard pipeline already running")
	ErrCognitiveWorkspaceNotFound = errors.New("cognitive workspace not found")
)

type pipelinePlanner interface {
	BuildWorkspacePlan(ctx context.Context, cognitiveWorkspaceID string, config PlanningConfig) (*CollaborativePlanningResult, error)
}

type pipelineExecutor interface {
	Execute(ctx context.Context, workspace CognitiveWorkspace, plan *CollaborativePlanningResult) (*ExecutionOutcome, error)
}

type Orchestrator struct {
	db       *sql.DB
	arb      Arbitrator
	planner  pipelinePlanner
	executor pipelineExecutor
	now      func() time.Time
	inflight sync.Map
}

func NewBlackboardOrchestrator(db *sql.DB, arb Arbitrator, planner Planner, executor *PlannerExecutor) *Orchestrator {
	return &Orchestrator{
		db:       db,
		arb:      arb,
		planner:  planner,
		executor: executor,
		now:      func() time.Time { return time.Now().UTC() },
	}
}

func (o *Orchestrator) RunPipeline(ctx context.Context, cognitiveWorkspaceID string) (*ExecutionOutcome, error) {
	if _, busy := o.inflight.LoadOrStore(cognitiveWorkspaceID, struct{}{}); busy {
		return nil, ErrPipelineAlreadyRunning
	}
	defer o.inflight.Delete(cognitiveWorkspaceID)

	workspace, err := o.loadWorkspace(ctx, cognitiveWorkspaceID)
	if err != nil {
		return nil, err
	}

	arbitration, err := o.arb.RankWorkspace(ctx, cognitiveWorkspaceID, ArbitrationConfig{
		Now:           o.currentTime(),
		PersistResult: true,
	})
	if err != nil {
		return nil, err
	}
	if len(arbitration.Ranked) == 0 {
		return &ExecutionOutcome{
			CognitiveWorkspaceID: workspace.ID,
			WorkspaceID:          workspace.WorkspaceID,
			Executed:             []ExecutedStep{},
			DeferralReason:       deferralReasonNoHypotheses,
		}, nil
	}

	plan, err := o.planner.BuildWorkspacePlan(ctx, cognitiveWorkspaceID, PlanningConfig{
		Now:           o.currentTime(),
		PersistResult: true,
	})
	if err != nil {
		return nil, err
	}
	return o.executor.Execute(ctx, *workspace, plan)
}

func (o *Orchestrator) loadWorkspace(ctx context.Context, cognitiveWorkspaceID string) (*CognitiveWorkspace, error) {
	row := o.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, status, created_at
		FROM cognitive_workspace
		WHERE id = ?
	`, cognitiveWorkspaceID)

	var workspace CognitiveWorkspace
	var status string
	if err := row.Scan(&workspace.ID, &workspace.WorkspaceID, &status, &workspace.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCognitiveWorkspaceNotFound
		}
		return nil, fmt.Errorf("load cognitive workspace: %w", err)
	}
	workspace.Status = WorkspaceStatus(status)
	return &workspace, nil
}

func (o *Orchestrator) currentTime() time.Time {
	if o != nil && o.now != nil {
		return o.now().UTC()
	}
	return time.Now().UTC()
}
