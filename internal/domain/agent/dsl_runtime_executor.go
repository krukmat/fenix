package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrDSLToolRegistryMissing = errors.New("dsl runtime requires tool registry")
	ErrDSLExecutorMissingDB   = errors.New("dsl runtime requires db")
	ErrDSLAgentLoopDetected   = errors.New("dsl runtime detected circular agent call")
	ErrDSLAgentDepthExceeded  = errors.New("dsl runtime call depth exceeded")
	ErrDSLAgentPolicyDenied   = errors.New("dsl runtime agent dispatch denied by policy")
)

const dslAgentCallDepthLimit = 5

type dslRuntimeExecutor struct {
	rc          *RunContext
	workspaceID string
	actorID     string
	triggerCtx  json.RawMessage

	toolCalls       []ToolCall
	pending         bool
	pendingApproval *skillApprovalResult
}

func newDSLRuntimeExecutor(rc *RunContext, input TriggerAgentInput, evalCtx map[string]any) *dslRuntimeExecutor {
	return &dslRuntimeExecutor{
		rc:          rc,
		workspaceID: input.WorkspaceID,
		actorID:     actorIDFromInput(input, evalCtx),
		triggerCtx:  marshalRuntimeContext(evalCtx),
		toolCalls:   make([]ToolCall, 0),
	}
}

func (e *dslRuntimeExecutor) Execute(ctx context.Context, op *RuntimeOperation, evalCtx map[string]any) (RuntimeExecutionResult, error) {
	switch op.Kind {
	case RuntimeOperationTool:
		return e.executeToolOperation(ctx, op)
	case RuntimeOperationAgent:
		return e.executeAgentOperation(ctx, op, evalCtx)
	default:
		return RuntimeExecutionResult{}, fmt.Errorf("%w: unsupported runtime operation kind %s", ErrDSLRuntimeFailed, op.Kind)
	}
}

func (e *dslRuntimeExecutor) ToolCallsJSON() json.RawMessage {
	return marshalSkillToolCalls(e.toolCalls)
}

func (e *dslRuntimeExecutor) PendingApproval() *skillApprovalResult {
	return e.pendingApproval
}

func (e *dslRuntimeExecutor) IsPending() bool {
	return e.pending
}

func (e *dslRuntimeExecutor) executeToolOperation(ctx context.Context, op *RuntimeOperation) (RuntimeExecutionResult, error) {
	if e.rc == nil || e.rc.ToolRegistry == nil {
		return RuntimeExecutionResult{}, ErrDSLToolRegistryMissing
	}

	params := cloneRuntimeParams(op.Params)
	approvalCfg := parseBridgeApprovalConfig(params)
	delete(params, "approval")

	mapped := mappedBridgeTool{
		name:   op.ToolName,
		params: params,
	}
	if err := checkMappedToolPolicy(ctx, e.rc, e.actorID, mapped.name); err != nil {
		return RuntimeExecutionResult{}, err
	}
	if approvalCfg != nil && approvalCfg.Required {
		call, pendingApproval, output, err := routeToApproval(ctx, e.rc, e.workspaceID, e.actorID, mapped, approvalCfg)
		if call != nil {
			e.toolCalls = append(e.toolCalls, *call)
		}
		if err != nil {
			return RuntimeExecutionResult{}, err
		}
		e.pending = true
		e.pendingApproval = pendingApproval
		return RuntimeExecutionResult{
			Output: decodeRuntimeOutput(output),
			Status: StatusAccepted,
			Stop:   true,
		}, nil
	}

	call, _, output, err := executeRegisteredTool(ctx, e.rc, e.workspaceID, mapped)
	if call != nil {
		e.toolCalls = append(e.toolCalls, *call)
	}
	if err != nil {
		return RuntimeExecutionResult{}, err
	}
	return RuntimeExecutionResult{Output: decodeRuntimeOutput(output)}, nil
}

func (e *dslRuntimeExecutor) executeAgentOperation(ctx context.Context, op *RuntimeOperation, evalCtx map[string]any) (RuntimeExecutionResult, error) {
	if e.rc == nil || e.rc.Orchestrator == nil {
		return RuntimeExecutionResult{}, ErrDSLRunnerMissingOrchestrator
	}
	if e.rc.DB == nil {
		return RuntimeExecutionResult{}, ErrDSLExecutorMissingDB
	}

	target, err := resolveActiveAgentDefinition(ctx, e.rc.DB, e.workspaceID, op.AgentName)
	if err != nil {
		return RuntimeExecutionResult{}, err
	}
	if err := checkMappedAgentPolicy(ctx, e.rc, e.actorID, target); err != nil {
		return RuntimeExecutionResult{}, err
	}
	if e.rc.CallDepth >= dslAgentCallDepthLimit {
		return RuntimeExecutionResult{}, ErrDSLAgentDepthExceeded
	}
	if containsCall(e.rc.CallChain, target.ID) {
		return RuntimeExecutionResult{}, ErrDSLAgentLoopDetected
	}

	rawInputs, err := json.Marshal(op.Params)
	if err != nil {
		return RuntimeExecutionResult{}, err
	}
	var triggeredBy *string
	if strings.TrimSpace(e.actorID) != "" && e.actorID != "system" {
		actorID := e.actorID
		triggeredBy = &actorID
	}

	subRun, err := e.rc.Orchestrator.ExecuteAgent(ctx, e.rc.WithCall(target.ID), TriggerAgentInput{
		AgentID:        target.ID,
		WorkspaceID:    e.workspaceID,
		TriggeredBy:    triggeredBy,
		TriggerType:    TriggerTypeManual,
		TriggerContext: e.triggerCtx,
		Inputs:         rawInputs,
	})
	if err != nil {
		return RuntimeExecutionResult{}, err
	}
	stored, err := e.rc.Orchestrator.GetAgentRun(ctx, e.workspaceID, subRun.ID)
	if err != nil {
		return RuntimeExecutionResult{}, err
	}

	output := map[string]any{
		"agent_id": target.ID,
		"status":   stored.Status,
		"run_id":   stored.ID,
	}
	switch stored.Status {
	case StatusFailed, StatusRejected:
		return RuntimeExecutionResult{}, fmt.Errorf("%w: sub-agent %s returned %s", ErrDSLRuntimeFailed, target.ID, stored.Status)
	case StatusAccepted:
		e.pending = true
		mergePendingApprovalMetadata(output, stored.Output)
		return RuntimeExecutionResult{
			Output: output,
			Status: StatusAccepted,
			Stop:   true,
		}, nil
	case StatusAbstained:
		return RuntimeExecutionResult{
			Output: output,
			Status: StatusAbstained,
		}, nil
	default:
		return RuntimeExecutionResult{Output: output}, nil
	}
}

func resolveActiveAgentDefinition(ctx context.Context, db *sql.DB, workspaceID, agentName string) (*Definition, error) {
	row := db.QueryRowContext(ctx, `
		SELECT id, workspace_id, name, description, agent_type, objective,
		       allowed_tools, limits, trigger_config, policy_set_id,
		       active_prompt_version_id, status, created_at, updated_at
		FROM agent_definition
		WHERE workspace_id = ? AND (id = ? OR name = ?) AND status = 'active'
		LIMIT 1
	`, workspaceID, agentName, agentName)
	def, err := scanAgentDefinition(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAgentNotFound
		}
		return nil, err
	}
	return def, nil
}

func cloneRuntimeParams(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func decodeRuntimeOutput(raw json.RawMessage) any {
	if len(raw) == 0 || !json.Valid(raw) {
		return nil
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return string(raw)
	}
	return decoded
}

func marshalRuntimeContext(evalCtx map[string]any) json.RawMessage {
	if len(evalCtx) == 0 {
		return json.RawMessage(emptyJSONObject)
	}
	raw, err := json.Marshal(evalCtx)
	if err != nil {
		return json.RawMessage(emptyJSONObject)
	}
	return raw
}

func containsCall(chain []string, agentID string) bool {
	for _, current := range chain {
		if strings.TrimSpace(current) == strings.TrimSpace(agentID) {
			return true
		}
	}
	return false
}

func mergePendingApprovalMetadata(output map[string]any, raw json.RawMessage) {
	if len(raw) == 0 || !json.Valid(raw) {
		output["action"] = "pending_approval"
		return
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		output["action"] = "pending_approval"
		return
	}
	if action, ok := decoded["action"]; ok {
		output["action"] = action
	} else {
		output["action"] = "pending_approval"
	}
	if approvalID, ok := decoded["approval_id"]; ok {
		output["approval_id"] = approvalID
	}
}

func checkMappedAgentPolicy(ctx context.Context, rc *RunContext, actorID string, target *Definition) error {
	if rc == nil || rc.PolicyEngine == nil || strings.TrimSpace(actorID) == "" || target == nil {
		return nil
	}
	allowed, err := rc.PolicyEngine.CheckAgentPermission(ctx, actorID, target.ID, target.Name)
	if err != nil {
		return err
	}
	if !allowed {
		return fmt.Errorf("%w: agent %s denied by policy", ErrDSLAgentPolicyDenied, target.ID)
	}
	return nil
}
