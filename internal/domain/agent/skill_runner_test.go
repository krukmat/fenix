package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/policy"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
	isqlite "github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
	_ "modernc.org/sqlite"
)

type stubToolExecutor struct {
	result json.RawMessage
}

func (s stubToolExecutor) Execute(_ context.Context, _ json.RawMessage) (json.RawMessage, error) {
	return s.result, nil
}

func TestSkillRunnerRunLoadsBridgeWorkflowFromInput(t *testing.T) {
	t.Parallel()

	db := setupSkillRunnerDB(t)
	registry := NewRunnerRegistry()
	registerSkillAgentTargets(t, db, registry, "evaluate_intent")
	orch := NewOrchestratorWithRegistry(db, registry)
	runner := NewSkillRunner(db)

	run, err := runner.Run(context.Background(), &RunContext{
		Orchestrator:   orch,
		RunnerRegistry: registry,
		DB:             db,
	}, TriggerAgentInput{
		AgentID:     "agent_skill_1",
		WorkspaceID: "ws_skill",
		TriggerType: TriggerTypeManual,
		Inputs: json.RawMessage(`{
			"name":"qualify_lead_bridge",
			"trigger":{"event":"lead.created"},
			"steps":[{"id":"step_1","action":{"verb":"AGENT","target":"evaluate_intent"}}]
		}`),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if run.Status != StatusSuccess {
		t.Fatalf("status = %s, want %s", run.Status, StatusSuccess)
	}
	var output SkillRunOutput
	if err := json.Unmarshal(run.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(output.Steps) != 1 || output.Steps[0].ID != "step_1" {
		t.Fatalf("unexpected step execution output: %#v", output.Steps)
	}
}

func TestSkillRunnerRunLoadsActiveSkillDefinition(t *testing.T) {
	t.Parallel()

	db := setupSkillRunnerDB(t)
	registry := NewRunnerRegistry()
	registerSkillAgentTargets(t, db, registry, "evaluate_intent")
	orch := NewOrchestratorWithRegistry(db, registry)
	runner := NewSkillRunner(db)

	mustExecSkillRunner(t, db, `INSERT INTO skill_definition (id, workspace_id, name, description, steps, agent_definition_id, status, created_at, updated_at)
	VALUES ('skill_1', 'ws_skill', 'qualify_lead_bridge', 'desc', '[{"id":"step_1","action":{"verb":"AGENT","target":"evaluate_intent"}}]', 'agent_skill_1', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)

	run, err := runner.Run(context.Background(), &RunContext{
		Orchestrator:   orch,
		RunnerRegistry: registry,
		DB:             db,
	}, TriggerAgentInput{
		AgentID:     "agent_skill_1",
		WorkspaceID: "ws_skill",
		TriggerType: TriggerTypeManual,
		Inputs:      json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	var output SkillRunOutput
	if err := json.Unmarshal(run.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if output.Source != "skill_definition" {
		t.Fatalf("output.Source = %s, want skill_definition", output.Source)
	}
	if output.StepCount != 1 {
		t.Fatalf("output.StepCount = %d, want 1", output.StepCount)
	}
	if len(output.Steps) != 1 || output.Steps[0].Verb != BridgeVerbAgent {
		t.Fatalf("unexpected output.Steps = %#v", output.Steps)
	}
}

func TestSkillRunnerRunExecutesAgentStepViaOrchestrator(t *testing.T) {
	t.Parallel()

	db := setupSkillRunnerDB(t)
	registry := NewRunnerRegistry()
	registerSkillAgentTargets(t, db, registry, "evaluate_intent")
	orch := NewOrchestratorWithRegistry(db, registry)
	runner := NewSkillRunner(db)

	run, err := runner.Run(context.Background(), &RunContext{
		Orchestrator:   orch,
		RunnerRegistry: registry,
		DB:             db,
	}, TriggerAgentInput{
		AgentID:     "agent_skill_1",
		WorkspaceID: "ws_skill",
		TriggerType: TriggerTypeManual,
		Inputs: json.RawMessage(`{
			"name":"qualify_lead_bridge",
			"trigger":{"event":"lead.created"},
			"steps":[{"id":"step_1","action":{"verb":"AGENT","target":"evaluate_intent"}}]
		}`),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if run.Status != StatusSuccess {
		t.Fatalf("status = %s, want %s", run.Status, StatusSuccess)
	}

	runs, total, err := orch.ListAgentRuns(context.Background(), "ws_skill", 50, 0)
	if err != nil {
		t.Fatalf("ListAgentRuns() error = %v", err)
	}
	if total < 2 || len(runs) < 2 {
		t.Fatalf("expected parent and nested runs, total=%d len=%d", total, len(runs))
	}
}

func TestSkillRunnerRunRequiresOrchestrator(t *testing.T) {
	t.Parallel()

	db := setupSkillRunnerDB(t)
	runner := NewSkillRunner(db)

	_, err := runner.Run(context.Background(), &RunContext{}, TriggerAgentInput{
		AgentID:     "agent_skill_1",
		WorkspaceID: "ws_skill",
		TriggerType: TriggerTypeManual,
	})
	if !errors.Is(err, ErrSkillRunnerMissingOrchestrator) {
		t.Fatalf("expected ErrSkillRunnerMissingOrchestrator, got %v", err)
	}
}

func TestSkillRunnerRunFailsWithoutActiveDefinition(t *testing.T) {
	t.Parallel()

	db := setupSkillRunnerDB(t)
	registry := NewRunnerRegistry()
	registerSkillAgentTargets(t, db, registry, "evaluate_intent", "enrich_lead", "notify_owner")
	orch := NewOrchestratorWithRegistry(db, registry)
	runner := NewSkillRunner(db)

	_, err := runner.Run(context.Background(), &RunContext{
		Orchestrator: orch,
		DB:           db,
	}, TriggerAgentInput{
		AgentID:     "agent_skill_1",
		WorkspaceID: "ws_skill",
		TriggerType: TriggerTypeManual,
		Inputs:      json.RawMessage(`{}`),
	})
	if !errors.Is(err, ErrSkillDefinitionNotFound) {
		t.Fatalf("expected ErrSkillDefinitionNotFound, got %v", err)
	}
}

func TestSkillRunnerRunExecutesStepsInOrder(t *testing.T) {
	t.Parallel()

	db := setupSkillRunnerDB(t)
	registry := NewRunnerRegistry()
	registerSkillAgentTargets(t, db, registry, "evaluate_intent", "enrich_lead", "notify_owner")
	orch := NewOrchestratorWithRegistry(db, registry)
	runner := NewSkillRunner(db)

	run, err := runner.Run(context.Background(), &RunContext{
		Orchestrator: orch,
		DB:           db,
	}, TriggerAgentInput{
		AgentID:     "agent_skill_1",
		WorkspaceID: "ws_skill",
		TriggerType: TriggerTypeManual,
		Inputs: json.RawMessage(`{
			"name":"sequence_bridge",
			"trigger":{"event":"lead.created"},
			"steps":[
				{"id":"step_1","action":{"verb":"AGENT","target":"evaluate_intent"}},
				{"id":"step_2","action":{"verb":"AGENT","target":"enrich_lead"}},
				{"id":"step_3","action":{"verb":"AGENT","target":"notify_owner"}}
			]
		}`),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	var output SkillRunOutput
	if err := json.Unmarshal(run.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(output.Steps) != 3 {
		t.Fatalf("len(output.Steps) = %d, want 3", len(output.Steps))
	}
	if output.Steps[0].ID != "step_1" || output.Steps[1].ID != "step_2" || output.Steps[2].ID != "step_3" {
		t.Fatalf("step order mismatch: %#v", output.Steps)
	}
}

func TestSkillRunnerRunStopsOnTerminalStepError(t *testing.T) {
	t.Parallel()

	db := setupSkillRunnerDB(t)
	registry := NewRunnerRegistry()
	orch := NewOrchestratorWithRegistry(db, registry)
	runner := NewSkillRunner(db)

	run, err := runner.Run(context.Background(), &RunContext{
		Orchestrator: orch,
		DB:           db,
	}, TriggerAgentInput{
		AgentID:     "agent_skill_1",
		WorkspaceID: "ws_skill",
		TriggerType: TriggerTypeManual,
		Inputs: json.RawMessage(`{
			"name":"broken_sequence_bridge",
			"trigger":{"event":"lead.created"},
			"steps":[
				{"id":"step_1","action":{"verb":"AGENT","target":"evaluate_intent"}},
				{"id":"step_2","action":{"verb":"NOTIFY","target":"salesperson"}},
				{"id":"step_3","action":{"verb":"AGENT","target":"notify_owner"}}
			]
		}`),
	})
	if err != nil {
		t.Fatalf("Run() returned transport error = %v", err)
	}
	if run.Status != StatusFailed {
		t.Fatalf("status = %s, want %s", run.Status, StatusFailed)
	}

	var output map[string]any
	if err := json.Unmarshal(run.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if output["error"] == nil {
		t.Fatalf("expected terminal error payload, got %#v", output)
	}
}

func TestSkillRunnerRunExecutesMappedSetAndNotifyThroughToolRegistry(t *testing.T) {
	t.Parallel()

	db := setupSkillRunnerDB(t)
	registry := NewRunnerRegistry()
	registerSkillAgentTargets(t, db, registry, "qualify_lead")
	orch := NewOrchestratorWithRegistry(db, registry)
	runner := NewSkillRunner(db)
	toolRegistry := setupSkillToolRegistry(t, db)

	run, err := runner.Run(context.Background(), &RunContext{
		Orchestrator: orch,
		ToolRegistry: toolRegistry,
		DB:           db,
	}, TriggerAgentInput{
		AgentID:     "agent_skill_1",
		WorkspaceID: "ws_skill",
		TriggerType: TriggerTypeEvent,
		TriggerContext: json.RawMessage(`{
			"case":{"id":"case_1"},
			"owner_id":"user_skill"
		}`),
		Inputs: json.RawMessage(`{
			"name":"mapped_actions_bridge",
			"trigger":{"event":"case.created"},
			"steps":[
				{"id":"step_1","action":{"verb":"SET","target":"case.status","args":{"value":"resolved"}}},
				{"id":"step_2","action":{"verb":"NOTIFY","target":"salesperson","args":{"message":"review resolution"}}}
			]
		}`),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if run.Status != StatusSuccess {
		t.Fatalf("status = %s, want %s", run.Status, StatusSuccess)
	}

	var toolCalls []ToolCall
	if err := json.Unmarshal(run.ToolCalls, &toolCalls); err != nil {
		t.Fatalf("unmarshal tool calls: %v", err)
	}
	if len(toolCalls) != 2 {
		t.Fatalf("len(toolCalls) = %d, want 2", len(toolCalls))
	}
	if toolCalls[0].ToolName != tool.BuiltinUpdateCase || toolCalls[1].ToolName != tool.BuiltinCreateTask {
		t.Fatalf("unexpected tool order: %#v", toolCalls)
	}

	steps, err := orch.ListRunSteps(context.Background(), "ws_skill", run.ID)
	if err != nil {
		t.Fatalf("ListRunSteps: %v", err)
	}
	bridgeSteps := filterRunStepsByType(steps, StepTypeBridgeStep)
	if len(bridgeSteps) != 2 {
		t.Fatalf("len(bridgeSteps) = %d, want 2", len(bridgeSteps))
	}
	if bridgeSteps[0].Status != StepStatusSuccess {
		t.Fatalf("bridge step 1 status = %s, want %s", bridgeSteps[0].Status, StepStatusSuccess)
	}
	if bridgeSteps[1].Status != StepStatusSuccess {
		t.Fatalf("bridge step 2 status = %s, want %s", bridgeSteps[1].Status, StepStatusSuccess)
	}
}

func TestSkillRunnerRunFailsWhenPolicyDeniesTool(t *testing.T) {
	t.Parallel()

	db := setupSkillRunnerDB(t)
	registry := NewRunnerRegistry()
	registerSkillAgentTargets(t, db, registry, "qualify_lead", "fallback_action")
	orch := NewOrchestratorWithRegistry(db, registry)
	runner := NewSkillRunner(db)
	toolRegistry := setupSkillToolRegistry(t, db)
	policyEngine := policy.NewPolicyEngine(db, nil, nil)
	triggeredBy := "user_skill"

	run, err := runner.Run(context.Background(), &RunContext{
		Orchestrator: orch,
		ToolRegistry: toolRegistry,
		PolicyEngine: policyEngine,
		DB:           db,
	}, TriggerAgentInput{
		AgentID:      "agent_skill_1",
		WorkspaceID:  "ws_skill",
		TriggeredBy:  &triggeredBy,
		TriggerType:  TriggerTypeEvent,
		TriggerContext: json.RawMessage(`{
			"case":{"id":"case_1"},
			"owner_id":"user_skill"
		}`),
		Inputs: json.RawMessage(`{
			"name":"policy_denied_bridge",
			"trigger":{"event":"case.created"},
			"steps":[
				{"id":"step_1","action":{"verb":"SET","target":"case.status","args":{"value":"resolved"}}}
			]
		}`),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if run.Status != StatusFailed {
		t.Fatalf("status = %s, want %s", run.Status, StatusFailed)
	}
}

func TestSkillRunnerRunExecutesBridgeWorkflowEndToEnd(t *testing.T) {
	t.Parallel()

	db := setupSkillRunnerDB(t)
	seedSkillRunnerRole(t, db, `{"tools":["update_case","create_task"]}`)

	orch := NewOrchestratorWithRegistry(db, NewRunnerRegistry())
	runner := NewSkillRunner(db)
	toolRegistry := setupSkillToolRegistry(t, db)
	policyEngine := policy.NewPolicyEngine(db, nil, nil)
	triggeredBy := "user_skill"

	run, err := runner.Run(context.Background(), &RunContext{
		Orchestrator: orch,
		ToolRegistry: toolRegistry,
		PolicyEngine: policyEngine,
		DB:           db,
	}, TriggerAgentInput{
		AgentID:      "agent_skill_1",
		WorkspaceID:  "ws_skill",
		TriggeredBy:  &triggeredBy,
		TriggerType:  TriggerTypeEvent,
		TriggerContext: json.RawMessage(`{
			"case":{"id":"case_1","priority":"high"},
			"owner_id":"user_skill"
		}`),
		Inputs: json.RawMessage(`{
			"name":"resolve_support_case_bridge",
			"trigger":{"event":"case.created"},
			"steps":[
				{
					"id":"step_set_status",
					"condition":{"left":"case.priority","operator":"IN","right":["high","urgent"]},
					"action":{"verb":"SET","target":"case.status","args":{"value":"resolved"}}
				},
				{
					"id":"step_notify_owner",
					"action":{"verb":"NOTIFY","target":"salesperson","args":{"message":"review resolved case"}}
				}
			]
		}`),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if run.Status != StatusSuccess {
		t.Fatalf("status = %s, want %s", run.Status, StatusSuccess)
	}

	var output SkillRunOutput
	if err := json.Unmarshal(run.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if output.BridgeName != "resolve_support_case_bridge" {
		t.Fatalf("output.BridgeName = %s, want resolve_support_case_bridge", output.BridgeName)
	}
	if len(output.Steps) != 2 {
		t.Fatalf("len(output.Steps) = %d, want 2", len(output.Steps))
	}
	if output.Steps[0].Status != StatusSuccess || output.Steps[1].Status != StatusSuccess {
		t.Fatalf("unexpected output steps: %#v", output.Steps)
	}

	var toolCalls []ToolCall
	if err := json.Unmarshal(run.ToolCalls, &toolCalls); err != nil {
		t.Fatalf("unmarshal tool calls: %v", err)
	}
	if len(toolCalls) != 2 {
		t.Fatalf("len(toolCalls) = %d, want 2", len(toolCalls))
	}
	if toolCalls[0].ToolName != tool.BuiltinUpdateCase || toolCalls[1].ToolName != tool.BuiltinCreateTask {
		t.Fatalf("unexpected tool calls: %#v", toolCalls)
	}

	steps, err := orch.ListRunSteps(context.Background(), "ws_skill", run.ID)
	if err != nil {
		t.Fatalf("ListRunSteps: %v", err)
	}
	bridgeSteps := filterRunStepsByType(steps, StepTypeBridgeStep)
	if len(bridgeSteps) != 2 {
		t.Fatalf("len(bridgeSteps) = %d, want 2", len(bridgeSteps))
	}
	if bridgeSteps[0].Status != StepStatusSuccess || bridgeSteps[1].Status != StepStatusSuccess {
		t.Fatalf("unexpected bridge trace statuses: %#v", bridgeSteps)
	}
}

func TestSkillRunnerRunPausesWhenApprovalIsRequired(t *testing.T) {
	t.Parallel()

	db := setupSkillRunnerDB(t)
	registry := NewRunnerRegistry()
	registerSkillAgentTargets(t, db, registry, "qualify_lead")
	orch := NewOrchestratorWithRegistry(db, registry)
	runner := NewSkillRunner(db)
	toolRegistry := setupSkillToolRegistry(t, db)
	approvalService := policy.NewApprovalService(db, nil)
	triggeredBy := "user_skill"

	run, err := runner.Run(context.Background(), &RunContext{
		Orchestrator:    orch,
		ToolRegistry:    toolRegistry,
		ApprovalService: approvalService,
		DB:              db,
	}, TriggerAgentInput{
		AgentID:      "agent_skill_1",
		WorkspaceID:  "ws_skill",
		TriggeredBy:  &triggeredBy,
		TriggerType:  TriggerTypeEvent,
		TriggerContext: json.RawMessage(`{
			"case":{"id":"case_1"},
			"owner_id":"user_skill"
		}`),
		Inputs: json.RawMessage(`{
			"name":"approval_bridge",
			"trigger":{"event":"case.created"},
			"steps":[
				{
					"id":"step_1",
					"action":{
						"verb":"SET",
						"target":"case.status",
						"args":{
							"value":"resolved",
							"approval":{"required":true,"approver_id":"user_skill","reason":"sensitive case mutation"}
						}
					}
				}
			]
		}`),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if run.Status != StatusAccepted {
		t.Fatalf("status = %s, want %s", run.Status, StatusAccepted)
	}

	var output map[string]any
	if err := json.Unmarshal(run.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if output["action"] != "pending_approval" {
		t.Fatalf("action = %#v, want pending_approval", output["action"])
	}
	if output["approval_id"] == nil {
		t.Fatalf("expected approval_id in output: %#v", output)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM approval_request WHERE workspace_id = 'ws_skill'`).Scan(&count); err != nil {
		t.Fatalf("count approval_request: %v", err)
	}
	if count != 1 {
		t.Fatalf("approval count = %d, want 1", count)
	}
}

func TestSkillRunnerRunExecutesConditionalStepWhenTrue(t *testing.T) {
	t.Parallel()

	db := setupSkillRunnerDB(t)
	registry := NewRunnerRegistry()
	registerSkillAgentTargets(t, db, registry, "qualify_lead")
	orch := NewOrchestratorWithRegistry(db, registry)
	runner := NewSkillRunner(db)

	run, err := runner.Run(context.Background(), &RunContext{
		Orchestrator: orch,
		RunnerRegistry: registry,
		DB:           db,
	}, TriggerAgentInput{
		AgentID:     "agent_skill_1",
		WorkspaceID: "ws_skill",
		TriggerType: TriggerTypeEvent,
		TriggerContext: json.RawMessage(`{
			"lead":{"score":0.9}
		}`),
		Inputs: json.RawMessage(`{
			"name":"conditional_true_bridge",
			"trigger":{"event":"lead.created"},
			"steps":[
				{
					"id":"step_1",
					"condition":{"left":"lead.score","operator":">=","right":0.8},
					"action":{"verb":"AGENT","target":"qualify_lead"}
				}
			]
		}`),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	var output SkillRunOutput
	if err := json.Unmarshal(run.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if output.Steps[0].Status != StatusSuccess {
		t.Fatalf("step status = %s, want %s", output.Steps[0].Status, StatusSuccess)
	}
}

func TestSkillRunnerRunSkipsConditionalStepWhenFalse(t *testing.T) {
	t.Parallel()

	db := setupSkillRunnerDB(t)
	registry := NewRunnerRegistry()
	registerSkillAgentTargets(t, db, registry, "qualify_lead", "fallback_action")
	orch := NewOrchestratorWithRegistry(db, registry)
	runner := NewSkillRunner(db)

	run, err := runner.Run(context.Background(), &RunContext{
		Orchestrator: orch,
		RunnerRegistry: registry,
		DB:           db,
	}, TriggerAgentInput{
		AgentID:     "agent_skill_1",
		WorkspaceID: "ws_skill",
		TriggerType: TriggerTypeEvent,
		TriggerContext: json.RawMessage(`{
			"lead":{"score":0.4}
		}`),
		Inputs: json.RawMessage(`{
			"name":"conditional_false_bridge",
			"trigger":{"event":"lead.created"},
			"steps":[
				{
					"id":"step_1",
					"condition":{"left":"lead.score","operator":">=","right":0.8},
					"action":{"verb":"AGENT","target":"qualify_lead"}
				},
				{
					"id":"step_2",
					"action":{"verb":"AGENT","target":"fallback_action"}
				}
			]
		}`),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	var output SkillRunOutput
	if err := json.Unmarshal(run.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(output.Steps) != 2 {
		t.Fatalf("unexpected output for conditional false path: status=%s output=%s", run.Status, string(run.Output))
	}
	if output.Steps[0].Status != StepStatusSkipped {
		t.Fatalf("step 1 status = %s, want %s", output.Steps[0].Status, StepStatusSkipped)
	}
	if output.Steps[1].Status != StatusSuccess {
		t.Fatalf("step 2 status = %s, want %s", output.Steps[1].Status, StatusSuccess)
	}

	steps, err := orch.ListRunSteps(context.Background(), "ws_skill", run.ID)
	if err != nil {
		t.Fatalf("ListRunSteps: %v", err)
	}
	bridgeSteps := filterRunStepsByType(steps, StepTypeBridgeStep)
	if len(bridgeSteps) != 2 {
		t.Fatalf("len(bridgeSteps) = %d, want 2", len(bridgeSteps))
	}
	if bridgeSteps[0].Status != StepStatusSkipped {
		t.Fatalf("conditional trace status = %s, want %s", bridgeSteps[0].Status, StepStatusSkipped)
	}
	if bridgeSteps[1].Status != StepStatusSuccess {
		t.Fatalf("fallback trace status = %s, want %s", bridgeSteps[1].Status, StepStatusSuccess)
	}
}

func TestSkillRunnerRunFailsOnConditionalTypeMismatch(t *testing.T) {
	t.Parallel()

	db := setupSkillRunnerDB(t)
	registry := NewRunnerRegistry()
	registerSkillAgentTargets(t, db, registry, "qualify_lead")
	orch := NewOrchestratorWithRegistry(db, registry)
	runner := NewSkillRunner(db)

	run, err := runner.Run(context.Background(), &RunContext{
		Orchestrator: orch,
		RunnerRegistry: registry,
		DB:           db,
	}, TriggerAgentInput{
		AgentID:     "agent_skill_1",
		WorkspaceID: "ws_skill",
		TriggerType: TriggerTypeEvent,
		TriggerContext: json.RawMessage(`{
			"lead":{"score":"high"}
		}`),
		Inputs: json.RawMessage(`{
			"name":"conditional_mismatch_bridge",
			"trigger":{"event":"lead.created"},
			"steps":[
				{
					"id":"step_1",
					"condition":{"left":"lead.score","operator":">=","right":0.8},
					"action":{"verb":"AGENT","target":"qualify_lead"}
				}
			]
		}`),
	})
	if err != nil {
		t.Fatalf("Run() returned transport error = %v", err)
	}
	if run.Status != StatusFailed {
		t.Fatalf("status = %s, want %s", run.Status, StatusFailed)
	}

	steps, err := orch.ListRunSteps(context.Background(), "ws_skill", run.ID)
	if err != nil {
		t.Fatalf("ListRunSteps: %v", err)
	}
	bridgeSteps := filterRunStepsByType(steps, StepTypeBridgeStep)
	if len(bridgeSteps) != 1 {
		t.Fatalf("len(bridgeSteps) = %d, want 1", len(bridgeSteps))
	}
	if bridgeSteps[0].Status != StepStatusFailed {
		t.Fatalf("failed trace status = %s, want %s", bridgeSteps[0].Status, StepStatusFailed)
	}
}

func filterRunStepsByType(steps []*RunStep, stepType string) []*RunStep {
	filtered := make([]*RunStep, 0)
	for _, step := range steps {
		if step.StepType == stepType {
			filtered = append(filtered, step)
		}
	}
	return filtered
}

func setupSkillRunnerDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := isqlite.MigrateUp(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	mustExecSkillRunner(t, db, `INSERT INTO workspace (id, name, slug, created_at, updated_at) VALUES ('ws_skill', 'Skill Runner', 'skill-runner', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExecSkillRunner(t, db, `INSERT INTO user_account (id, workspace_id, email, display_name, status, created_at, updated_at) VALUES ('user_skill', 'ws_skill', 'skill@example.com', 'Skill User', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExecSkillRunner(t, db, `INSERT INTO agent_definition (id, workspace_id, name, description, agent_type, objective, allowed_tools, limits, trigger_config, status, created_at, updated_at)
	VALUES ('agent_skill_1', 'ws_skill', 'Skill Agent', 'Bridge runner', 'skill', '{}', '[]', '{}', '{}', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func seedSkillRunnerRole(t *testing.T, db *sql.DB, permissions string) {
	t.Helper()

	now := time.Now().UTC().Format(time.RFC3339)
	roleID := uuid.NewV7().String()
	userRoleID := uuid.NewV7().String()

	if _, err := db.Exec(`
		INSERT INTO role (id, workspace_id, name, permissions, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, roleID, "ws_skill", "skill-runner-role", permissions, now, now); err != nil {
		t.Fatalf("insert role: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO user_role (id, user_id, role_id, created_at)
		VALUES (?, ?, ?, ?)
	`, userRoleID, "user_skill", roleID, now); err != nil {
		t.Fatalf("insert user_role: %v", err)
	}
}

func mustExecSkillRunner(t *testing.T, db *sql.DB, query string) {
	t.Helper()
	if _, err := db.Exec(query); err != nil {
		t.Fatalf("exec failed: %v", err)
	}
}

func setupSkillToolRegistry(t *testing.T, db *sql.DB) *tool.ToolRegistry {
	t.Helper()

	registry := tool.NewToolRegistry(db)

	createToolDefinition := func(name string, schema string) {
		if _, err := registry.CreateToolDefinition(context.Background(), tool.CreateToolDefinitionInput{
			WorkspaceID: "ws_skill",
			Name:        name,
			InputSchema: json.RawMessage(schema),
		}); err != nil {
			t.Fatalf("CreateToolDefinition(%s) error = %v", name, err)
		}
	}

	createToolDefinition(tool.BuiltinUpdateCase, `{"type":"object","required":["case_id"],"properties":{"case_id":{"type":"string"},"status":{"type":"string"},"priority":{"type":"string"}},"additionalProperties":false}`)
	createToolDefinition(tool.BuiltinCreateTask, `{"type":"object","required":["owner_id","title","entity_type","entity_id"],"properties":{"owner_id":{"type":"string"},"title":{"type":"string"},"entity_type":{"type":"string"},"entity_id":{"type":"string"}},"additionalProperties":false}`)
	createToolDefinition(tool.BuiltinSendReply, `{"type":"object","required":["case_id","body"],"properties":{"case_id":{"type":"string"},"body":{"type":"string"}},"additionalProperties":false}`)

	if err := registry.Register(tool.BuiltinUpdateCase, stubToolExecutor{result: json.RawMessage(`{"status":"updated"}`)}); err != nil {
		t.Fatalf("Register(update_case) error = %v", err)
	}
	if err := registry.Register(tool.BuiltinCreateTask, stubToolExecutor{result: json.RawMessage(`{"status":"created"}`)}); err != nil {
		t.Fatalf("Register(create_task) error = %v", err)
	}
	if err := registry.Register(tool.BuiltinSendReply, stubToolExecutor{result: json.RawMessage(`{"status":"sent"}`)}); err != nil {
		t.Fatalf("Register(send_reply) error = %v", err)
	}

	return registry
}

func registerSkillAgentTargets(t *testing.T, db *sql.DB, registry *RunnerRegistry, names ...string) {
	t.Helper()
	for _, name := range names {
		agentID := "agent_" + name
		mustExecSkillRunner(t, db, `INSERT INTO agent_definition (id, workspace_id, name, description, agent_type, objective, allowed_tools, limits, trigger_config, status, created_at, updated_at)
		VALUES ('`+agentID+`', 'ws_skill', '`+name+`', 'Nested target', 'nested', '{}', '[]', '{}', '{}', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	}
	if err := registry.Register("nested", stubNestedRunner{}); err != nil && !errors.Is(err, ErrRunnerAlreadyExists) {
		t.Fatalf("registry.Register(nested) error = %v", err)
	}
}
