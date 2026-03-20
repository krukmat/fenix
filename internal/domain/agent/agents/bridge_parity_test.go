package agents

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/domain/policy"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

func TestBridgeWorkflowParityWithGoSupportResolveFlow(t *testing.T) {
	t.Parallel()

	goDB := setupAgentTestDB(t)
	defer goDB.Close()
	goRun, goCaseID, goOwnerID := runGoSupportResolveFlow(t, goDB)

	bridgeDB := setupAgentTestDB(t)
	defer bridgeDB.Close()
	bridgeRun, bridgeCaseID, bridgeOwnerID := runBridgeSupportResolveFlow(t, bridgeDB)

	if goRun.Status != agent.StatusSuccess || bridgeRun.Status != agent.StatusSuccess {
		t.Fatalf("unexpected statuses go=%s bridge=%s", goRun.Status, bridgeRun.Status)
	}

	goCase, err := crm.NewCaseService(goDB).Get(context.Background(), workspaceIDForRun(t, goDB, goRun.ID), goCaseID)
	if err != nil {
		t.Fatalf("load go case: %v", err)
	}
	bridgeCase, err := crm.NewCaseService(bridgeDB).Get(context.Background(), workspaceIDForRun(t, bridgeDB, bridgeRun.ID), bridgeCaseID)
	if err != nil {
		t.Fatalf("load bridge case: %v", err)
	}
	if goCase.Status != "resolved" || bridgeCase.Status != "resolved" {
		t.Fatalf("unexpected case statuses go=%s bridge=%s", goCase.Status, bridgeCase.Status)
	}

	goTools := extractToolNames(t, goRun.ToolCalls)
	bridgeTools := extractToolNames(t, bridgeRun.ToolCalls)
	assertToolParity(t, goTools, []string{tool.BuiltinUpdateCase, tool.BuiltinSendReply})
	assertToolParity(t, bridgeTools, []string{tool.BuiltinUpdateCase, tool.BuiltinSendReply})

	if countNotesForCase(t, goDB, workspaceIDForRun(t, goDB, goRun.ID), goCaseID) != 1 {
		t.Fatalf("expected one reply note for Go support flow")
	}
	if countNotesForCase(t, bridgeDB, workspaceIDForRun(t, bridgeDB, bridgeRun.ID), bridgeCaseID) != 1 {
		t.Fatalf("expected one reply note for bridge flow")
	}

	goSteps, err := agent.NewOrchestrator(goDB).ListRunSteps(context.Background(), workspaceIDForRun(t, goDB, goRun.ID), goRun.ID)
	if err != nil {
		t.Fatalf("go ListRunSteps: %v", err)
	}
	bridgeSteps, err := agent.NewOrchestrator(bridgeDB).ListRunSteps(context.Background(), workspaceIDForRun(t, bridgeDB, bridgeRun.ID), bridgeRun.ID)
	if err != nil {
		t.Fatalf("bridge ListRunSteps: %v", err)
	}

	if len(filterRunStepsByType(goSteps, agent.StepTypeBridgeStep)) != 0 {
		t.Fatalf("go support flow should not emit bridge_step traces")
	}
	if len(filterRunStepsByType(bridgeSteps, agent.StepTypeBridgeStep)) != 2 {
		t.Fatalf("bridge flow should emit 2 bridge_step traces")
	}

	if !hasFinalizeSuccess(goSteps) || !hasFinalizeSuccess(bridgeSteps) {
		t.Fatalf("both flows must end with a successful finalize step")
	}

	if goOwnerID == "" || bridgeOwnerID == "" {
		t.Fatal("owner ids should not be empty")
	}
}

func runGoSupportResolveFlow(t *testing.T, db *sql.DB) (*agent.Run, string, string) {
	t.Helper()

	wsID, ownerID := seedSupportWorkspace(t, db)
	insertSupportAgentDefinition(t, db, wsID)
	caseID := seedSupportCase(t, db, wsID, ownerID, "medium")
	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{
		results: &knowledge.SearchResults{
			Items: []knowledge.SearchResult{{Score: 0.9, Snippet: "restart the service"}},
		},
	})

	run, err := sa.Run(supportRunContext(context.Background(), wsID, ownerID), SupportAgentConfig{
		WorkspaceID:   wsID,
		CaseID:        caseID,
		CustomerQuery: "service is down",
		Priority:      "medium",
	})
	if err != nil {
		t.Fatalf("support Run: %v", err)
	}

	stored, err := agent.NewOrchestrator(db).GetAgentRun(context.Background(), wsID, run.ID)
	if err != nil {
		t.Fatalf("load go run: %v", err)
	}
	return stored, caseID, ownerID
}

func runBridgeSupportResolveFlow(t *testing.T, db *sql.DB) (*agent.Run, string, string) {
	t.Helper()

	wsID, ownerID := seedSupportWorkspace(t, db)
	insertSkillAgentDefinition(t, db, wsID)
	caseID := seedSupportCase(t, db, wsID, ownerID, "medium")
	seedBridgeParityRole(t, db, wsID, ownerID, `{"tools":["update_case","send_reply"]}`)

	registry := tool.NewToolRegistry(db)
	if err := tool.RegisterBuiltInExecutors(registry, tool.BuiltinServices{
		DB:   db,
		Case: crm.NewCaseService(db),
	}); err != nil {
		t.Fatalf("register builtins: %v", err)
	}
	if err := registry.EnsureBuiltInToolDefinitionsForAllWorkspaces(context.Background()); err != nil {
		t.Fatalf("ensure builtins: %v", err)
	}

	orch := agent.NewOrchestratorWithRegistry(db, agent.NewRunnerRegistry())
	runner := agent.NewSkillRunner(db)
	policyEngine := policy.NewPolicyEngine(db, nil, nil)

	run, err := runner.Run(supportRunContext(context.Background(), wsID, ownerID), &agent.RunContext{
		Orchestrator: orch,
		ToolRegistry: registry,
		PolicyEngine: policyEngine,
		DB:           db,
	}, agent.TriggerAgentInput{
		AgentID:      "skill-agent",
		WorkspaceID:  wsID,
		TriggeredBy:  &ownerID,
		TriggerType:  agent.TriggerTypeEvent,
		TriggerContext: mustJSON(map[string]any{
			"case":     map[string]any{"id": caseID, "priority": "medium"},
			"owner_id": ownerID,
		}),
		Inputs: mustJSON(map[string]any{
			"name": "resolve_support_case_parity",
			"trigger": map[string]any{
				"event": "case.created",
			},
			"steps": []map[string]any{
				{
					"id": "step_set_status",
					"action": map[string]any{
						"verb":   "SET",
						"target": "case.status",
						"args":   map[string]any{"value": "resolved"},
					},
				},
				{
					"id": "step_send_reply",
					"action": map[string]any{
						"verb":   "NOTIFY",
						"target": "contact",
						"args":   map[string]any{"message": "We applied a solution and resolved your case."},
					},
				},
			},
		}),
	})
	if err != nil {
		t.Fatalf("bridge Run: %v", err)
	}

	stored, err := agent.NewOrchestrator(db).GetAgentRun(context.Background(), wsID, run.ID)
	if err != nil {
		t.Fatalf("load bridge run: %v", err)
	}
	return stored, caseID, ownerID
}

func insertSkillAgentDefinition(t *testing.T, db *sql.DB, workspaceID string) {
	t.Helper()
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES ('skill-agent', ?, 'Skill Agent', 'skill', 'active')`,
		workspaceID,
	)
	if err != nil {
		t.Fatalf("insert skill agent_definition: %v", err)
	}
}

func seedBridgeParityRole(t *testing.T, db *sql.DB, workspaceID, userID, permissions string) {
	t.Helper()

	now := time.Now().UTC().Format(time.RFC3339)
	roleID := uuid.NewV7().String()
	userRoleID := uuid.NewV7().String()

	if _, err := db.Exec(`
		INSERT INTO role (id, workspace_id, name, permissions, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, roleID, workspaceID, "bridge-parity-role", permissions, now, now); err != nil {
		t.Fatalf("insert role: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO user_role (id, user_id, role_id, created_at)
		VALUES (?, ?, ?, ?)
	`, userRoleID, userID, roleID, now); err != nil {
		t.Fatalf("insert user_role: %v", err)
	}
}

func extractToolNames(t *testing.T, raw json.RawMessage) []string {
	t.Helper()

	var items []map[string]any
	if err := json.Unmarshal(raw, &items); err != nil {
		t.Fatalf("unmarshal tool calls: %v", err)
	}
	names := make([]string, 0, len(items))
	for _, item := range items {
		if name, ok := item["tool_name"].(string); ok {
			names = append(names, name)
		}
	}
	return names
}

func assertToolParity(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("tool count mismatch got=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tool mismatch at %d got=%s want=%s", i, got[i], want[i])
		}
	}
}

func countNotesForCase(t *testing.T, db *sql.DB, workspaceID, caseID string) int {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM note WHERE workspace_id = ? AND entity_type = 'case' AND entity_id = ?`, workspaceID, caseID).Scan(&count); err != nil {
		t.Fatalf("count notes: %v", err)
	}
	return count
}

func workspaceIDForRun(t *testing.T, db *sql.DB, runID string) string {
	t.Helper()
	var workspaceID string
	if err := db.QueryRow(`SELECT workspace_id FROM agent_run WHERE id = ?`, runID).Scan(&workspaceID); err != nil {
		t.Fatalf("load workspace_id for run: %v", err)
	}
	return workspaceID
}

func filterRunStepsByType(steps []*agent.RunStep, stepType string) []*agent.RunStep {
	filtered := make([]*agent.RunStep, 0)
	for _, step := range steps {
		if step.StepType == stepType {
			filtered = append(filtered, step)
		}
	}
	return filtered
}

func hasFinalizeSuccess(steps []*agent.RunStep) bool {
	for _, step := range steps {
		if step.StepType == agent.StepTypeFinalize && step.Status == agent.StepStatusSuccess {
			return true
		}
	}
	return false
}
