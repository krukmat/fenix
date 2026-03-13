package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
	"github.com/matiasleandrokruk/fenix/internal/domain/agent/agents"
	tooldomain "github.com/matiasleandrokruk/fenix/internal/domain/tool"
)

const defaultInsightsShadowAgentID = "insights-shadow-agent"

var ErrInsightsShadowNotConfigured = errors.New("insights shadow mode is not configured")

type insightsShadowExecutor struct {
	runner         *agent.DSLRunner
	orchestrator   *agent.Orchestrator
	toolRegistry   *tooldomain.ToolRegistry
	db             *sql.DB
	defaultAgentID string
}

type insightsShadowExecution struct {
	WrapperRun   *agent.Run
	EffectiveRun *agent.Run
}

func newInsightsShadowExecutor(
	runner *agent.DSLRunner,
	orchestrator *agent.Orchestrator,
	toolRegistry *tooldomain.ToolRegistry,
	db *sql.DB,
) *insightsShadowExecutor {
	if runner == nil || orchestrator == nil {
		return nil
	}
	return &insightsShadowExecutor{
		runner:         runner,
		orchestrator:   orchestrator,
		toolRegistry:   toolRegistry,
		db:             db,
		defaultAgentID: defaultInsightsShadowAgentID,
	}
}

func (h *InsightsAgentHandler) executeInsightsShadow(
	ctx context.Context,
	config agents.InsightsAgentConfig,
	shadowAgentID string,
	primaryRun *agent.Run,
) map[string]any {
	if errResp := ensureInsightsShadowConfigured(h); errResp != nil {
		return errResp
	}
	primaryStored := h.loadPrimaryShadowRun(ctx, config.WorkspaceID, primaryRun)
	execution, err := h.shadow.Execute(ctx, config, shadowAgentID, primaryRun.ID)
	if err != nil {
		return buildInsightsShadowErrorResponse(err, execution)
	}
	return buildInsightsShadowSuccessResponse(primaryStored, execution)
}

func ensureInsightsShadowConfigured(h *InsightsAgentHandler) map[string]any {
	if h == nil || h.shadow == nil {
		return map[string]any{
			"enabled": true,
			"error":   ErrInsightsShadowNotConfigured.Error(),
		}
	}
	return nil
}

func (h *InsightsAgentHandler) loadPrimaryShadowRun(ctx context.Context, workspaceID string, primaryRun *agent.Run) *agent.Run {
	if primaryRun == nil || h == nil || h.shadow == nil || h.shadow.orchestrator == nil {
		return primaryRun
	}
	stored, err := h.shadow.orchestrator.GetAgentRun(ctx, workspaceID, primaryRun.ID)
	if err != nil || stored == nil {
		return primaryRun
	}
	return stored
}

func buildInsightsShadowErrorResponse(err error, execution *insightsShadowExecution) map[string]any {
	resp := map[string]any{
		"enabled": true,
		"error":   err.Error(),
	}
	if execution != nil && execution.WrapperRun != nil {
		resp["run_id"] = execution.WrapperRun.ID
		resp["status"] = execution.WrapperRun.Status
	}
	return resp
}

func buildInsightsShadowSuccessResponse(primaryStored *agent.Run, execution *insightsShadowExecution) map[string]any {
	run := execution.WrapperRun
	effective := coalesceShadowRun(execution.EffectiveRun, run)
	resp := map[string]any{
		"enabled":             true,
		"run_id":              run.ID,
		"status":              effective.Status,
		"agent_definition_id": run.DefinitionID,
	}
	if effective != nil {
		resp["effective_run_id"] = effective.ID
	}
	resp["comparison"] = buildInsightsShadowComparisonFromRuns(primaryStored, run, effective)
	return resp
}

func coalesceShadowRun(effective, fallback *agent.Run) *agent.Run {
	if effective != nil {
		return effective
	}
	return fallback
}

func (e *insightsShadowExecutor) Execute(
	ctx context.Context,
	config agents.InsightsAgentConfig,
	shadowAgentID string,
	primaryRunID string,
) (*insightsShadowExecution, error) {
	triggerContext, err := json.Marshal(map[string]any{
		"shadow_mode":        true,
		"shadow_of_run_id":   primaryRunID,
		"pilot":              "insights",
		"primary_agent_id":   "insights-agent",
		"primary_agent_type": "insights",
		"query":              config.Query,
		"language":           config.Language,
	})
	if err != nil {
		return nil, err
	}
	return e.executeWorkflow(ctx, config, shadowAgentID, triggerContext)
}

func (e *insightsShadowExecutor) ExecutePrimary(
	ctx context.Context,
	config agents.InsightsAgentConfig,
	agentID string,
) (*insightsShadowExecution, error) {
	triggerContext, err := json.Marshal(map[string]any{
		"pilot":            "insights",
		"rollout_mode":     "declarative_primary",
		"primary_path":     "dsl",
		"original_trigger": "insights.trigger",
		"query":            config.Query,
		"language":         config.Language,
	})
	if err != nil {
		return nil, err
	}
	return e.executeWorkflow(ctx, config, agentID, triggerContext)
}

func (e *insightsShadowExecutor) executeWorkflow(
	ctx context.Context,
	config agents.InsightsAgentConfig,
	shadowAgentID string,
	triggerContext json.RawMessage,
) (*insightsShadowExecution, error) {
	if e == nil || e.runner == nil || e.orchestrator == nil {
		return nil, ErrInsightsShadowNotConfigured
	}
	agentID := strings.TrimSpace(shadowAgentID)
	if agentID == "" {
		agentID = e.defaultAgentID
	}
	inputs, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	rc := &agent.RunContext{
		Orchestrator: e.orchestrator,
		ToolRegistry: e.toolRegistry,
		DB:           e.db,
	}
	wrapperRun, err := e.runner.Run(ctx, rc, agent.TriggerAgentInput{
		AgentID:        agentID,
		WorkspaceID:    config.WorkspaceID,
		TriggeredBy:    config.TriggeredByUserID,
		TriggerType:    agent.TriggerTypeManual,
		TriggerContext: triggerContext,
		Inputs:         inputs,
	})
	if err != nil {
		return nil, err
	}
	effectiveRun := e.resolveEffectiveRun(ctx, config.WorkspaceID, wrapperRun)
	return &insightsShadowExecution{
		WrapperRun:   wrapperRun,
		EffectiveRun: effectiveRun,
	}, nil
}

func (e *insightsShadowExecutor) resolveEffectiveRun(ctx context.Context, workspaceID string, wrapperRun *agent.Run) *agent.Run {
	if wrapperRun == nil {
		return nil
	}
	childID := extractChildRunID(wrapperRun.Output)
	if childID == "" {
		return wrapperRun
	}
	childRun, err := e.orchestrator.GetAgentRun(ctx, workspaceID, childID)
	if err != nil || childRun == nil {
		return wrapperRun
	}
	return childRun
}
