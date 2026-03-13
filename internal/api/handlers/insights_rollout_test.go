package handlers

import (
	"context"
	"database/sql"
	"testing"
)

func TestParseInsightsRolloutConfig(t *testing.T) {
	t.Parallel()

	cfg := parseInsightsRolloutConfig(map[string]any{
		"agent_spec": map[string]any{
			"pilots": map[string]any{
				insightsPilotKey: map[string]any{
					"enabled":         true,
					"mode":            "declarative",
					"shadow_agent_id": "custom-shadow",
				},
			},
		},
	})
	if !cfg.Enabled || !cfg.DeclarativePrimary {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if cfg.AgentID != "custom-shadow" {
		t.Fatalf("agent id = %q, want custom-shadow", cfg.AgentID)
	}

	cfg = parseInsightsRolloutConfig(map[string]any{
		"agent_spec": map[string]any{
			"pilots": map[string]any{
				insightsPilotKey: map[string]any{
					"enabled": true,
					"mode":    "go",
				},
			},
		},
	})
	if !cfg.Enabled || cfg.DeclarativePrimary {
		t.Fatalf("expected go-primary config, got %+v", cfg)
	}
	if cfg.AgentID != defaultInsightsShadowAgentID {
		t.Fatalf("default agent id = %q, want %q", cfg.AgentID, defaultInsightsShadowAgentID)
	}
}

func TestLoadInsightsRolloutConfig(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	setWorkspaceSettings(t, db, wsID, `{
		"agent_spec": {
			"pilots": {
				"insights": {
					"enabled": true,
					"mode": "declarative"
				}
			}
		}
	}`)

	cfg := loadInsightsRolloutConfig(context.Background(), db, wsID)
	if !cfg.Enabled || !cfg.DeclarativePrimary {
		t.Fatalf("unexpected rollout config: %+v", cfg)
	}

	cfg = loadInsightsRolloutConfig(context.Background(), db, "")
	if cfg.Enabled {
		t.Fatalf("expected empty config for empty workspace, got %+v", cfg)
	}

	cfg = loadInsightsRolloutConfig(context.Background(), (*sql.DB)(nil), wsID)
	if cfg.Enabled {
		t.Fatalf("expected empty config for nil db, got %+v", cfg)
	}
}

func TestExtractChildRunID(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
		"statements": [
			{"output": {"agent_id": "insights-agent", "run_id": "child-run-1"}}
		]
	}`)
	if got := extractChildRunID(raw); got != "child-run-1" {
		t.Fatalf("extractChildRunID = %q, want child-run-1", got)
	}

	if got := extractChildRunID([]byte(`{}`)); got != "" {
		t.Fatalf("extractChildRunID empty payload = %q, want empty", got)
	}
}
