package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
)

const insightsRolloutModeDeclarative = "declarative"
const insightsRolloutModeGo = "go"

type insightsRolloutConfig struct {
	Enabled            bool
	DeclarativePrimary bool
	AgentID            string
	Source             string
}

func loadInsightsRolloutConfig(ctx context.Context, db *sql.DB, workspaceID string) insightsRolloutConfig {
	if db == nil || strings.TrimSpace(workspaceID) == "" {
		return insightsRolloutConfig{}
	}
	var raw sql.NullString
	err := db.QueryRowContext(ctx, `
		SELECT settings
		FROM workspace
		WHERE id = ?
		LIMIT 1
	`, workspaceID).Scan(&raw)
	if err != nil || !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return insightsRolloutConfig{}
	}

	var settings map[string]any
	if jsonErr := json.Unmarshal([]byte(raw.String), &settings); jsonErr != nil {
		return insightsRolloutConfig{}
	}
	return parseInsightsRolloutConfig(settings)
}

func parseInsightsRolloutConfig(settings map[string]any) insightsRolloutConfig {
	pilot := nestedMap(settings, "agent_spec", "pilots", "insights")
	if len(pilot) == 0 {
		return insightsRolloutConfig{}
	}

	_, enabled, isDeclarative, isGo := classifyInsightsRolloutMode(pilot)
	if !enabled && !isDeclarative && !isGo {
		return insightsRolloutConfig{}
	}
	return insightsRolloutConfig{
		Enabled:            true,
		DeclarativePrimary: isDeclarative || (enabled && !isGo),
		AgentID:            rolloutAgentID(pilot),
		Source:             "workspace.settings",
	}
}

func classifyInsightsRolloutMode(pilot map[string]any) (string, bool, bool, bool) {
	mode, _ := pilot["mode"].(string)
	enabled, _ := pilot["enabled"].(bool)
	trimmedMode := strings.TrimSpace(mode)
	return trimmedMode,
		enabled,
		strings.EqualFold(trimmedMode, insightsRolloutModeDeclarative),
		strings.EqualFold(trimmedMode, insightsRolloutModeGo)
}

func rolloutAgentID(pilot map[string]any) string {
	agentID, _ := pilot["shadow_agent_id"].(string)
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return defaultInsightsShadowAgentID
	}
	return agentID
}

func nestedMap(input map[string]any, path ...string) map[string]any {
	current := input
	for _, segment := range path {
		next, _ := current[segment].(map[string]any)
		if len(next) == 0 {
			return nil
		}
		current = next
	}
	return current
}

func buildInsightsRolloutResponse(config insightsRolloutConfig, wrapperRun, effectiveRun *agent.Run) map[string]any {
	mode := "go_primary"
	if config.DeclarativePrimary {
		mode = "declarative_primary"
	}
	resp := map[string]any{
		"enabled":  config.Enabled,
		"selected": config.DeclarativePrimary,
		"mode":     mode,
		"source":   config.Source,
	}
	if wrapperRun != nil {
		resp["agent_definition_id"] = wrapperRun.DefinitionID
	}
	if effectiveRun != nil {
		resp["effective_run_id"] = effectiveRun.ID
		resp["effective_status"] = effectiveRun.Status
	}
	return resp
}
