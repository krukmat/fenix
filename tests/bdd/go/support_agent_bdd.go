package gobdd

import (
	"fmt"

	"github.com/cucumber/godog"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
)

func registerSupportAgentScenarios(ctx *godog.ScenarioContext, state *scenarioState) {
	ctx.Step(`^a support case has grounded evidence and an allowed resolution path$`, func() error {
		return setupSupportScenario(state, "medium", "service is down", knowledge.ConfidenceHigh, []float64{0.93, 0.87})
	})
	ctx.Step(`^the Support Agent is triggered for the case$`, func() error {
		return triggerSupportAgent(state, "service is down", "medium")
	})
	ctx.Step(`^the support run outcome is completed$`, func() error {
		return expectSupportRunOutcome(state, "completed")
	})
	ctx.Step(`^the support run is recorded in audit and usage$`, func() error {
		return expectSupportAuditAndUsage(state, "agent.support.run.completed")
	})

	ctx.Step(`^a support case only has medium-confidence evidence$`, func() error {
		return setupSupportScenario(state, "medium", "service is unstable", knowledge.ConfidenceMedium, []float64{0.70})
	})
	ctx.Step(`^the Support Agent is triggered for an abstention path$`, func() error {
		return triggerSupportAgent(state, "service is unstable", "medium")
	})
	ctx.Step(`^the support run outcome is abstained$`, func() error {
		return expectSupportRunOutcome(state, "abstained")
	})
	ctx.Step(`^the run explains the lack of decisive evidence$`, func() error {
		run, err := fetchAgentRun(state)
		if err != nil {
			return err
		}
		output, ok := run["output"].(map[string]any)
		if !ok {
			return fmt.Errorf("missing support output payload")
		}
		if got, _ := output["Details"].(string); got == "" {
			return fmt.Errorf("expected abstention details in output")
		}
		return nil
	})

	ctx.Step(`^a high-priority support case has a sensitive but grounded remediation$`, func() error {
		return setupSupportScenario(state, "high", "please apply the remediation", knowledge.ConfidenceHigh, []float64{0.91, 0.86})
	})
	ctx.Step(`^the Support Agent proposes the sensitive action$`, func() error {
		return triggerSupportAgent(state, "please apply the remediation", "high")
	})
	ctx.Step(`^the support run outcome is awaiting approval$`, func() error {
		return expectSupportRunOutcome(state, "awaiting_approval")
	})
	ctx.Step(`^the approval request is available to the operator$`, func() error {
		approvals, err := listApprovals(state)
		if err != nil {
			return err
		}
		if len(approvals) == 0 {
			return fmt.Errorf("expected at least one pending approval")
		}
		state.lastApprovalID, _ = approvals[0]["id"].(string)
		if got, _ := approvals[0]["status"].(string); got != "pending" {
			return fmt.Errorf("approval status = %q, want pending", got)
		}
		return nil
	})

	ctx.Step(`^a high-priority support case lacks grounding for autonomous resolution$`, func() error {
		return setupSupportScenario(state, "high", "critical outage", knowledge.ConfidenceLow, []float64{})
	})
	ctx.Step(`^the Support Agent triggers a human handoff$`, func() error {
		return triggerSupportAgent(state, "critical outage", "high")
	})
	ctx.Step(`^the support run outcome is handed off$`, func() error {
		return expectSupportRunOutcome(state, "handed_off")
	})
	ctx.Step(`^the handoff package preserves case context and evidence$`, func() error {
		data, err := fetchHandoff(state)
		if err != nil {
			return err
		}
		if got, _ := data["caseId"].(string); got == "" {
			return fmt.Errorf("expected caseId in handoff package")
		}
		evidencePack, ok := data["evidencePack"].(map[string]any)
		if !ok {
			return fmt.Errorf("expected handoff evidencePack")
		}
		if evidencePack["schema_version"] != knowledge.EvidencePackSchemaVersion {
			return fmt.Errorf("handoff schema_version = %v, want %s", evidencePack["schema_version"], knowledge.EvidencePackSchemaVersion)
		}
		return nil
	})
}

func setupSupportScenario(state *scenarioState, priority, query string, confidence knowledge.ConfidenceLevel, scores []float64) error {
	runtime, err := ensureBDDRuntime(state)
	if err != nil {
		return err
	}
	runtime.evidence.packs = map[string]*knowledge.EvidencePack{}
	if err := runtime.ensureSupportAgentDefinition(); err != nil {
		return err
	}
	caseID, err := runtime.createSupportCase(priority)
	if err != nil {
		return err
	}
	runtime.evidence.set(query, newBDDEvidencePack(query, confidence, scores...))
	state.lastEntityID = caseID
	return nil
}

func triggerSupportAgent(state *scenarioState, customerQuery, priority string) error {
	runtime, err := ensureBDDRuntime(state)
	if err != nil {
		return err
	}
	status, body, err := runtime.request("POST", "/api/v1/agents/support/trigger", runtime.userID, map[string]any{
		"case_id":        state.lastEntityID,
		"customer_query": customerQuery,
		"priority":       priority,
	})
	if err != nil {
		return err
	}
	state.lastStatusCode = status
	state.lastResponseBody = body
	if status != 201 {
		return fmt.Errorf("support trigger status = %d, want 201", status)
	}
	decoded, err := decodeBDDEnvelope(body)
	if err != nil {
		return err
	}
	data, ok := decoded["data"].(map[string]any)
	if !ok {
		return fmt.Errorf("missing support trigger data")
	}
	runID, _ := data["id"].(string)
	if runID == "" {
		return fmt.Errorf("missing run id in support trigger")
	}
	state.lastRunID = runID
	return nil
}

func expectSupportRunOutcome(state *scenarioState, want string) error {
	run, err := fetchAgentRun(state)
	if err != nil {
		return err
	}
	if got, _ := run["status"].(string); got != want {
		return fmt.Errorf("support run status = %q, want %q", got, want)
	}
	return nil
}

func expectSupportAuditAndUsage(state *scenarioState, action string) error {
	if _, err := fetchAuditEvents(state, action); err != nil {
		return err
	}
	events, err := fetchUsageEvents(state)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return fmt.Errorf("expected usage events for run %s", state.lastRunID)
	}
	return nil
}

func fetchAgentRun(state *scenarioState) (map[string]any, error) {
	runtime, err := ensureBDDRuntime(state)
	if err != nil {
		return nil, err
	}
	status, body, err := runtime.request("GET", "/api/v1/agents/runs/"+state.lastRunID, runtime.userID, nil)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("get run status = %d, want 200", status)
	}
	decoded, err := decodeBDDEnvelope(body)
	if err != nil {
		return nil, err
	}
	data, ok := decoded["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing run data payload")
	}
	return data, nil
}

func listApprovals(state *scenarioState) ([]map[string]any, error) {
	runtime, err := ensureBDDRuntime(state)
	if err != nil {
		return nil, err
	}
	status, body, err := runtime.request("GET", "/api/v1/approvals", runtime.userID, nil)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("list approvals status = %d, want 200", status)
	}
	decoded, err := decodeBDDEnvelope(body)
	if err != nil {
		return nil, err
	}
	items, ok := decoded["data"].([]any)
	if !ok {
		return nil, fmt.Errorf("missing approvals array")
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		row, ok := item.(map[string]any)
		if ok {
			out = append(out, row)
		}
	}
	return out, nil
}

func fetchHandoff(state *scenarioState) (map[string]any, error) {
	runtime, err := ensureBDDRuntime(state)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/api/v1/agents/runs/%s/handoff?case_id=%s", state.lastRunID, state.lastEntityID)
	status, body, err := runtime.request("GET", path, runtime.userID, nil)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("handoff status = %d, want 200", status)
	}
	decoded, err := decodeBDDEnvelope(body)
	if err != nil {
		return nil, err
	}
	data, ok := decoded["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing handoff payload")
	}
	return data, nil
}

func fetchAuditEvents(state *scenarioState, action string) ([]map[string]any, error) {
	runtime, err := ensureBDDRuntime(state)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/api/v1/audit/events?action=%s", action)
	status, body, err := runtime.request("GET", path, runtime.userID, nil)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("audit query status = %d, want 200", status)
	}
	decoded, err := decodeBDDEnvelope(body)
	if err != nil {
		return nil, err
	}
	items, ok := decoded["data"].([]any)
	if !ok {
		return nil, fmt.Errorf("missing audit items")
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		row, ok := item.(map[string]any)
		if ok {
			out = append(out, row)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("expected audit items for action %s", action)
	}
	return out, nil
}

func fetchUsageEvents(state *scenarioState) ([]map[string]any, error) {
	runtime, err := ensureBDDRuntime(state)
	if err != nil {
		return nil, err
	}
	status, body, err := runtime.request("GET", "/api/v1/usage?run_id="+state.lastRunID, runtime.userID, nil)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("usage status = %d, want 200", status)
	}
	decoded, err := decodeBDDEnvelope(body)
	if err != nil {
		return nil, err
	}
	items, ok := decoded["data"].([]any)
	if !ok {
		return nil, fmt.Errorf("missing usage items")
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		row, ok := item.(map[string]any)
		if ok {
			out = append(out, row)
		}
	}
	return out, nil
}
