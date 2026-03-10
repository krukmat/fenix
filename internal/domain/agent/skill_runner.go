package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/policy"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

var (
	ErrSkillRunnerMissingOrchestrator = errors.New("skill runner requires orchestrator")
	ErrSkillRunnerMissingDB           = errors.New("skill runner requires db")
	ErrSkillDefinitionNotFound        = errors.New("skill definition not found")
	ErrSkillDefinitionInactive        = errors.New("skill definition is not active")
	ErrSkillStepExecutionFailed       = errors.New("skill step execution failed")
	ErrSkillToolRegistryMissing       = errors.New("skill runner requires tool registry for mapped actions")
)

const bridgeEntityCase = "case"

type SkillRunner struct {
	db *sql.DB
}

type SkillRunOutput struct {
	BridgeName string               `json:"bridge_name"`
	Source     string               `json:"source"`
	StepCount  int                  `json:"step_count"`
	Steps      []SkillStepExecution `json:"steps"`
}

type SkillStepExecution struct {
	ID     string `json:"id"`
	Verb   string `json:"verb"`
	Target string `json:"target"`
	Status string `json:"status"`
}

type skillApprovalResult struct {
	ApprovalID string `json:"approval_id"`
	Action     string `json:"action"`
}

func NewSkillRunner(db *sql.DB) *SkillRunner {
	return &SkillRunner{db: db}
}

func (r *SkillRunner) Run(ctx context.Context, rc *RunContext, input TriggerAgentInput) (*Run, error) {
	if rc == nil || rc.Orchestrator == nil {
		return nil, ErrSkillRunnerMissingOrchestrator
	}
	if r.db == nil {
		return nil, ErrSkillRunnerMissingDB
	}

	workflow, source, evalCtx, err := r.loadBridgeWorkflow(ctx, input)
	if err != nil {
		return nil, err
	}

	accepted, err := r.triggerAndAccept(ctx, rc, input)
	if err != nil {
		return nil, err
	}

	executedSteps, toolCalls, pendingApproval, err := r.executeSequentialSteps(ctx, rc, input.WorkspaceID, accepted.ID, actorIDFromInput(input, evalCtx), workflow, evalCtx)
	return r.finalizeRun(ctx, rc, input.WorkspaceID, accepted.ID, workflow, source, executedSteps, toolCalls, pendingApproval, err)
}

func (r *SkillRunner) triggerAndAccept(ctx context.Context, rc *RunContext, input TriggerAgentInput) (*Run, error) {
	run, err := rc.Orchestrator.TriggerAgent(ctx, input)
	if err != nil {
		return nil, err
	}
	return rc.Orchestrator.UpdateAgentRunStatus(ctx, input.WorkspaceID, run.ID, StatusAccepted)
}

func (r *SkillRunner) finalizeRun(
	ctx context.Context,
	rc *RunContext,
	workspaceID, runID string,
	workflow *BridgeWorkflow,
	source string,
	executedSteps []SkillStepExecution,
	toolCalls json.RawMessage,
	pendingApproval *skillApprovalResult,
	execErr error,
) (*Run, error) {
	if execErr != nil {
		return applyFailedRunUpdate(ctx, rc, workspaceID, runID, workflow, source, toolCalls, execErr)
	}
	if pendingApproval != nil {
		return applyPendingApprovalUpdate(ctx, rc, workspaceID, runID, workflow, source, toolCalls, pendingApproval)
	}
	return applySuccessRunUpdate(ctx, rc, workspaceID, runID, workflow, source, executedSteps, toolCalls)
}

func applyFailedRunUpdate(ctx context.Context, rc *RunContext, workspaceID, runID string, workflow *BridgeWorkflow, source string, toolCalls json.RawMessage, execErr error) (*Run, error) {
	output, err := json.Marshal(map[string]any{
		"bridge_name": workflow.Name,
		"source":      source,
		"step_count":  len(workflow.Steps),
		"error":       execErr.Error(),
	})
	if err != nil {
		return nil, err
	}
	return rc.Orchestrator.UpdateAgentRun(ctx, workspaceID, runID, emptyTracesUpdate(StatusFailed, output, toolCalls, true))
}

func applyPendingApprovalUpdate(ctx context.Context, rc *RunContext, workspaceID, runID string, workflow *BridgeWorkflow, source string, toolCalls json.RawMessage, approval *skillApprovalResult) (*Run, error) {
	output, err := json.Marshal(map[string]any{
		"bridge_name": workflow.Name,
		"source":      source,
		"step_count":  len(workflow.Steps),
		"action":      "pending_approval",
		"approval_id": approval.ApprovalID,
	})
	if err != nil {
		return nil, err
	}
	return rc.Orchestrator.UpdateAgentRun(ctx, workspaceID, runID, emptyTracesUpdate(StatusAccepted, output, toolCalls, false))
}

func applySuccessRunUpdate(ctx context.Context, rc *RunContext, workspaceID, runID string, workflow *BridgeWorkflow, source string, executedSteps []SkillStepExecution, toolCalls json.RawMessage) (*Run, error) {
	output, err := json.Marshal(SkillRunOutput{
		BridgeName: workflow.Name,
		Source:     source,
		StepCount:  len(workflow.Steps),
		Steps:      executedSteps,
	})
	if err != nil {
		return nil, err
	}
	return rc.Orchestrator.UpdateAgentRun(ctx, workspaceID, runID, emptyTracesUpdate(StatusSuccess, output, toolCalls, true))
}

func emptyTracesUpdate(status string, output, toolCalls json.RawMessage, completed bool) RunUpdates {
	return RunUpdates{
		Status:               status,
		Output:               output,
		ReasoningTrace:       json.RawMessage(emptyJSONArray),
		RetrievalQueries:     json.RawMessage(emptyJSONArray),
		RetrievedEvidenceIDs: json.RawMessage(emptyJSONArray),
		ToolCalls:            toolCalls,
		Completed:            completed,
	}
}

type stepResult struct {
	execution     SkillStepExecution
	call          *ToolCall
	pendingApproval *skillApprovalResult
	done          bool // stop sequence after this step
}

func (r *SkillRunner) executeSequentialSteps(
	ctx context.Context,
	rc *RunContext,
	workspaceID string,
	runID string,
	actorID string,
	workflow *BridgeWorkflow,
	evalCtx map[string]any,
) ([]SkillStepExecution, json.RawMessage, *skillApprovalResult, error) {
	executed := make([]SkillStepExecution, 0, len(workflow.Steps))
	toolCalls := make([]ToolCall, 0)
	for _, step := range workflow.Steps {
		sr, err := r.executeSingleStep(ctx, rc, workspaceID, runID, actorID, step, evalCtx)
		if err != nil {
			return executed, marshalSkillToolCalls(toolCalls), nil, err
		}
		if sr.call != nil {
			toolCalls = append(toolCalls, *sr.call)
		}
		executed = append(executed, sr.execution)
		if sr.done {
			return executed, marshalSkillToolCalls(toolCalls), sr.pendingApproval, nil
		}
	}
	return executed, marshalSkillToolCalls(toolCalls), nil, nil
}

func (r *SkillRunner) executeSingleStep(
	ctx context.Context,
	rc *RunContext,
	workspaceID, runID, actorID string,
	step BridgeStep,
	evalCtx map[string]any,
) (stepResult, error) {
	traceStepID, traceErr := insertBridgeRunStep(ctx, rc, workspaceID, runID, marshalBridgeStepInput(step))
	if traceErr != nil {
		return stepResult{}, traceErr
	}
	if step.Condition != nil {
		return r.evaluateConditionalStep(ctx, rc, workspaceID, traceStepID, step, evalCtx)
	}
	return r.executeAndTraceStep(ctx, rc, workspaceID, traceStepID, actorID, step, evalCtx)
}

func (r *SkillRunner) evaluateConditionalStep(
	ctx context.Context,
	rc *RunContext,
	workspaceID, traceStepID string,
	step BridgeStep,
	evalCtx map[string]any,
) (stepResult, error) {
	ok, err := evaluateBridgeCondition(*step.Condition, evalCtx)
	if err != nil {
		_ = updateBridgeRunStep(ctx, rc, workspaceID, traceStepID, StepStatusFailed, nil, err)
		return stepResult{}, err
	}
	if !ok {
		_ = updateBridgeRunStep(ctx, rc, workspaceID, traceStepID, StepStatusSkipped, json.RawMessage(`{"result":"condition_false"}`), nil)
		return stepResult{execution: skippedExecution(step)}, nil
	}
	return r.executeAndTraceStep(ctx, rc, workspaceID, traceStepID, "", step, evalCtx)
}

func (r *SkillRunner) executeAndTraceStep(
	ctx context.Context,
	rc *RunContext,
	workspaceID, traceStepID, actorID string,
	step BridgeStep,
	evalCtx map[string]any,
) (stepResult, error) {
	call, pendingApproval, stepOutput, err := executeBridgeStep(ctx, rc, workspaceID, actorID, step, evalCtx)
	if err != nil {
		_ = updateBridgeRunStep(ctx, rc, workspaceID, traceStepID, StepStatusFailed, stepOutput, err)
		return stepResult{}, err
	}
	if pendingApproval != nil {
		_ = updateBridgeRunStep(ctx, rc, workspaceID, traceStepID, StepStatusRunning, stepOutput, nil)
		return stepResult{
			execution:       pendingExecution(step),
			call:            call,
			pendingApproval: pendingApproval,
			done:            true,
		}, nil
	}
	_ = updateBridgeRunStep(ctx, rc, workspaceID, traceStepID, StepStatusSuccess, stepOutput, nil)
	return stepResult{execution: successExecution(step), call: call}, nil
}

func skippedExecution(step BridgeStep) SkillStepExecution {
	return SkillStepExecution{ID: step.ID, Verb: strings.ToUpper(step.Action.Verb), Target: step.Action.Target, Status: StepStatusSkipped}
}

func successExecution(step BridgeStep) SkillStepExecution {
	return SkillStepExecution{ID: step.ID, Verb: strings.ToUpper(step.Action.Verb), Target: step.Action.Target, Status: StatusSuccess}
}

func pendingExecution(step BridgeStep) SkillStepExecution {
	return SkillStepExecution{ID: step.ID, Verb: strings.ToUpper(step.Action.Verb), Target: step.Action.Target, Status: "pending_approval"}
}

func executeBridgeStep(
	ctx context.Context,
	rc *RunContext,
	workspaceID string,
	actorID string,
	step BridgeStep,
	evalCtx map[string]any,
) (*ToolCall, *skillApprovalResult, json.RawMessage, error) {
	if err := validateBridgeStepAction(step); err != nil {
		return nil, nil, nil, err
	}
	op, err := NewVerbMapper().MapBridgeStep(step, evalCtx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%w: %w", ErrSkillStepExecutionFailed, err)
	}

	switch op.Kind {
	case RuntimeOperationTool:
		return executeMappedBridgeTool(ctx, rc, workspaceID, actorID, step, op)
	case RuntimeOperationAgent:
		return executeMappedBridgeAgent(ctx, rc, workspaceID, actorID, step, evalCtx, op)
	default:
		return nil, nil, nil, fmt.Errorf("%w: step %s: unsupported runtime operation kind", ErrSkillStepExecutionFailed, step.ID)
	}
}

func requireArg(step BridgeStep, key string) error {
	if step.Action.Args == nil {
		return fmt.Errorf("%w: step %s: %s requires args", ErrSkillStepExecutionFailed, step.ID, strings.ToUpper(step.Action.Verb))
	}
	if _, ok := step.Action.Args[key]; !ok {
		return fmt.Errorf("%w: step %s: %s requires args.%s", ErrSkillStepExecutionFailed, step.ID, strings.ToUpper(step.Action.Verb), key)
	}
	return nil
}

type mappedBridgeTool struct {
	name   string
	params map[string]any
}

func executeMappedBridgeTool(
	ctx context.Context,
	rc *RunContext,
	workspaceID string,
	actorID string,
	step BridgeStep,
	op *RuntimeOperation,
) (*ToolCall, *skillApprovalResult, json.RawMessage, error) {
	if op == nil || op.Kind != RuntimeOperationTool {
		return nil, nil, nil, fmt.Errorf("%w: bridge action is not mappable", ErrSkillStepExecutionFailed)
	}
	return executeMappedTool(ctx, rc, workspaceID, actorID, step, mappedBridgeTool{
		name:   op.ToolName,
		params: op.Params,
	})
}

func executeMappedBridgeAgent(
	ctx context.Context,
	rc *RunContext,
	workspaceID string,
	actorID string,
	step BridgeStep,
	evalCtx map[string]any,
	op *RuntimeOperation,
) (*ToolCall, *skillApprovalResult, json.RawMessage, error) {
	if err := validateBridgeAgentContext(rc, op); err != nil {
		return nil, nil, nil, err
	}
	target, err := resolveActiveAgentDefinition(ctx, rc.DB, workspaceID, op.AgentName)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := validateBridgeAgentCall(ctx, rc, actorID, target); err != nil {
		return nil, nil, nil, err
	}
	rawInputs, err := json.Marshal(op.Params)
	if err != nil {
		return nil, nil, nil, err
	}
	stored, err := invokeBridgeSubAgent(ctx, rc, workspaceID, actorID, target, rawInputs, evalCtx)
	if err != nil {
		return nil, nil, nil, err
	}
	return buildBridgeSubAgentOutput(target, stored)
}

func validateBridgeAgentContext(rc *RunContext, op *RuntimeOperation) error {
	if rc == nil || rc.Orchestrator == nil {
		return ErrSkillRunnerMissingOrchestrator
	}
	if rc.DB == nil {
		return ErrSkillRunnerMissingDB
	}
	if op == nil || op.Kind != RuntimeOperationAgent {
		return fmt.Errorf("%w: bridge action is not mappable", ErrSkillStepExecutionFailed)
	}
	return nil
}

func validateBridgeAgentCall(ctx context.Context, rc *RunContext, actorID string, target *Definition) error {
	if err := checkMappedAgentPolicy(ctx, rc, actorID, target); err != nil {
		return fmt.Errorf("%w: %w", ErrSkillStepExecutionFailed, err)
	}
	if rc.CallDepth >= dslAgentCallDepthLimit {
		return ErrDSLAgentDepthExceeded
	}
	if containsCall(rc.CallChain, target.ID) {
		return ErrDSLAgentLoopDetected
	}
	return nil
}

func invokeBridgeSubAgent(ctx context.Context, rc *RunContext, workspaceID, actorID string, target *Definition, rawInputs json.RawMessage, evalCtx map[string]any) (*Run, error) {
	var triggeredBy *string
	if strings.TrimSpace(actorID) != "" && actorID != "system" {
		triggeredBy = &actorID
	}
	subRun, err := rc.Orchestrator.ExecuteAgent(ctx, rc.WithCall(target.ID), TriggerAgentInput{
		AgentID:        target.ID,
		WorkspaceID:    workspaceID,
		TriggeredBy:    triggeredBy,
		TriggerType:    TriggerTypeManual,
		TriggerContext: marshalRuntimeContext(evalCtx),
		Inputs:         rawInputs,
	})
	if err != nil {
		return nil, err
	}
	return rc.Orchestrator.GetAgentRun(ctx, workspaceID, subRun.ID)
}

func buildBridgeSubAgentOutput(target *Definition, stored *Run) (*ToolCall, *skillApprovalResult, json.RawMessage, error) {
	output, err := json.Marshal(map[string]any{
		"agent_id": target.ID,
		"run_id":   stored.ID,
		"status":   stored.Status,
	})
	if err != nil {
		return nil, nil, nil, err
	}
	switch stored.Status {
	case StatusFailed, StatusRejected:
		return nil, nil, output, fmt.Errorf("%w: sub-agent %s returned %s", ErrSkillStepExecutionFailed, target.ID, stored.Status)
	case StatusAccepted:
		return nil, extractPendingApprovalResult(stored.Output), output, nil
	default:
		return nil, nil, output, nil
	}
}

func executeMappedTool(
	ctx context.Context,
	rc *RunContext,
	workspaceID string,
	actorID string,
	step BridgeStep,
	mapped mappedBridgeTool,
) (*ToolCall, *skillApprovalResult, json.RawMessage, error) {
	if mapped.name == "" {
		return nil, nil, nil, fmt.Errorf("%w: bridge action is not mappable", ErrSkillStepExecutionFailed)
	}
	if err := checkMappedToolPolicy(ctx, rc, actorID, mapped.name); err != nil {
		return nil, nil, nil, err
	}
	if approvalCfg := parseBridgeApprovalConfig(step.Action.Args); approvalCfg != nil && approvalCfg.Required {
		return routeToApproval(ctx, rc, workspaceID, actorID, mapped, approvalCfg)
	}
	return executeRegisteredTool(ctx, rc, workspaceID, mapped)
}

func checkMappedToolPolicy(ctx context.Context, rc *RunContext, actorID, toolName string) error {
	if rc == nil || rc.PolicyEngine == nil || actorID == "" {
		return nil
	}
	allowed, err := rc.PolicyEngine.CheckToolPermission(ctx, actorID, toolName)
	if err != nil {
		return err
	}
	if !allowed {
		return fmt.Errorf("%w: tool %s denied by policy", ErrSkillStepExecutionFailed, toolName)
	}
	return nil
}

func routeToApproval(ctx context.Context, rc *RunContext, workspaceID, actorID string, mapped mappedBridgeTool, cfg *bridgeApprovalConfig) (*ToolCall, *skillApprovalResult, json.RawMessage, error) {
	if rc == nil || rc.ApprovalService == nil {
		return nil, nil, nil, fmt.Errorf("%w: approval service is required", ErrSkillStepExecutionFailed)
	}
	approvalResult, call, output, err := createBridgeApproval(ctx, rc.ApprovalService, workspaceID, actorID, mapped, cfg)
	return call, approvalResult, output, err
}

func executeRegisteredTool(ctx context.Context, rc *RunContext, workspaceID string, mapped mappedBridgeTool) (*ToolCall, *skillApprovalResult, json.RawMessage, error) {
	if rc == nil || rc.ToolRegistry == nil {
		return nil, nil, nil, ErrSkillToolRegistryMissing
	}
	rawParams, err := json.Marshal(mapped.params)
	if err != nil {
		return nil, nil, nil, err
	}
	result, err := rc.ToolRegistry.Execute(ctx, workspaceID, mapped.name, rawParams)
	call := &ToolCall{ToolName: mapped.name, Params: rawParams, Result: result}
	if err != nil {
		call.Error = err.Error()
		output, _ := json.Marshal(map[string]any{"tool": mapped.name, "result": "failed", "error": err.Error()})
		return call, nil, output, err
	}
	output, marshalErr := json.Marshal(map[string]any{"tool": mapped.name, "result": "success"})
	if marshalErr != nil {
		return call, nil, nil, marshalErr
	}
	return call, nil, output, nil
}

func validateBridgeStepAction(step BridgeStep) error {
	verb := strings.ToUpper(strings.TrimSpace(step.Action.Verb))
	switch verb {
	case BridgeVerbSet:
		return requireArg(step, "value")
	case BridgeVerbNotify:
		return requireArg(step, "message")
	case BridgeVerbAgent:
		return nil
	default:
		return fmt.Errorf("%w: step %s: unsupported verb", ErrSkillStepExecutionFailed, step.ID)
	}
}

func extractPendingApprovalResult(raw json.RawMessage) *skillApprovalResult {
	if len(raw) == 0 || !json.Valid(raw) {
		return &skillApprovalResult{Action: "pending_approval"}
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return &skillApprovalResult{Action: "pending_approval"}
	}
	action, _ := payload["action"].(string)
	if action == "" {
		action = "pending_approval"
	}
	approvalID, _ := payload["approval_id"].(string)
	return &skillApprovalResult{
		ApprovalID: approvalID,
		Action:     action,
	}
}

func resolvePrimaryEntity(evalCtx map[string]any) (string, string) {
	for _, entityType := range []string{bridgeEntityCase, "lead", "deal", "contact"} {
		if id := resolveEntityID(evalCtx, entityType); id != "" {
			return entityType, id
		}
	}
	return "", ""
}

func resolveEntityID(evalCtx map[string]any, entityType string) string {
	value := resolveBridgeValue(evalCtx, entityType+".id")
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func resolveOwnerID(evalCtx map[string]any) string {
	for _, key := range []string{"owner_id", "salesperson_id", "user_id"} {
		if value := resolveBridgeValue(evalCtx, key); value != nil {
			return fmt.Sprint(value)
		}
	}
	return ""
}

func marshalSkillToolCalls(toolCalls []ToolCall) json.RawMessage {
	if len(toolCalls) == 0 {
		return json.RawMessage(emptyJSONArray)
	}
	raw, err := json.Marshal(toolCalls)
	if err != nil {
		return json.RawMessage(emptyJSONArray)
	}
	return raw
}

type bridgeApprovalConfig struct {
	Required   bool
	ApproverID string
	Reason     string
}

func parseBridgeApprovalConfig(args map[string]any) *bridgeApprovalConfig {
	if len(args) == 0 {
		return nil
	}
	raw, ok := args["approval"].(map[string]any)
	if !ok {
		return nil
	}
	cfg := &bridgeApprovalConfig{}
	required, hasRequired := raw["required"].(bool)
	if hasRequired {
		cfg.Required = required
	}
	approverID, hasApprover := raw["approver_id"].(string)
	if hasApprover {
		cfg.ApproverID = strings.TrimSpace(approverID)
	}
	reason, hasReason := raw["reason"].(string)
	if hasReason {
		cfg.Reason = strings.TrimSpace(reason)
	}
	return cfg
}

func createBridgeApproval(
	ctx context.Context,
	approvalService *policy.ApprovalService,
	workspaceID string,
	actorID string,
	mapped mappedBridgeTool,
	cfg *bridgeApprovalConfig,
) (*skillApprovalResult, *ToolCall, json.RawMessage, error) {
	rawParams, err := json.Marshal(mapped.params)
	if err != nil {
		return nil, nil, nil, err
	}
	resourceType := "tool"
	resourceID := mapped.name
	approverID := cfg.ApproverID
	if approverID == "" {
		approverID = actorID
	}
	req, err := approvalService.CreateApprovalRequest(ctx, policy.CreateApprovalRequestInput{
		WorkspaceID:  workspaceID,
		RequestedBy:  actorID,
		ApproverID:   approverID,
		Action:       mapped.name,
		ResourceType: &resourceType,
		ResourceID:   &resourceID,
		Payload:      rawParams,
		Reason:       stringPtrOrNil(cfg.Reason),
		ExpiresAt:    time.Now().UTC().Add(24 * time.Hour),
	})
	if err != nil {
		return nil, nil, nil, err
	}
	callResult, _ := json.Marshal(map[string]any{"approval_id": req.ID})
	output, _ := json.Marshal(map[string]any{
		"result":      "pending_approval",
		"approval_id": req.ID,
		"tool":        mapped.name,
	})
	return &skillApprovalResult{
			ApprovalID: req.ID,
			Action:     "pending_approval",
		}, &ToolCall{
			ToolName: "approval.requested",
			Params:   rawParams,
			Result:   callResult,
		}, output, nil
}

func marshalBridgeStepInput(step BridgeStep) json.RawMessage {
	raw, err := json.Marshal(map[string]any{
		"id":        step.ID,
		"condition": step.Condition,
		"action":    step.Action,
	})
	if err != nil {
		return nil
	}
	return raw
}

func insertBridgeRunStep(ctx context.Context, rc *RunContext, workspaceID, runID string, input json.RawMessage) (string, error) {
	if rc == nil || rc.DB == nil {
		return "", nil
	}
	tx, err := rc.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }()

	index, err := nextRunStepIndexTx(ctx, tx, workspaceID, runID)
	if err != nil {
		return "", err
	}
	stepID := uuid.NewV7().String()
	now := time.Now().UTC()
	insertErr := insertRunStepTx(ctx, tx, &RunStep{
		ID:          stepID,
		WorkspaceID: workspaceID,
		RunID:       runID,
		StepIndex:   index,
		StepType:    StepTypeBridgeStep,
		Status:      StepStatusRunning,
		Attempt:     1,
		Input:       input,
		StartedAt:   timePtr(now),
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if insertErr != nil {
		return "", insertErr
	}
	if commitErr := tx.Commit(); commitErr != nil {
		return "", commitErr
	}
	return stepID, nil
}

func updateBridgeRunStep(ctx context.Context, rc *RunContext, workspaceID, stepID, status string, output json.RawMessage, stepErr error) error {
	if rc == nil || rc.DB == nil || stepID == "" {
		return nil
	}
	tx, err := rc.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var errText *string
	if stepErr != nil {
		msg := stepErr.Error()
		errText = &msg
	}
	updateErr := updateRunStepStateTx(ctx, tx, stepID, workspaceID, status, nil, output, errText)
	if updateErr != nil {
		return updateErr
	}
	return tx.Commit()
}

func actorIDFromInput(input TriggerAgentInput, evalCtx map[string]any) string {
	if input.TriggeredBy != nil && strings.TrimSpace(*input.TriggeredBy) != "" {
		return strings.TrimSpace(*input.TriggeredBy)
	}
	if value := resolveBridgeValue(evalCtx, "user_id"); value != nil {
		return fmt.Sprint(value)
	}
	if value := resolveBridgeValue(evalCtx, "owner_id"); value != nil {
		return fmt.Sprint(value)
	}
	return "system"
}

func stringPtrOrNil(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	trimmed := strings.TrimSpace(value)
	return &trimmed
}

func (r *SkillRunner) loadBridgeWorkflow(ctx context.Context, input TriggerAgentInput) (*BridgeWorkflow, string, map[string]any, error) {
	wf, envelopeCtx, err := decodeBridgeWorkflowInput(input.Inputs)
	if err == nil {
		return wf, "input", mergeBridgeContexts(input.TriggerContext, envelopeCtx), nil
	}
	if isHardDecodeError(err, input.Inputs) {
		return nil, "", nil, err
	}
	return r.loadFromSkillDefinition(ctx, input)
}

func isHardDecodeError(err error, raw json.RawMessage) bool {
	if len(raw) == 0 || !json.Valid(raw) ||
		errors.Is(err, ErrSkillDefinitionNotFound) || errors.Is(err, ErrBridgeWorkflowInvalid) {
		return false
	}
	var syntaxErr *json.SyntaxError
	return !errors.As(err, &syntaxErr)
}

func (r *SkillRunner) loadFromSkillDefinition(ctx context.Context, input TriggerAgentInput) (*BridgeWorkflow, string, map[string]any, error) {
	wf, source, err := r.loadActiveSkillDefinition(ctx, input.WorkspaceID, input.AgentID)
	if err != nil {
		return nil, "", nil, err
	}
	return wf, source, mergeBridgeContexts(input.TriggerContext, nil), nil
}

func (r *SkillRunner) loadActiveSkillDefinition(ctx context.Context, workspaceID, agentID string) (*BridgeWorkflow, string, error) {
	def, err := r.querySkillDefinition(ctx, workspaceID, agentID)
	if err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(def.Status) != "active" {
		return nil, "", ErrSkillDefinitionInactive
	}
	wf, err := buildBridgeWorkflowFromDefinition(def)
	if err != nil {
		return nil, "", err
	}
	return wf, "skill_definition", nil
}

func (r *SkillRunner) querySkillDefinition(ctx context.Context, workspaceID, agentID string) (*SkillDefinition, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, name, description, steps, agent_definition_id, status, created_at, updated_at
		FROM skill_definition
		WHERE workspace_id = ? AND agent_definition_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, workspaceID, agentID)
	return scanSkillDefinitionRow(row)
}

func scanSkillDefinitionRow(row *sql.Row) (*SkillDefinition, error) {
	var def SkillDefinition
	var description, agentDefinitionID, steps sql.NullString
	if err := row.Scan(
		&def.ID, &def.WorkspaceID, &def.Name,
		&description, &steps, &agentDefinitionID,
		&def.Status, &def.CreatedAt, &def.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSkillDefinitionNotFound
		}
		return nil, err
	}
	if description.Valid {
		def.Description = &description.String
	}
	if agentDefinitionID.Valid {
		def.DefinitionID = &agentDefinitionID.String
	}
	if steps.Valid {
		def.Steps = json.RawMessage(steps.String)
	}
	return &def, nil
}

func buildBridgeWorkflowFromDefinition(def *SkillDefinition) (*BridgeWorkflow, error) {
	wf := &BridgeWorkflow{
		Name:    def.Name,
		Trigger: BridgeTrigger{Event: TriggerTypeManual},
	}
	if err := json.Unmarshal(def.Steps, &wf.Steps); err != nil {
		return nil, fmt.Errorf("decode skill_definition steps: %w", err)
	}
	if err := wf.Validate(); err != nil {
		return nil, err
	}
	return wf, nil
}

type bridgeWorkflowEnvelope struct {
	BridgeWorkflow *BridgeWorkflow `json:"bridge_workflow"`
	Context        map[string]any  `json:"context"`
}

func decodeBridgeWorkflowInput(raw json.RawMessage) (*BridgeWorkflow, map[string]any, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" || trimmed == emptyJSONObject {
		return nil, nil, ErrSkillDefinitionNotFound
	}
	if wf, err := decodeDirectWorkflow(raw); err == nil {
		return wf, nil, nil
	}
	return decodeEnvelopeWorkflow(raw)
}

func decodeDirectWorkflow(raw json.RawMessage) (*BridgeWorkflow, error) {
	var wf BridgeWorkflow
	if err := json.Unmarshal(raw, &wf); err != nil || wf.Name == "" {
		return nil, ErrSkillDefinitionNotFound
	}
	if err := wf.Validate(); err != nil {
		return nil, err
	}
	return &wf, nil
}

func decodeEnvelopeWorkflow(raw json.RawMessage) (*BridgeWorkflow, map[string]any, error) {
	var envelope bridgeWorkflowEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, nil, err
	}
	if envelope.BridgeWorkflow == nil {
		return nil, nil, ErrSkillDefinitionNotFound
	}
	if err := envelope.BridgeWorkflow.Validate(); err != nil {
		return nil, nil, err
	}
	return envelope.BridgeWorkflow, envelope.Context, nil
}

func mergeBridgeContexts(trigger json.RawMessage, envelope map[string]any) map[string]any {
	ctx := make(map[string]any)
	if len(trigger) > 0 && json.Valid(trigger) {
		var decoded map[string]any
		if err := json.Unmarshal(trigger, &decoded); err == nil {
			for k, v := range decoded {
				ctx[k] = v
			}
		}
	}
	for k, v := range envelope {
		ctx[k] = v
	}
	return ctx
}

func evaluateBridgeCondition(condition BridgeCondition, evalCtx map[string]any) (bool, error) {
	left := resolveBridgeValue(evalCtx, condition.Left)
	right := condition.Right
	operator := strings.ToUpper(strings.TrimSpace(condition.Operator))

	switch operator {
	case BridgeOpEQ:
		return compareEquality(left, right), nil
	case BridgeOpNEQ:
		return !compareEquality(left, right), nil
	case BridgeOpGT, BridgeOpLT, BridgeOpGTE, BridgeOpLTE:
		return evaluateOrderedOp(operator, left, right, condition.Left)
	case BridgeOpIn:
		return evaluateInOp(left, right)
	default:
		return false, fmt.Errorf("%w: unsupported condition operator", ErrSkillStepExecutionFailed)
	}
}

func evaluateOrderedOp(operator string, left, right any, fieldName string) (bool, error) {
	lv, lok := toFloat64(left)
	rv, rok := toFloat64(right)
	if !lok || !rok {
		return false, fmt.Errorf("%w: condition %s requires numeric operands", ErrSkillStepExecutionFailed, fieldName)
	}
	switch operator {
	case BridgeOpGT:
		return lv > rv, nil
	case BridgeOpLT:
		return lv < rv, nil
	case BridgeOpGTE:
		return lv >= rv, nil
	default:
		return lv <= rv, nil
	}
}

func evaluateInOp(left, right any) (bool, error) {
	values, ok := right.([]any)
	if !ok {
		return false, fmt.Errorf("%w: IN requires array right operand", ErrSkillStepExecutionFailed)
	}
	for _, item := range values {
		if compareEquality(left, item) {
			return true, nil
		}
	}
	return false, nil
}

func resolveBridgeValue(evalCtx map[string]any, dotted string) any {
	current := any(evalCtx)
	for _, part := range strings.Split(strings.TrimSpace(dotted), ".") {
		obj, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current, ok = obj[part]
		if !ok {
			return nil
		}
	}
	return current
}

func compareEquality(left, right any) bool {
	lv, leftOK := toFloat64(left)
	rv, rightOK := toFloat64(right)
	if leftOK && rightOK {
		return lv == rv
	}
	return fmt.Sprint(left) == fmt.Sprint(right)
}

func toFloat64(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		out, err := v.Float64()
		return out, err == nil
	default:
		return 0, false
	}
}
