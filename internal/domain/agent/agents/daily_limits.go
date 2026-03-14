package agents

import (
	"context"
	"database/sql"
)

func checkDailyRunAndCostLimits(
	ctx context.Context,
	db *sql.DB,
	workspaceID string,
	agentDefinitionID string,
	maxRuns int,
	maxCost float64,
	runLimitErr error,
	costLimitErr error,
) error {
	if db == nil {
		return nil
	}

	runsToday, err := dailyRunsForAgent(ctx, db, workspaceID, agentDefinitionID)
	if err != nil {
		return err
	}
	if runsToday >= maxRuns {
		return runLimitErr
	}

	dailyCost, err := dailyCostForAgent(ctx, db, workspaceID, agentDefinitionID)
	if err != nil {
		return err
	}
	if dailyCost >= maxCost {
		return costLimitErr
	}
	return nil
}

func dailyRunsForAgent(ctx context.Context, db *sql.DB, workspaceID, agentDefinitionID string) (int, error) {
	var runsToday int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM agent_run
		WHERE workspace_id = ?
		  AND agent_definition_id = ?
		  AND date(created_at) = date('now')
	`, workspaceID, agentDefinitionID).Scan(&runsToday)
	return runsToday, err
}

func dailyCostForAgent(ctx context.Context, db *sql.DB, workspaceID, agentDefinitionID string) (float64, error) {
	var dailyCost float64
	err := db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(total_cost), 0)
		FROM agent_run
		WHERE workspace_id = ?
		  AND agent_definition_id = ?
		  AND date(created_at) = date('now')
	`, workspaceID, agentDefinitionID).Scan(&dailyCost)
	return dailyCost, err
}
