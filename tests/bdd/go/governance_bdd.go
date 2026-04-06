package gobdd

import (
	"fmt"

	"github.com/cucumber/godog"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
)

func registerGovernanceScenarios(ctx *godog.ScenarioContext, state *scenarioState) {
	ctx.Step(`^a completed governed support run exists$`, func() error {
		if err := setupSupportScenario(state, "medium", "inspect this run", knowledge.ConfidenceHigh, []float64{0.93, 0.89}); err != nil {
			return err
		}
		return triggerSupportAgent(state, "inspect this run", "medium")
	})
	ctx.Step(`^the governance operator inspects the run$`, func() error {
		_, err := fetchAgentRun(state)
		return err
	})
	ctx.Step(`^the operator can see the run outcome and runtime trace$`, func() error {
		run, err := fetchAgentRun(state)
		if err != nil {
			return err
		}
		if _, ok := run["runtime_status"].(string); !ok {
			return fmt.Errorf("expected runtime_status in run payload")
		}
		if _, ok := run["reasoningTrace"]; !ok {
			return fmt.Errorf("expected reasoningTrace in run payload")
		}
		return nil
	})
	ctx.Step(`^the audit trail shows actor, action, and timestamp$`, func() error {
		events, err := fetchAuditEvents(state, "agent.support.run.completed")
		if err != nil {
			return err
		}
		event := events[0]
		if event["actor_id"] == nil || event["action"] == nil || event["created_at"] == nil {
			return fmt.Errorf("audit event missing actor/action/timestamp: %#v", event)
		}
		return nil
	})

	ctx.Step(`^a governed support run is awaiting approval$`, func() error {
		if err := setupSupportScenario(state, "high", "needs sensitive action", knowledge.ConfidenceHigh, []float64{0.95, 0.87}); err != nil {
			return err
		}
		if err := triggerSupportAgent(state, "needs sensitive action", "high"); err != nil {
			return err
		}
		return expectSupportRunOutcome(state, "awaiting_approval")
	})
	ctx.Step(`^the governance operator lists pending approvals$`, func() error {
		approvals, err := listApprovals(state)
		if err != nil {
			return err
		}
		if len(approvals) == 0 {
			return fmt.Errorf("expected pending approvals")
		}
		state.lastApprovalID, _ = approvals[0]["id"].(string)
		return nil
	})
	ctx.Step(`^the governance operator can approve the request$`, func() error {
		return decideApproval(state, "approved")
	})
	ctx.Step(`^the approval decision is accepted$`, func() error {
		return expectApprovalAudit(state, "approval.approved")
	})

	ctx.Step(`^a governance rejection decision has been applied to a pending approval$`, func() error {
		if err := setupSupportScenario(state, "high", "reject this remediation", knowledge.ConfidenceHigh, []float64{0.90, 0.86}); err != nil {
			return err
		}
		if err := triggerSupportAgent(state, "reject this remediation", "high"); err != nil {
			return err
		}
		approvals, err := listApprovals(state)
		if err != nil {
			return err
		}
		if len(approvals) == 0 {
			return fmt.Errorf("expected pending approval before rejection")
		}
		state.lastApprovalID, _ = approvals[0]["id"].(string)
		return decideApproval(state, "rejected")
	})
	ctx.Step(`^the governance operator inspects the audit trail for the rejection$`, func() error {
		_, err := fetchAuditEvents(state, "approval.rejected")
		return err
	})
	ctx.Step(`^the rejection is recorded in the audit trail$`, func() error {
		return expectApprovalAudit(state, "approval.rejected")
	})

	ctx.Step(`^a governed run has emitted usage and a quota state exists$`, func() error {
		if err := setupSupportScenario(state, "medium", "quota inspection", knowledge.ConfidenceHigh, []float64{0.92}); err != nil {
			return err
		}
		if err := triggerSupportAgent(state, "quota inspection", "medium"); err != nil {
			return err
		}
		runtime, err := ensureBDDRuntime(state)
		if err != nil {
			return err
		}
		policyID, err := runtime.recordQuotaState()
		if err != nil {
			return err
		}
		state.lastApprovalID = policyID
		return nil
	})
	ctx.Step(`^the governance operator inspects usage and quota state$`, func() error {
		if _, err := fetchUsageEvents(state); err != nil {
			return err
		}
		_, err := fetchQuotaState(state)
		return err
	})
	ctx.Step(`^the operator can see usage events for the run$`, func() error {
		events, err := fetchUsageEvents(state)
		if err != nil {
			return err
		}
		if len(events) == 0 {
			return fmt.Errorf("expected usage events")
		}
		return nil
	})
	ctx.Step(`^the operator can see the current quota state$`, func() error {
		data, err := fetchQuotaState(state)
		if err != nil {
			return err
		}
		if got, _ := data["quotaPolicyId"].(string); got == "" {
			return fmt.Errorf("expected quotaPolicyId in quota state")
		}
		return nil
	})
}

func decideApproval(state *scenarioState, decision string) error {
	runtime, err := ensureBDDRuntime(state)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/api/v1/approvals/%s", state.lastApprovalID)
	status, _, err := runtime.request("PUT", path, runtime.userID, map[string]any{"decision": decision})
	if err != nil {
		return err
	}
	if status != 204 {
		return fmt.Errorf("approval decision status = %d, want 204", status)
	}
	return nil
}

func expectApprovalAudit(state *scenarioState, action string) error {
	events, err := fetchAuditEvents(state, action)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return fmt.Errorf("expected audit events for %s", action)
	}
	return nil
}

func fetchQuotaState(state *scenarioState) (map[string]any, error) {
	runtime, err := ensureBDDRuntime(state)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/api/v1/quota-state?quota_policy_id=%s", state.lastApprovalID)
	status, body, err := runtime.request("GET", path, runtime.userID, nil)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("quota-state status = %d, want 200", status)
	}
	decoded, err := decodeBDDEnvelope(body)
	if err != nil {
		return nil, err
	}
	data, ok := decoded["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing quota-state payload")
	}
	return data, nil
}
