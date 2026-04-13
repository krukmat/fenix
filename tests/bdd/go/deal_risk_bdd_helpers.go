package gobdd

import (
	"fmt"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
)

func setupDealRiskScenario(state *scenarioState) error {
	runtime, err := ensureBDDRuntime(state)
	if err != nil {
		return err
	}
	if err := runtime.ensureDealRiskAgentDefinition(); err != nil {
		return err
	}
	dealID, err := runtime.createSalesDeal()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if err := runtime.markDealStale(dealID, now.Add(-45*24*time.Hour), now.Add(-31*24*time.Hour)); err != nil {
		return err
	}
	runtime.evidence.setResults(&knowledge.SearchResults{
		Items: []knowledge.SearchResult{{KnowledgeItemID: "ev-risk-1", Score: 0.92, Snippet: "Deal has been stalled for over 30 days"}},
	})
	state.lastEntityID = dealID
	return nil
}

func triggerDealRiskAgent(state *scenarioState) error {
	runtime, err := ensureBDDRuntime(state)
	if err != nil {
		return err
	}
	status, body, err := runtime.request("POST", "/api/v1/agents/deal-risk/trigger", runtime.userID, map[string]any{
		"deal_id":  state.lastEntityID,
		"language": "es",
	})
	if err != nil {
		return err
	}
	state.lastStatusCode = status
	state.lastResponseBody = body
	if status != 201 {
		return fmt.Errorf("deal risk trigger status = %d, want 201", status)
	}
	decoded, err := decodeBDDEnvelope(body)
	if err != nil {
		return err
	}
	runID, _ := decoded["run_id"].(string)
	if runID == "" {
		return fmt.Errorf("missing run_id in deal risk trigger")
	}
	state.lastRunID = runID
	return nil
}

func expectDealRiskFlagged(state *scenarioState) error {
	run, err := fetchAgentRun(state)
	if err != nil {
		return err
	}
	if got, _ := run["runtime_status"].(string); got != "escalated" {
		return fmt.Errorf("deal risk runtime_status = %q, want %q", got, "escalated")
	}
	output, ok := run["output"].(map[string]any)
	if !ok {
		return fmt.Errorf("missing deal risk output payload")
	}
	signals, ok := output["signals"].(map[string]any)
	if !ok {
		return fmt.Errorf("missing signals payload")
	}
	if got, _ := signals["risk_level"].(string); got != "high" {
		return fmt.Errorf("risk_level = %q, want %q", got, "high")
	}
	if got, _ := output["action"].(string); got != "create_task" {
		return fmt.Errorf("action = %q, want %q", got, "create_task")
	}
	return nil
}

func expectDealRiskEvidence(state *scenarioState) error {
	run, err := fetchAgentRun(state)
	if err != nil {
		return err
	}
	output, ok := run["output"].(map[string]any)
	if !ok {
		return fmt.Errorf("missing deal risk output payload")
	}
	evidence, ok := output["evidence_summary"].([]any)
	if !ok || len(evidence) == 0 {
		return fmt.Errorf("expected grounded evidence summary")
	}
	toolCalls, ok := run["toolCalls"].([]any)
	if !ok || len(toolCalls) == 0 {
		return fmt.Errorf("expected tool calls in run")
	}
	return nil
}
