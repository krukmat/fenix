package agent

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	schedulerdomain "github.com/matiasleandrokruk/fenix/internal/domain/scheduler"
)

func TestWorkflowResumeHandlerHandleResumesFromStepIndex(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	mustExecDSLRunner(t, db, `INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
	VALUES ('agent_dsl_resume', 'ws_dsl', 'dsl support', 'dsl', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExecDSLRunner(t, db, `INSERT INTO workflow (id, workspace_id, agent_definition_id, name, dsl_source, version, status, created_at, updated_at)
	VALUES ('wf_dsl_resume', 'ws_dsl', 'agent_dsl_resume', 'resume_support_case', 'WORKFLOW resume_support_case
ON case.created
SET case.status = "resolved"
NOTIFY contact WITH "done"', 1, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)

	orch := NewOrchestratorWithRegistry(db, NewRunnerRegistry())
	run, err := orch.TriggerAgent(context.Background(), TriggerAgentInput{
		AgentID:        "agent_dsl_resume",
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

	runner := NewDSLRunner(db)
	handler := NewWorkflowResumeHandler(runner, &RunContext{
		Orchestrator: orch,
		ToolRegistry: setupDSLToolRegistry(t, db),
		DB:           db,
	})

	payload, _ := json.Marshal(schedulerdomain.WorkflowResumePayload{
		WorkflowID:      "wf_dsl_resume",
		RunID:           run.ID,
		ResumeStepIndex: 1,
	})
	err = handler.Handle(context.Background(), &schedulerdomain.ScheduledJob{
		ID:          "job-resume-1",
		WorkspaceID: "ws_dsl",
		JobType:     schedulerdomain.JobTypeWorkflowResume,
		Payload:     payload,
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	stored, err := orch.GetAgentRun(context.Background(), "ws_dsl", run.ID)
	if err != nil {
		t.Fatalf("GetAgentRun() error = %v", err)
	}
	if stored.Status != StatusSuccess {
		t.Fatalf("status = %s, want %s", stored.Status, StatusSuccess)
	}

	var output map[string]any
	if err := json.Unmarshal(stored.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if resumed, _ := output["resumed"].(bool); !resumed {
		t.Fatalf("expected resumed=true, got %#v", output)
	}
	statements, ok := output["statements"].([]any)
	if !ok || len(statements) != 1 {
		t.Fatalf("expected one resumed statement, got %#v", output["statements"])
	}

	var toolCalls []ToolCall
	if err := json.Unmarshal(stored.ToolCalls, &toolCalls); err != nil {
		t.Fatalf("unmarshal tool calls: %v", err)
	}
	if len(toolCalls) != 1 {
		t.Fatalf("len(toolCalls) = %d, want 1", len(toolCalls))
	}

	steps, err := orch.ListRunSteps(context.Background(), "ws_dsl", run.ID)
	if err != nil {
		t.Fatalf("ListRunSteps() error = %v", err)
	}
	dslSteps := filterRunStepsByType(steps, StepTypeDSLStatement)
	if len(dslSteps) != 1 {
		t.Fatalf("len(dslSteps) = %d, want 1", len(dslSteps))
	}
	if dslSteps[0].Status != StepStatusSuccess {
		t.Fatalf("dsl step status = %s, want %s", dslSteps[0].Status, StepStatusSuccess)
	}
}

func TestWorkflowResumeHandlerHandleBlocksArchivedWorkflow(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	mustExecDSLRunner(t, db, `INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
	VALUES ('agent_dsl_resume_archived', 'ws_dsl', 'dsl support', 'dsl', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExecDSLRunner(t, db, `INSERT INTO workflow (id, workspace_id, agent_definition_id, name, dsl_source, version, status, archived_at, created_at, updated_at)
	VALUES ('wf_dsl_resume_archived', 'ws_dsl', 'agent_dsl_resume_archived', 'resume_support_case', 'WORKFLOW resume_support_case
ON case.created
NOTIFY contact WITH "done"', 1, 'archived', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)

	orch := NewOrchestratorWithRegistry(db, NewRunnerRegistry())
	run, err := orch.TriggerAgent(context.Background(), TriggerAgentInput{
		AgentID:        "agent_dsl_resume_archived",
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

	runner := NewDSLRunner(db)
	handler := NewWorkflowResumeHandler(runner, &RunContext{
		Orchestrator: orch,
		ToolRegistry: setupDSLToolRegistry(t, db),
		DB:           db,
	})

	payload, _ := json.Marshal(schedulerdomain.WorkflowResumePayload{
		WorkflowID:      "wf_dsl_resume_archived",
		RunID:           run.ID,
		ResumeStepIndex: 0,
	})
	err = handler.Handle(context.Background(), &schedulerdomain.ScheduledJob{
		ID:          "job-resume-archived",
		WorkspaceID: "ws_dsl",
		JobType:     schedulerdomain.JobTypeWorkflowResume,
		Payload:     payload,
	})
	if !errors.Is(err, ErrDSLWorkflowNotActive) {
		t.Fatalf("expected ErrDSLWorkflowNotActive, got %v", err)
	}

	stored, err := orch.GetAgentRun(context.Background(), "ws_dsl", run.ID)
	if err != nil {
		t.Fatalf("GetAgentRun() error = %v", err)
	}
	if stored.Status != StatusFailed {
		t.Fatalf("status = %s, want %s", stored.Status, StatusFailed)
	}
	var output map[string]any
	if err := json.Unmarshal(stored.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if output["error"] != ErrDSLWorkflowNotActive.Error() {
		t.Fatalf("expected archived workflow error in output, got %#v", output)
	}
}

func TestWorkflowResumeHandlerHandleRejectsInvalidJob(t *testing.T) {
	t.Parallel()

	handler := NewWorkflowResumeHandler(NewDSLRunner(setupDSLRunnerDB(t)), &RunContext{})

	err := handler.Handle(context.Background(), &schedulerdomain.ScheduledJob{
		ID:          "job-invalid",
		WorkspaceID: "",
		JobType:     schedulerdomain.JobTypeWorkflowResume,
		Payload:     json.RawMessage(`{}`),
	})
	if !errors.Is(err, ErrWorkflowResumeJobInvalid) {
		t.Fatalf("expected ErrWorkflowResumeJobInvalid, got %v", err)
	}
}

func TestWorkflowResumeHandlerHandleRejectsMissingRunner(t *testing.T) {
	t.Parallel()

	handler := NewWorkflowResumeHandler(nil, &RunContext{})
	err := handler.Handle(context.Background(), &schedulerdomain.ScheduledJob{
		ID:          "job-missing-runner",
		WorkspaceID: "ws_dsl",
		JobType:     schedulerdomain.JobTypeWorkflowResume,
		Payload:     json.RawMessage(`{"workflow_id":"wf-1","run_id":"run-1","resume_step_index":0}`),
	})
	if !errors.Is(err, ErrWorkflowResumeHandlerMissingRunner) {
		t.Fatalf("expected ErrWorkflowResumeHandlerMissingRunner, got %v", err)
	}
}

func TestWorkflowResumeHandlerHandleRejectsInvalidPayload(t *testing.T) {
	t.Parallel()

	handler := NewWorkflowResumeHandler(NewDSLRunner(setupDSLRunnerDB(t)), &RunContext{})
	err := handler.Handle(context.Background(), &schedulerdomain.ScheduledJob{
		ID:          "job-invalid-payload",
		WorkspaceID: "ws_dsl",
		JobType:     schedulerdomain.JobTypeWorkflowResume,
		Payload:     json.RawMessage(`{`),
	})
	if err == nil {
		t.Fatal("Handle() expected error")
	}
}
