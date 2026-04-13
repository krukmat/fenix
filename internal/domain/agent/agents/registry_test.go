package agents

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
)

func TestRegisterCurrentGoRunners_RegistersAllAgentTypes(t *testing.T) {
	registry := agent.NewRunnerRegistry()

	err := RegisterCurrentGoRunners(registry, GoAgentRunners{
		Support:     &SupportAgent{},
		Prospecting: &ProspectingAgent{},
		KB:          &KBAgent{},
		Insights:    &InsightsAgent{},
		DealRisk:    &DealRiskAgent{},
	})
	if err != nil {
		t.Fatalf("RegisterCurrentGoRunners() error = %v", err)
	}

	cases := []string{
		AgentTypeSupport,
		AgentTypeProspecting,
		AgentTypeKB,
		AgentTypeInsights,
		AgentTypeDealRisk,
	}
	for _, agentType := range cases {
		runner, ok := registry.Get(agentType)
		if !ok {
			t.Fatalf("Get(%q) ok = false, want true", agentType)
		}
		if runner == nil {
			t.Fatalf("Get(%q) returned nil runner", agentType)
		}
	}
}

func TestRegisterCurrentGoRunners_RejectsNilRegistry(t *testing.T) {
	err := RegisterCurrentGoRunners(nil, GoAgentRunners{
		Support:     &SupportAgent{},
		Prospecting: &ProspectingAgent{},
		KB:          &KBAgent{},
		Insights:    &InsightsAgent{},
		DealRisk:    &DealRiskAgent{},
	})
	if err != ErrRunnerRegistryNil {
		t.Fatalf("error = %v, want %v", err, ErrRunnerRegistryNil)
	}
}

func TestRegisterCurrentGoRunners_RejectsNilAgent(t *testing.T) {
	registry := agent.NewRunnerRegistry()

	err := RegisterCurrentGoRunners(registry, GoAgentRunners{
		Support:     &SupportAgent{},
		Prospecting: nil,
		KB:          &KBAgent{},
		Insights:    &InsightsAgent{},
		DealRisk:    &DealRiskAgent{},
	})
	if err != ErrGoAgentNil {
		t.Fatalf("error = %v, want %v", err, ErrGoAgentNil)
	}
}

func TestRegisterCurrentGoRunners_RejectsNilDealRiskAgent(t *testing.T) {
	registry := agent.NewRunnerRegistry()

	err := RegisterCurrentGoRunners(registry, GoAgentRunners{
		Support:     &SupportAgent{},
		Prospecting: &ProspectingAgent{},
		KB:          &KBAgent{},
		Insights:    &InsightsAgent{},
		DealRisk:    nil,
	})
	if err != ErrGoAgentNil {
		t.Fatalf("error = %v, want %v", err, ErrGoAgentNil)
	}
}

func TestRegisterDSLRunner_RegistersDSLType(t *testing.T) {
	registry := agent.NewRunnerRegistry()
	runner := agent.NewDSLRunner(nil)

	if err := RegisterDSLRunner(registry, runner); err != nil {
		t.Fatalf("RegisterDSLRunner() error = %v", err)
	}

	got, ok := registry.Get(AgentTypeDSL)
	if !ok {
		t.Fatalf("Get(%q) ok = false, want true", AgentTypeDSL)
	}
	if got == nil {
		t.Fatalf("Get(%q) returned nil runner", AgentTypeDSL)
	}
}

func TestRegisterDSLRunner_RejectsNilInputs(t *testing.T) {
	if err := RegisterDSLRunner(nil, &agent.DSLRunner{}); err != ErrRunnerRegistryNil {
		t.Fatalf("error = %v, want %v", err, ErrRunnerRegistryNil)
	}

	registry := agent.NewRunnerRegistry()
	if err := RegisterDSLRunner(registry, nil); err != ErrDSLRunnerNil {
		t.Fatalf("error = %v, want %v", err, ErrDSLRunnerNil)
	}
}

func TestRegisterDSLRunner_DoesNotBreakGoRunners(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	wsID, ownerID := seedSupportWorkspace(t, db)
	insertSupportAgentDefinition(t, db, wsID)
	insertDSLSupportAlias(t, db, wsID)
	caseID := seedSupportCase(t, db, wsID, ownerID, "medium")
	insertDSLAgentDefinition(t, db, wsID)
	insertDSLWorkflow(t, db, wsID, caseID)

	search := &mockKnowledgeSearch{
		results: &knowledge.SearchResults{
			Items: []knowledge.SearchResult{{Score: 0.9, Snippet: "restart the service"}},
		},
	}

	registry := agent.NewRunnerRegistry()
	orch := agent.NewOrchestratorWithRegistry(db, registry)
	supportAgent := newTestSupportAgent(t, db, search)
	supportAgent.orchestrator = orch

	if err := RegisterCurrentGoRunners(registry, GoAgentRunners{
		Support:     supportAgent,
		Prospecting: &ProspectingAgent{},
		KB:          &KBAgent{},
		Insights:    &InsightsAgent{},
		DealRisk:    &DealRiskAgent{},
	}); err != nil {
		t.Fatalf("RegisterCurrentGoRunners() error = %v", err)
	}
	if err := RegisterDSLRunner(registry, agent.NewDSLRunner(db)); err != nil {
		t.Fatalf("RegisterDSLRunner() error = %v", err)
	}

	supportInputs := mustJSON(map[string]any{
		"workspace_id":   wsID,
		"case_id":        caseID,
		"customer_query": "service is down",
		"priority":       "medium",
	})
	supportRun, err := orch.ExecuteAgent(supportRunContext(context.Background(), wsID, ownerID), &agent.RunContext{}, agent.TriggerAgentInput{
		AgentID:     "support-agent",
		WorkspaceID: wsID,
		TriggerType: agent.TriggerTypeManual,
		Inputs:      supportInputs,
	})
	if err != nil {
		t.Fatalf("ExecuteAgent(support) error = %v", err)
	}
	storedSupport, err := agent.NewOrchestrator(db).GetAgentRun(context.Background(), wsID, supportRun.ID)
	if err != nil {
		t.Fatalf("GetAgentRun(support) error = %v", err)
	}
	if storedSupport.Status != agent.StatusSuccess {
		t.Fatalf("support status = %s, want %s", storedSupport.Status, agent.StatusSuccess)
	}

	dslRun, err := orch.ExecuteAgent(context.Background(), &agent.RunContext{DB: db}, agent.TriggerAgentInput{
		AgentID:        "dsl-agent",
		WorkspaceID:    wsID,
		TriggerType:    agent.TriggerTypeEvent,
		TriggerContext: json.RawMessage(`{"case":{"id":"case-dsl-1"}}`),
	})
	if err != nil {
		t.Fatalf("ExecuteAgent(dsl) error = %v", err)
	}
	storedDSL, err := agent.NewOrchestrator(db).GetAgentRun(context.Background(), wsID, dslRun.ID)
	if err != nil {
		t.Fatalf("GetAgentRun(dsl) error = %v", err)
	}
	if storedDSL.Status != agent.StatusSuccess {
		t.Fatalf("dsl status = %s, want %s", storedDSL.Status, agent.StatusSuccess)
	}
}

func TestExecuteAgent_WithRegisteredSupportRunner_UsesRealGoAgent(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	wsID, ownerID := seedSupportWorkspace(t, db)
	insertSupportAgentDefinition(t, db, wsID)
	caseID := seedSupportCase(t, db, wsID, ownerID, "medium")

	search := &mockKnowledgeSearch{
		results: &knowledge.SearchResults{
			Items: []knowledge.SearchResult{{Score: 0.9, Snippet: "restart the service"}},
		},
	}

	registry := agent.NewRunnerRegistry()
	orch := agent.NewOrchestratorWithRegistry(db, registry)
	supportAgent := newTestSupportAgent(t, db, search)
	supportAgent.orchestrator = orch

	err := RegisterCurrentGoRunners(registry, GoAgentRunners{
		Support:     supportAgent,
		Prospecting: &ProspectingAgent{},
		KB:          &KBAgent{},
		Insights:    &InsightsAgent{},
		DealRisk:    &DealRiskAgent{},
	})
	if err != nil {
		t.Fatalf("RegisterCurrentGoRunners() error = %v", err)
	}

	inputs := mustJSON(map[string]any{
		"workspace_id":   wsID,
		"case_id":        caseID,
		"customer_query": "service is down",
		"priority":       "medium",
	})

	run, err := orch.ExecuteAgent(supportRunContext(context.Background(), wsID, ownerID), &agent.RunContext{}, agent.TriggerAgentInput{
		AgentID:     "support-agent",
		WorkspaceID: wsID,
		TriggerType: agent.TriggerTypeManual,
		Inputs:      inputs,
	})
	if err != nil {
		t.Fatalf("ExecuteAgent() error = %v", err)
	}

	stored, err := agent.NewOrchestrator(db).GetAgentRun(context.Background(), wsID, run.ID)
	if err != nil {
		t.Fatalf("GetAgentRun() error = %v", err)
	}
	if stored.Status != agent.StatusSuccess {
		t.Fatalf("Status = %q, want %q", stored.Status, agent.StatusSuccess)
	}

	caseTicket, err := crm.NewCaseService(db).Get(context.Background(), wsID, caseID)
	if err != nil {
		t.Fatalf("Get case: %v", err)
	}
	if caseTicket.Status != "resolved" {
		t.Fatalf("Case status = %q, want %q", caseTicket.Status, "resolved")
	}
}

func insertDSLAgentDefinition(t *testing.T, db *sql.DB, wsID string) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
		VALUES ('dsl-agent', ?, 'dsl workflow runner', 'dsl', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, wsID); err != nil {
		t.Fatalf("insert dsl agent definition: %v", err)
	}
}

func insertDSLWorkflow(t *testing.T, db *sql.DB, wsID string, caseID string) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO workflow (id, workspace_id, agent_definition_id, name, dsl_source, version, status, created_at, updated_at)
		VALUES ('dsl-workflow-1', ?, 'dsl-agent', 'resolve_support_case', ?, 1, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, wsID, "WORKFLOW resolve_support_case\nON case.created\nAGENT support_agent WITH {\"workspace_id\":\""+wsID+"\",\"case_id\":\""+caseID+"\",\"customer_query\":\"service is down\",\"priority\":\"medium\"}"); err != nil {
		t.Fatalf("insert dsl workflow: %v", err)
	}
}

func insertDSLSupportAlias(t *testing.T, db *sql.DB, wsID string) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
		VALUES ('support_agent', ?, 'support_agent', 'support', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, wsID); err != nil {
		t.Fatalf("insert dsl support alias: %v", err)
	}
}
