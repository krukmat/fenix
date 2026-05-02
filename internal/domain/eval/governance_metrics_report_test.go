package eval

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestBuildGovernanceMetricsReportAggregatesRunsAndBreakdowns(t *testing.T) {
	t.Parallel()

	latencyA := int64(480)
	latencyB := int64(1200)
	costA := 0.007
	costB := 0.011
	now := time.Date(2026, 5, 2, 13, 0, 0, 0, time.UTC)

	report := BuildGovernanceMetricsReport(now, []GovernanceReviewCase{
		{
			Trace: ActualRunTrace{
				RunID:        "run-1",
				ScenarioID:   "sc-support-004",
				TriggerType:  "case.created",
				FinalOutcome: "awaiting_approval",
				LatencyMs:    &latencyA,
				TotalCost:    &costA,
				Retries:      1,
				InputEvent:   json.RawMessage(`{"workflow_id":"wf-support"}`),
				ToolCalls:    []TraceToolCall{{ToolName: "request_approval"}, {ToolName: "update_case"}},
				PolicyDecisions: []TracePolicyDecision{
					{Action: "tool:update_case", Outcome: "require_approval"},
				},
				ApprovalEvents: []TraceApprovalEvent{{ApprovalID: "apr-1", Action: "support.case.update", Status: "pending"}},
				AuditEvents:    []TraceAuditEvent{{ActorID: "run-1"}},
			},
			Result: RegressionScenarioResult{
				ScenarioID: "sc-support-004",
				Passed:     true,
				HardGateAssessment: HardGateAssessment{
					FinalVerdict: VerdictPass,
				},
			},
		},
		{
			Trace: ActualRunTrace{
				RunID:        "run-2",
				ScenarioID:   "sc-support-002",
				TriggerType:  "case.created",
				FinalOutcome: "blocked",
				LatencyMs:    &latencyB,
				TotalCost:    &costB,
				Output:       json.RawMessage(`{"workflow_id":"wf-support"}`),
				ToolCalls:    []TraceToolCall{{ToolName: "send_email", Status: "blocked"}},
				PolicyDecisions: []TracePolicyDecision{
					{Action: "tool:send_email", Outcome: "deny"},
				},
				AuditEvents: []TraceAuditEvent{{ActorID: "run-2"}},
			},
			Result: RegressionScenarioResult{
				ScenarioID:         "sc-support-002",
				Passed:             false,
				HardGateViolations: []HardGateViolation{{Gate: "policy_denial_bypass"}},
				HardGateAssessment: HardGateAssessment{
					FinalVerdict: VerdictFailedValidation,
				},
			},
		},
	})

	if report.GeneratedAt != now {
		t.Fatalf("GeneratedAt = %s, want %s", report.GeneratedAt, now)
	}
	if report.Summary.TotalRuns != 2 {
		t.Fatalf("TotalRuns = %d, want 2", report.Summary.TotalRuns)
	}
	if report.Summary.PassedRuns != 1 || report.Summary.FailedRuns != 1 {
		t.Fatalf("pass/fail summary = %#v", report.Summary)
	}
	if report.Summary.PolicyDenialCount != 1 {
		t.Fatalf("PolicyDenialCount = %d, want 1", report.Summary.PolicyDenialCount)
	}
	if report.Summary.HardGateViolationCount != 1 {
		t.Fatalf("HardGateViolationCount = %d, want 1", report.Summary.HardGateViolationCount)
	}
	if report.Summary.ApprovalRequestCount != 1 {
		t.Fatalf("ApprovalRequestCount = %d, want 1", report.Summary.ApprovalRequestCount)
	}
	if report.Summary.TotalToolCalls != 3 {
		t.Fatalf("TotalToolCalls = %d, want 3", report.Summary.TotalToolCalls)
	}
	if report.Summary.PassRate != 0.5 {
		t.Fatalf("PassRate = %.2f, want 0.50", report.Summary.PassRate)
	}
	if report.Summary.AverageLatencyMs != 840 {
		t.Fatalf("AverageLatencyMs = %.2f, want 840", report.Summary.AverageLatencyMs)
	}
	if report.Summary.TotalCost != 0.018 {
		t.Fatalf("TotalCost = %.4f, want 0.0180", report.Summary.TotalCost)
	}
	if len(report.ByWorkflow) != 1 || report.ByWorkflow[0].Key != "wf-support" {
		t.Fatalf("ByWorkflow = %#v, want wf-support aggregate", report.ByWorkflow)
	}
	if len(report.ByTool) != 3 {
		t.Fatalf("ByTool len = %d, want 3 distinct tool rows", len(report.ByTool))
	}
	if report.Runs[0].WorkflowID != "wf-support" {
		t.Fatalf("Runs[0].WorkflowID = %q, want wf-support", report.Runs[0].WorkflowID)
	}
}

func TestGovernanceMetricsReportToMarkdownIncludesInspectableSections(t *testing.T) {
	t.Parallel()

	report := GovernanceMetricsReport{
		GeneratedAt: time.Date(2026, 5, 2, 13, 0, 0, 0, time.UTC),
		Summary: GovernanceMetricsSummary{
			TotalRuns:              1,
			PassedRuns:             1,
			PassRate:               1,
			PolicyDenialCount:      0,
			HardGateViolationCount: 0,
		},
		Runs: []GovernanceRunMetric{{
			RunID:        "run-1",
			ScenarioID:   "sc-support-004",
			WorkflowID:   "wf-support",
			ActorID:      "run-1",
			FinalOutcome: "awaiting_approval",
			FinalVerdict: VerdictPass,
		}},
		ByWorkflow: []GovernanceDimensionMetric{{Key: "wf-support", Runs: 1, PassedRuns: 1, PassRate: 1}},
		ByTool:     []GovernanceToolMetric{{ToolName: "request_approval", Calls: 1}},
	}

	md := report.ToMarkdown()
	if !strings.Contains(md, "# Governance Metrics Report") {
		t.Fatalf("markdown missing title: %q", md)
	}
	if !strings.Contains(md, "## Runs") {
		t.Fatalf("markdown missing runs section: %q", md)
	}
	if !strings.Contains(md, "## By Workflow") {
		t.Fatalf("markdown missing workflow section: %q", md)
	}
	if !strings.Contains(md, "## By Tool") {
		t.Fatalf("markdown missing tool section: %q", md)
	}
}
