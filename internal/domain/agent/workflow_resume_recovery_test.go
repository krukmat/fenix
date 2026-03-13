package agent

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	schedulerdomain "github.com/matiasleandrokruk/fenix/internal/domain/scheduler"
)

func TestWorkflowResumeRecoveryAfterRestartProcessesPendingJobOnce(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	mustExecDSLRunner(t, db, `INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
	VALUES ('agent_dsl_restart', 'ws_dsl', 'dsl support', 'dsl', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExecDSLRunner(t, db, `INSERT INTO workflow (id, workspace_id, agent_definition_id, name, dsl_source, version, status, created_at, updated_at)
	VALUES ('wf_dsl_restart', 'ws_dsl', 'agent_dsl_restart', 'restart_support_case', 'WORKFLOW restart_support_case
ON case.created
SET case.status = "resolved"
NOTIFY contact WITH "done"', 1, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)

	orch := NewOrchestratorWithRegistry(db, NewRunnerRegistry())
	run, err := orch.TriggerAgent(context.Background(), TriggerAgentInput{
		AgentID:        "agent_dsl_restart",
		WorkspaceID:    "ws_dsl",
		TriggerType:    TriggerTypeEvent,
		TriggerContext: json.RawMessage(`{"case":{"id":"case-1"}}`),
	})
	if err != nil {
		t.Fatalf("TriggerAgent() error = %v", err)
	}
	run, err = orch.UpdateAgentRunStatus(context.Background(), "ws_dsl", run.ID, StatusAccepted)
	if err != nil {
		t.Fatalf("UpdateAgentRunStatus() error = %v", err)
	}

	repo := schedulerdomain.NewRepository(db)
	scheduler := schedulerdomain.NewService(repo)
	encoded, err := schedulerdomain.EncodeWorkflowResumePayload(schedulerdomain.WorkflowResumePayload{
		WorkflowID:      "wf_dsl_restart",
		RunID:           run.ID,
		ResumeStepIndex: 1,
	})
	if err != nil {
		t.Fatalf("EncodeWorkflowResumePayload() error = %v", err)
	}
	if _, err := scheduler.Schedule(context.Background(), schedulerdomain.ScheduleJobInput{
		WorkspaceID: "ws_dsl",
		JobType:     schedulerdomain.JobTypeWorkflowResume,
		Payload:     encoded,
		ExecuteAt:   time.Now().UTC().Add(-1 * time.Minute),
		SourceID:    "wf_dsl_restart",
	}); err != nil {
		t.Fatalf("Schedule() error = %v", err)
	}

	runner := NewDSLRunner(db)
	resumeHandler := NewWorkflowResumeHandler(runner, &RunContext{
		Orchestrator: orch,
		ToolRegistry: setupDSLToolRegistry(t, db),
		DB:           db,
	})

	var handled int32
	workerAfterRestart := schedulerdomain.NewWorker(repo, func(ctx context.Context, job *schedulerdomain.ScheduledJob) error {
		atomic.AddInt32(&handled, 1)
		return resumeHandler.Handle(ctx, job)
	})

	if err := workerAfterRestart.RunCycle(context.Background()); err != nil {
		t.Fatalf("RunCycle(after restart) error = %v", err)
	}
	if got := atomic.LoadInt32(&handled); got != 1 {
		t.Fatalf("handled count = %d, want 1", got)
	}

	stored, err := orch.GetAgentRun(context.Background(), "ws_dsl", run.ID)
	if err != nil {
		t.Fatalf("GetAgentRun() error = %v", err)
	}
	if stored.Status != StatusSuccess {
		t.Fatalf("status = %s, want %s", stored.Status, StatusSuccess)
	}

	workerSecondRestart := schedulerdomain.NewWorker(repo, func(ctx context.Context, job *schedulerdomain.ScheduledJob) error {
		atomic.AddInt32(&handled, 1)
		return resumeHandler.Handle(ctx, job)
	})
	if err := workerSecondRestart.RunCycle(context.Background()); err != nil {
		t.Fatalf("RunCycle(second restart) error = %v", err)
	}
	if got := atomic.LoadInt32(&handled); got != 1 {
		t.Fatalf("handled count after second restart = %d, want still 1", got)
	}
}
