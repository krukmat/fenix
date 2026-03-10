package agents

import (
	"context"
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
	})
	if err != nil {
		t.Fatalf("RegisterCurrentGoRunners() error = %v", err)
	}

	cases := []string{
		AgentTypeSupport,
		AgentTypeProspecting,
		AgentTypeKB,
		AgentTypeInsights,
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
	})
	if err != ErrGoAgentNil {
		t.Fatalf("error = %v, want %v", err, ErrGoAgentNil)
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
