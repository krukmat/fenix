package gobdd

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/cucumber/godog"
	agentdomain "github.com/matiasleandrokruk/fenix/internal/domain/agent"
	policydomain "github.com/matiasleandrokruk/fenix/internal/domain/policy"
	schedulerdomain "github.com/matiasleandrokruk/fenix/internal/domain/scheduler"
	tooldomain "github.com/matiasleandrokruk/fenix/internal/domain/tool"
	isqlite "github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	// Register the pure-Go SQLite driver for in-memory BDD runtime tests.
	_ "modernc.org/sqlite"
)

const (
	bddUserID                  = "user_bdd"
	bddAgentTypeDSL            = "dsl"
	bddEventPayloadCase        = `{"case":{"id":"case-1"}}`
	bddOneDSLStepErrFmt        = "len(dslSteps) = %d, want 1"
	bddStepStatusErrFmt        = "step status = %s, want %s"
	bddAgentA6Failure          = "agent_a6_failure"
	bddErrExpectedArchivedFlow = "expected ErrDSLWorkflowNotActive, got %w"
)

type workflowRuntimeState struct {
	db            *sql.DB
	orchestrator  *agentdomain.Orchestrator
	registry      *agentdomain.RunnerRegistry
	runner        *agentdomain.DSLRunner
	toolRegistry  *tooldomain.ToolRegistry
	approvalSvc   *policydomain.ApprovalService
	schedulerRepo *schedulerdomain.Repository
	schedulerSvc  *schedulerdomain.Service
	resumeHandler *agentdomain.WorkflowResumeHandler
	lastRun       *agentdomain.Run
	lastSteps     []*agentdomain.RunStep
	lastJobs      []*schedulerdomain.ScheduledJob
	lastErr       error
	activeAgentID string
}

type bddToolExecutor struct {
	result json.RawMessage
	err    error
}

type bddPendingNestedRunner struct{}

func (e bddToolExecutor) Execute(_ context.Context, _ json.RawMessage) (json.RawMessage, error) {
	if e.err != nil {
		return nil, e.err
	}
	return e.result, nil
}

func (bddPendingNestedRunner) Run(ctx context.Context, rc *agentdomain.RunContext, input agentdomain.TriggerAgentInput) (*agentdomain.Run, error) {
	run, err := rc.Orchestrator.TriggerAgent(ctx, input)
	if err != nil {
		return nil, err
	}
	output, _ := json.Marshal(map[string]any{
		"action":      "pending_approval",
		"approval_id": "apr_bdd_nested_1",
	})
	return rc.Orchestrator.UpdateAgentRun(ctx, input.WorkspaceID, run.ID, agentdomain.RunUpdates{
		Status:    agentdomain.StatusAccepted,
		Output:    output,
		ToolCalls: json.RawMessage(`[]`),
	})
}

func (s *workflowRuntimeState) close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func initWorkflowRuntimeScenarios(ctx *godog.ScenarioContext, state *scenarioState) {
	registerA4WorkflowExecutionSteps(ctx, state)
	registerA6DeferredActionSteps(ctx, state)
}

func registerA4WorkflowExecutionSteps(ctx *godog.ScenarioContext, state *scenarioState) {
	ctx.Step(`^an active workflow matches an incoming event$`, func() error {
		runtimeState, err := setupWorkflowRuntimeBDD("")
		if err != nil {
			return err
		}
		seedErr := runtimeState.seedActiveWorkflow("agent_a4_exec", "wf_a4_exec", "resolve_support_case", "WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"resolved\"\nNOTIFY contact WITH \"done\"")
		if seedErr != nil {
			return seedErr
		}
		state.workflowRuntime = runtimeState
		return nil
	})
	ctx.Step(`^the workflow runtime has registered tools available$`, func() error {
		if state.workflowRuntime == nil || state.workflowRuntime.toolRegistry == nil {
			return godog.ErrPending
		}
		return nil
	})
	ctx.Step(`^the workflow runtime executes the matching workflow$`, func() error {
		if state.workflowRuntime == nil || state.workflowRuntime.activeAgentID == "" {
			return godog.ErrPending
		}
		return state.workflowRuntime.runWorkflow(state.workflowRuntime.activeAgentID, agentdomain.TriggerTypeEvent, json.RawMessage(bddEventPayloadCase))
	})
	ctx.Step(`^the workflow run completes successfully$`, func() error {
		if state.workflowRuntime == nil || state.workflowRuntime.lastRun == nil {
			return godog.ErrPending
		}
		if state.workflowRuntime.lastRun.Status != agentdomain.StatusSuccess {
			return fmt.Errorf("run status = %s, want %s", state.workflowRuntime.lastRun.Status, agentdomain.StatusSuccess)
		}
		return nil
	})
	ctx.Step(`^the workflow steps are recorded in the audit trail$`, func() error {
		if state.workflowRuntime == nil || len(state.workflowRuntime.lastSteps) == 0 {
			return godog.ErrPending
		}
		dslSteps := filterBDDRunStepsByType(state.workflowRuntime.lastSteps, agentdomain.StepTypeDSLStatement)
		if len(dslSteps) == 0 {
			return fmt.Errorf("expected traced workflow steps")
		}
		return nil
	})

	ctx.Step(`^an active workflow contains a conditional branch$`, func() error {
		runtimeState, err := setupWorkflowRuntimeBDD("")
		if err != nil {
			return err
		}
		seedErr := runtimeState.seedActiveWorkflow("agent_a4_conditional", "wf_a4_conditional", "conditional_case", "WORKFLOW conditional_case\nON case.created\nIF case.priority == \"high\":\n  NOTIFY contact WITH \"done\"")
		if seedErr != nil {
			return seedErr
		}
		state.workflowRuntime = runtimeState
		return nil
	})
	ctx.Step(`^the workflow runtime executes the workflow with a non-matching condition$`, func() error {
		if state.workflowRuntime == nil {
			return godog.ErrPending
		}
		return state.workflowRuntime.runWorkflow("agent_a4_conditional", agentdomain.TriggerTypeEvent, json.RawMessage(`{"case":{"id":"case-1","priority":"low"}}`))
	})
	ctx.Step(`^the conditional step is recorded as skipped$`, func() error {
		if state.workflowRuntime == nil || len(state.workflowRuntime.lastSteps) == 0 {
			return godog.ErrPending
		}
		dslSteps := filterBDDRunStepsByType(state.workflowRuntime.lastSteps, agentdomain.StepTypeDSLStatement)
		if len(dslSteps) != 1 {
			return fmt.Errorf(bddOneDSLStepErrFmt, len(dslSteps))
		}
		if dslSteps[0].Status != agentdomain.StepStatusSkipped {
			return fmt.Errorf(bddStepStatusErrFmt, dslSteps[0].Status, agentdomain.StepStatusSkipped)
		}
		return nil
	})
	ctx.Step(`^the workflow run still completes successfully$`, func() error {
		return expectBDDRunStatus(state.workflowRuntime, agentdomain.StatusSuccess)
	})

	ctx.Step(`^an active workflow maps a statement to a failing tool$`, func() error {
		runtimeState, err := setupWorkflowRuntimeBDD(tooldomain.BuiltinUpdateCase)
		if err != nil {
			return err
		}
		seedErr := runtimeState.seedActiveWorkflow("agent_a4_failure", "wf_a4_failure", "fail_case", "WORKFLOW fail_case\nON case.created\nSET case.status = \"resolved\"")
		if seedErr != nil {
			return seedErr
		}
		state.workflowRuntime = runtimeState
		return nil
	})
	ctx.Step(`^the workflow run fails$`, func() error {
		return expectBDDRunStatus(state.workflowRuntime, agentdomain.StatusFailed)
	})
	ctx.Step(`^the failing workflow step is recorded in the audit trail$`, func() error {
		if state.workflowRuntime == nil || len(state.workflowRuntime.lastSteps) == 0 {
			return godog.ErrPending
		}
		dslSteps := filterBDDRunStepsByType(state.workflowRuntime.lastSteps, agentdomain.StepTypeDSLStatement)
		if len(dslSteps) == 0 {
			return fmt.Errorf("expected traced workflow steps")
		}
		last := dslSteps[len(dslSteps)-1]
		if last.Status != agentdomain.StepStatusFailed {
			return fmt.Errorf("last step status = %s, want %s", last.Status, agentdomain.StepStatusFailed)
		}
		return nil
	})

	ctx.Step(`^an active workflow delegates a nested action that requires approval$`, func() error {
		runtimeState, err := setupWorkflowRuntimeBDD("")
		if err != nil {
			return err
		}
		seedAgentErr := runtimeState.seedAgentDefinition("nested_pending_target", "pending_agent", "nested_pending")
		if seedAgentErr != nil {
			return seedAgentErr
		}
		seedWorkflowErr := runtimeState.seedActiveWorkflow("agent_a4_approval", "wf_a4_approval", "approve_case", "WORKFLOW approve_case\nON case.created\nAGENT pending_agent WITH {\"case_id\":\"case-1\"}")
		if seedWorkflowErr != nil {
			return seedWorkflowErr
		}
		registerErr := runtimeState.registerNestedPendingRunner()
		if registerErr != nil {
			return registerErr
		}
		state.workflowRuntime = runtimeState
		return nil
	})
	ctx.Step(`^the workflow run remains pending approval$`, func() error {
		return expectBDDRunStatus(state.workflowRuntime, agentdomain.StatusAccepted)
	})
	ctx.Step(`^the pending approval is recorded in the runtime trace$`, func() error {
		if state.workflowRuntime == nil || len(state.workflowRuntime.lastSteps) == 0 {
			return godog.ErrPending
		}
		dslSteps := filterBDDRunStepsByType(state.workflowRuntime.lastSteps, agentdomain.StepTypeDSLStatement)
		if len(dslSteps) != 1 {
			return fmt.Errorf(bddOneDSLStepErrFmt, len(dslSteps))
		}
		if dslSteps[0].Status != agentdomain.StepStatusRunning {
			return fmt.Errorf(bddStepStatusErrFmt, dslSteps[0].Status, agentdomain.StepStatusRunning)
		}
		var traceOutput map[string]any
		if err := json.Unmarshal(dslSteps[0].Output, &traceOutput); err != nil {
			return err
		}
		output, ok := traceOutput["output"].(map[string]any)
		if !ok || output["action"] != "pending_approval" {
			return fmt.Errorf("expected pending_approval marker, got %#v", traceOutput)
		}
		return nil
	})
}

func registerA6DeferredActionSteps(ctx *godog.ScenarioContext, state *scenarioState) {
	ctx.Step(`^a workflow run reaches a wait step that must resume later$`, func() error {
		runtimeState, err := setupWorkflowRuntimeBDD("")
		if err != nil {
			return err
		}
		seedErr := runtimeState.seedActiveWorkflow("agent_a6_wait", "wf_a6_wait", "wait_case", "WORKFLOW wait_case\nON case.created\nWAIT 0\nNOTIFY contact WITH \"done\"")
		if seedErr != nil {
			return seedErr
		}
		state.workflowRuntime = runtimeState
		return nil
	})
	ctx.Step(`^the runtime schedules the deferred action$`, func() error {
		if state.workflowRuntime == nil {
			return godog.ErrPending
		}
		if err := state.workflowRuntime.runWorkflow("agent_a6_wait", agentdomain.TriggerTypeEvent, json.RawMessage(bddEventPayloadCase)); err != nil {
			return err
		}
		jobs, err := state.workflowRuntime.schedulerRepo.ListDue(context.Background(), time.Now().UTC().Add(time.Second), 10)
		if err != nil {
			return err
		}
		state.workflowRuntime.lastJobs = jobs
		return nil
	})
	ctx.Step(`^the deferred job is stored under scheduler control$`, func() error {
		if state.workflowRuntime == nil || len(state.workflowRuntime.lastJobs) == 0 {
			return godog.ErrPending
		}
		if len(state.workflowRuntime.lastJobs) != 1 {
			return fmt.Errorf("len(jobs) = %d, want 1", len(state.workflowRuntime.lastJobs))
		}
		return nil
	})
	ctx.Step(`^the workflow can resume from the deferred action state$`, func() error {
		if state.workflowRuntime == nil || len(state.workflowRuntime.lastJobs) == 0 {
			return godog.ErrPending
		}
		if err := state.workflowRuntime.resumeJob(state.workflowRuntime.lastJobs[0]); err != nil {
			return err
		}
		if err := expectBDDRunStatus(state.workflowRuntime, agentdomain.StatusSuccess); err != nil {
			return err
		}
		var output map[string]any
		if err := json.Unmarshal(state.workflowRuntime.lastRun.Output, &output); err != nil {
			return err
		}
		resumed, _ := output["resumed"].(bool)
		if !resumed {
			return fmt.Errorf("expected resumed=true, got %#v", output)
		}
		return nil
	})

	ctx.Step(`^a scheduled workflow resume targets an archived workflow$`, func() error {
		runtimeState, err := setupWorkflowRuntimeBDD("")
		if err != nil {
			return err
		}
		seedAgentErr := runtimeState.seedAgentDefinition("agent_a6_archived", "archived_case", bddAgentTypeDSL)
		if seedAgentErr != nil {
			return seedAgentErr
		}
		seedWorkflowErr := runtimeState.seedWorkflowRecord("wf_a6_archived", "agent_a6_archived", "archived_case", "WORKFLOW archived_case\nON case.created\nNOTIFY contact WITH \"done\"", "archived")
		if seedWorkflowErr != nil {
			return seedWorkflowErr
		}
		seedRunErr := runtimeState.seedAcceptedRun("agent_a6_archived", "wf_a6_archived", 0)
		if seedRunErr != nil {
			return seedRunErr
		}
		state.workflowRuntime = runtimeState
		return nil
	})
	ctx.Step(`^the resume handler processes the archived workflow job$`, func() error {
		if state.workflowRuntime == nil || len(state.workflowRuntime.lastJobs) == 0 {
			return godog.ErrPending
		}
		state.workflowRuntime.lastErr = state.workflowRuntime.resumeHandler.Handle(context.Background(), state.workflowRuntime.lastJobs[0])
		_ = state.workflowRuntime.refreshRunFromCurrent()
		return nil
	})
	ctx.Step(`^the deferred resume is rejected$`, func() error {
		if state.workflowRuntime == nil {
			return godog.ErrPending
		}
		if !errors.Is(state.workflowRuntime.lastErr, agentdomain.ErrDSLWorkflowNotActive) {
			return fmt.Errorf(bddErrExpectedArchivedFlow, state.workflowRuntime.lastErr)
		}
		return nil
	})
	ctx.Step(`^the workflow run records the archived workflow error$`, func() error {
		if err := expectBDDRunStatus(state.workflowRuntime, agentdomain.StatusFailed); err != nil {
			return err
		}
		var output map[string]any
		if err := json.Unmarshal(state.workflowRuntime.lastRun.Output, &output); err != nil {
			return err
		}
		if output["error"] != agentdomain.ErrDSLWorkflowNotActive.Error() {
			return fmt.Errorf("unexpected archived workflow output = %#v", output)
		}
		return nil
	})

	ctx.Step(`^a scheduled workflow resume points to a failing step$`, func() error {
		runtimeState, err := setupWorkflowRuntimeBDD(tooldomain.BuiltinUpdateCase)
		if err != nil {
			return err
		}
		seedAgentErr := runtimeState.seedAgentDefinition(bddAgentA6Failure, "resume_failure_case", bddAgentTypeDSL)
		if seedAgentErr != nil {
			return seedAgentErr
		}
		seedWorkflowErr := runtimeState.seedWorkflowRecord("wf_a6_failure", bddAgentA6Failure, "resume_failure_case", "WORKFLOW resume_failure_case\nON case.created\nSET case.status = \"resolved\"", "active")
		if seedWorkflowErr != nil {
			return seedWorkflowErr
		}
		seedRunErr := runtimeState.seedAcceptedRun(bddAgentA6Failure, "wf_a6_failure", 0)
		if seedRunErr != nil {
			return seedRunErr
		}
		state.workflowRuntime = runtimeState
		return nil
	})
	ctx.Step(`^the resume handler processes the failing workflow job$`, func() error {
		if state.workflowRuntime == nil || len(state.workflowRuntime.lastJobs) == 0 {
			return godog.ErrPending
		}
		state.workflowRuntime.lastErr = state.workflowRuntime.resumeHandler.Handle(context.Background(), state.workflowRuntime.lastJobs[0])
		_ = state.workflowRuntime.refreshRunFromCurrent()
		return nil
	})
	ctx.Step(`^the deferred resume fails safely$`, func() error {
		return expectBDDRunStatus(state.workflowRuntime, agentdomain.StatusFailed)
	})
	ctx.Step(`^the workflow run records the resume execution error$`, func() error {
		if state.workflowRuntime == nil || len(state.workflowRuntime.lastSteps) == 0 {
			return godog.ErrPending
		}
		dslSteps := filterBDDRunStepsByType(state.workflowRuntime.lastSteps, agentdomain.StepTypeDSLStatement)
		if len(dslSteps) != 1 {
			return fmt.Errorf(bddOneDSLStepErrFmt, len(dslSteps))
		}
		if dslSteps[0].Status != agentdomain.StepStatusFailed {
			return fmt.Errorf(bddStepStatusErrFmt, dslSteps[0].Status, agentdomain.StepStatusFailed)
		}
		return nil
	})
}

func setupWorkflowRuntimeBDD(failingTool string) (*workflowRuntimeState, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}
	if err = isqlite.MigrateUp(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err = db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES (?, 'Workflow BDD', 'workflow-bdd', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, bddWorkspaceID); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err = db.Exec(`
		INSERT INTO user_account (id, workspace_id, email, display_name, status, created_at, updated_at)
		VALUES (?, ?, 'bdd@example.com', 'BDD User', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, bddUserID, bddWorkspaceID); err != nil {
		_ = db.Close()
		return nil, err
	}

	registry := agentdomain.NewRunnerRegistry()
	orchestrator := agentdomain.NewOrchestratorWithRegistry(db, registry)
	toolRegistry, err := setupBDDToolRegistry(db, failingTool)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	schedulerRepo := schedulerdomain.NewRepository(db)
	schedulerSvc := schedulerdomain.NewService(schedulerRepo)
	approvalSvc := policydomain.NewApprovalService(db, nil)
	runner := agentdomain.NewDSLRunner(db)
	resumeHandler := agentdomain.NewWorkflowResumeHandler(runner, &agentdomain.RunContext{
		Orchestrator:    orchestrator,
		ToolRegistry:    toolRegistry,
		ApprovalService: approvalSvc,
		Scheduler:       schedulerSvc,
		RunnerRegistry:  registry,
		DB:              db,
	})

	return &workflowRuntimeState{
		db:            db,
		orchestrator:  orchestrator,
		registry:      registry,
		runner:        runner,
		toolRegistry:  toolRegistry,
		approvalSvc:   approvalSvc,
		schedulerRepo: schedulerRepo,
		schedulerSvc:  schedulerSvc,
		resumeHandler: resumeHandler,
	}, nil
}

func setupBDDToolRegistry(db *sql.DB, failingTool string) (*tooldomain.ToolRegistry, error) {
	registry := tooldomain.NewToolRegistry(db)
	defs := []struct {
		name   string
		schema string
	}{
		{
			name:   tooldomain.BuiltinUpdateCase,
			schema: `{"type":"object","required":["case_id"],"properties":{"case_id":{"type":"string"},"status":{"type":"string"},"priority":{"type":"string"},"approval":{"type":"object"}},"additionalProperties":false}`,
		},
		{
			name:   tooldomain.BuiltinCreateTask,
			schema: `{"type":"object","required":["owner_id","title","entity_type","entity_id"],"properties":{"owner_id":{"type":"string"},"title":{"type":"string"},"entity_type":{"type":"string"},"entity_id":{"type":"string"}},"additionalProperties":false}`,
		},
		{
			name:   tooldomain.BuiltinSendReply,
			schema: `{"type":"object","required":["case_id","body"],"properties":{"case_id":{"type":"string"},"body":{"type":"string"}},"additionalProperties":false}`,
		},
	}
	for _, def := range defs {
		if _, err := registry.CreateToolDefinition(context.Background(), tooldomain.CreateToolDefinitionInput{
			WorkspaceID: bddWorkspaceID,
			Name:        def.name,
			InputSchema: json.RawMessage(def.schema),
		}); err != nil {
			return nil, err
		}
		executor := bddToolExecutor{result: json.RawMessage(`{"status":"ok"}`)}
		if def.name == failingTool {
			executor.err = errors.New("bdd tool failure")
		}
		if err := registry.Register(def.name, executor); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

func (s *workflowRuntimeState) seedActiveWorkflow(agentID string, workflowID string, name string, dsl string) error {
	if err := s.seedAgentDefinition(agentID, name, bddAgentTypeDSL); err != nil {
		return err
	}
	if err := s.seedWorkflowRecord(workflowID, agentID, name, dsl, "active"); err != nil {
		return err
	}
	s.activeAgentID = agentID
	return nil
}

func (s *workflowRuntimeState) seedAgentDefinition(agentID string, name string, agentType string) error {
	_, err := s.db.Exec(`
		INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, agentID, bddWorkspaceID, name, agentType)
	return err
}

func (s *workflowRuntimeState) seedWorkflowRecord(workflowID string, agentID string, name string, dsl string, status string) error {
	archivedAt := any(nil)
	if status == "archived" {
		archivedAt = time.Now().UTC()
	}
	_, err := s.db.Exec(`
		INSERT INTO workflow (id, workspace_id, agent_definition_id, name, dsl_source, version, status, archived_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 1, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, workflowID, bddWorkspaceID, agentID, name, dsl, status, archivedAt)
	return err
}

func (s *workflowRuntimeState) registerNestedPendingRunner() error {
	return s.registry.Register("nested_pending", bddPendingNestedRunner{})
}

func (s *workflowRuntimeState) runWorkflow(agentID string, triggerType string, triggerContext json.RawMessage) error {
	if s == nil {
		return godog.ErrPending
	}
	run, err := s.runner.Run(context.Background(), &agentdomain.RunContext{
		Orchestrator:    s.orchestrator,
		ToolRegistry:    s.toolRegistry,
		ApprovalService: s.approvalSvc,
		Scheduler:       s.schedulerSvc,
		RunnerRegistry:  s.registry,
		DB:              s.db,
	}, agentdomain.TriggerAgentInput{
		AgentID:        agentID,
		WorkspaceID:    bddWorkspaceID,
		TriggeredBy:    stringPtrBDD(bddUserID),
		TriggerType:    triggerType,
		TriggerContext: triggerContext,
	})
	if err != nil {
		s.lastErr = err
		return err
	}
	return s.refreshRun(run.ID)
}

func (s *workflowRuntimeState) seedAcceptedRun(agentID string, workflowID string, resumeStepIndex int) error {
	run, err := s.orchestrator.TriggerAgent(context.Background(), agentdomain.TriggerAgentInput{
		AgentID:        agentID,
		WorkspaceID:    bddWorkspaceID,
		TriggerType:    agentdomain.TriggerTypeEvent,
		TriggerContext: json.RawMessage(bddEventPayloadCase),
	})
	if err != nil {
		return err
	}
	if _, err = s.orchestrator.UpdateAgentRunStatus(context.Background(), bddWorkspaceID, run.ID, agentdomain.StatusAccepted); err != nil {
		return err
	}
	job, err := s.schedulerSvc.Schedule(context.Background(), schedulerdomain.ScheduleJobInput{
		WorkspaceID: bddWorkspaceID,
		JobType:     schedulerdomain.JobTypeWorkflowResume,
		Payload: schedulerdomain.WorkflowResumePayload{
			WorkflowID:      workflowID,
			RunID:           run.ID,
			ResumeStepIndex: resumeStepIndex,
		},
		ExecuteAt: time.Now().UTC(),
		SourceID:  workflowID,
	})
	if err != nil {
		return err
	}
	s.lastRun = run
	s.lastJobs = []*schedulerdomain.ScheduledJob{job}
	return nil
}

func (s *workflowRuntimeState) resumeJob(job *schedulerdomain.ScheduledJob) error {
	if err := s.resumeHandler.Handle(context.Background(), job); err != nil {
		s.lastErr = err
		_ = s.refreshRunFromCurrent()
		return err
	}
	return s.refreshRunFromCurrent()
}

func (s *workflowRuntimeState) refreshRunFromCurrent() error {
	if s.lastRun == nil {
		return godog.ErrPending
	}
	return s.refreshRun(s.lastRun.ID)
}

func (s *workflowRuntimeState) refreshRun(runID string) error {
	run, err := s.orchestrator.GetAgentRun(context.Background(), bddWorkspaceID, runID)
	if err != nil {
		return err
	}
	steps, err := s.orchestrator.ListRunSteps(context.Background(), bddWorkspaceID, runID)
	if err != nil {
		return err
	}
	s.lastRun = run
	s.lastSteps = steps
	return nil
}

func expectBDDRunStatus(state *workflowRuntimeState, status string) error {
	if state == nil || state.lastRun == nil {
		return godog.ErrPending
	}
	if state.lastRun.Status != status {
		return fmt.Errorf("run status = %s, want %s", state.lastRun.Status, status)
	}
	return nil
}

func filterBDDRunStepsByType(steps []*agentdomain.RunStep, stepType string) []*agentdomain.RunStep {
	filtered := make([]*agentdomain.RunStep, 0, len(steps))
	for _, step := range steps {
		if step.StepType == stepType {
			filtered = append(filtered, step)
		}
	}
	return filtered
}

func stringPtrBDD(value string) *string {
	return &value
}
