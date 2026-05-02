package eval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func makeRegressionHappyScenario() GoldenScenario {
	return GoldenScenario{
		ID:     "sc-support-001",
		Title:  "Support happy path",
		Domain: "support",
		InputEvent: ScenarioInputEvent{
			Type: "case.created",
		},
		Expected: ScenarioExpected{
			FinalOutcome:      "success",
			RequiredEvidence:  []string{"case:CASE-001", "account:ACC-001"},
			ForbiddenEvidence: []string{"knowledge:FORBIDDEN-SRC"},
			PolicyDecisions: []ExpectedPolicyDecision{
				{Action: "tool:create_task", ExpectedOutcome: "allow"},
				{Action: "tool:add_case_note", ExpectedOutcome: "allow"},
			},
			ToolCalls: []ExpectedToolCall{
				{ToolName: "create_task", Required: true},
				{ToolName: "add_case_note", Required: false},
			},
			ForbiddenToolCalls: []ForbiddenToolCall{
				{ToolName: "send_email", Reason: "requires approval"},
			},
			ApprovalBehavior: &ExpectedApprovalBehavior{
				Required: false,
			},
			AuditEvents: []string{"agent.run.started", "tool.executed"},
			FinalState: map[string]any{
				"case.status": "In Progress",
			},
		},
	}
}

func makeRegressionMatchingTrace() ActualRunTrace {
	trace := ActualRunTrace{
		RunID:        "run-001",
		WorkspaceID:  "ws-001",
		FinalOutcome: "success",
		EvidenceSources: []string{
			"case:CASE-001",
			"account:ACC-001",
		},
		PolicyDecisions: []TracePolicyDecision{
			{Action: "tool:create_task", Outcome: "allow"},
			{Action: "tool:add_case_note", Outcome: "allow"},
		},
		ToolCalls: []TraceToolCall{
			{ToolName: "create_task", Status: "executed"},
			{ToolName: "add_case_note", Status: "executed"},
		},
		AuditEvents: []TraceAuditEvent{
			{ID: "evt-001", Action: "agent.run.started", Outcome: "success", ActorID: "run-001"},
			{ID: "evt-002", Action: "tool.executed", Outcome: "success", ActorID: "run-001"},
		},
		ContractValidation: TraceContractValidation{
			MutatorsTraceable: true,
			PolicysTraceable:  true,
		},
	}
	setRegressionFinalState(&trace, map[string]any{"case.status": "In Progress"})
	return trace
}

func makeRegressionMetricsScenario() GoldenScenario {
	return GoldenScenario{
		ID:     "sc-metrics-001",
		Title:  "Metrics happy path",
		Domain: "support",
		InputEvent: ScenarioInputEvent{
			Type: "case.created",
		},
		Expected: ScenarioExpected{
			FinalOutcome:      "success",
			RequiredEvidence:  []string{"case:CASE-001", "account:ACC-001"},
			ForbiddenEvidence: []string{"knowledge:FORBIDDEN-SRC"},
			PolicyDecisions: []ExpectedPolicyDecision{
				{Action: "tool:create_task", ExpectedOutcome: "allow"},
				{Action: "tool:add_case_note", ExpectedOutcome: "allow"},
			},
			ToolCalls: []ExpectedToolCall{
				{ToolName: "create_task", Required: true},
				{ToolName: "add_case_note", Required: true},
			},
			ForbiddenToolCalls: []ForbiddenToolCall{
				{ToolName: "send_email", Reason: "requires approval"},
			},
			ApprovalBehavior: &ExpectedApprovalBehavior{
				Required:        true,
				ExpectedOutcome: "approved",
			},
			AuditEvents: []string{"agent.run.started", "tool.executed"},
			FinalState: map[string]any{
				"case.status": "In Progress",
				"priority":    "high",
			},
		},
		Thresholds: ScenarioThresholds{
			MaxLatencyMs: 5000,
			MaxToolCalls: 3,
		},
	}
}

func makeRegressionMetricsTrace() ActualRunTrace {
	latency := int64(4500)
	trace := ActualRunTrace{
		RunID:        "run-metrics-001",
		WorkspaceID:  "ws-001",
		FinalOutcome: "success",
		EvidenceSources: []string{
			"case:CASE-001",
			"account:ACC-001",
		},
		PolicyDecisions: []TracePolicyDecision{
			{Action: "tool:create_task", Outcome: "allow"},
			{Action: "tool:add_case_note", Outcome: "allow"},
		},
		ApprovalEvents: []TraceApprovalEvent{
			{ApprovalID: "ap-001", Action: "send_email", Status: "approved"},
		},
		ToolCalls: []TraceToolCall{
			{ToolName: "create_task", Status: "executed"},
			{ToolName: "add_case_note", Status: "executed"},
		},
		AuditEvents: []TraceAuditEvent{
			{ID: "evt-001", Action: "agent.run.started", Outcome: "success", ActorID: "run-metrics-001"},
			{ID: "evt-002", Action: "tool.executed", Outcome: "success", ActorID: "run-metrics-001"},
		},
		ContractValidation: TraceContractValidation{
			MutatorsTraceable: true,
			PolicysTraceable:  true,
		},
		LatencyMs: &latency,
	}
	setRegressionFinalState(&trace, map[string]any{
		"case.status": "In Progress",
		"priority":    "high",
	})
	return trace
}

func makeRegressionHardGateScenario() GoldenScenario {
	return GoldenScenario{
		ID:     "sc-hardgate-001",
		Title:  "Hard gate scenario",
		Domain: "support",
		InputEvent: ScenarioInputEvent{
			Type: "case.created",
		},
		Expected: ScenarioExpected{
			FinalOutcome:      "success",
			RequiredEvidence:  []string{"case:CASE-001"},
			ForbiddenEvidence: []string{"knowledge:FORBIDDEN-SRC"},
			PolicyDecisions: []ExpectedPolicyDecision{
				{Action: "tool:create_task", ExpectedOutcome: "allow"},
			},
			ToolCalls: []ExpectedToolCall{
				{ToolName: "create_task", Required: true},
			},
			ForbiddenToolCalls: []ForbiddenToolCall{
				{ToolName: "send_email", Reason: "customer communication requires approval"},
			},
			AuditEvents: []string{"tool.executed"},
			FinalState: map[string]any{
				"case.status": "In Progress",
			},
		},
		Thresholds: ScenarioThresholds{
			MaxLatencyMs: 5000,
			MaxToolCalls: 3,
			MaxRetries:   1,
		},
	}
}

func makeRegressionHardGateTrace() ActualRunTrace {
	latency := int64(2000)
	trace := ActualRunTrace{
		RunID:        "run-hardgate-001",
		WorkspaceID:  "ws-001",
		FinalOutcome: "success",
		EvidenceSources: []string{
			"case:CASE-001",
		},
		PolicyDecisions: []TracePolicyDecision{
			{Action: "tool:create_task", Outcome: "allow"},
		},
		ApprovalEvents: []TraceApprovalEvent{
			{ApprovalID: "ap-001", Action: "send_email", Status: "approved"},
		},
		ToolCalls: []TraceToolCall{
			{ToolName: "create_task", Status: "executed"},
		},
		AuditEvents: []TraceAuditEvent{
			{ID: "evt-001", Action: "tool.executed", Outcome: "success", ActorID: "run-hardgate-001"},
		},
		ContractValidation: TraceContractValidation{
			MutatorsTraceable: true,
			PolicysTraceable:  true,
		},
		LatencyMs: &latency,
	}
	setRegressionFinalState(&trace, map[string]any{"case.status": "In Progress"})
	return trace
}

func setRegressionFinalState(trace *ActualRunTrace, state map[string]any) {
	raw, _ := json.Marshal(state)
	trace.FinalStateRaw = raw
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func TestRegressionSuiteAllPass(t *testing.T) {
	t.Parallel()

	runner := RegressionRunner{
		Now: fixedRegressionNow,
	}
	report := runner.Run([]RegressionCase{
		{
			Scenario: makeRegressionHappyScenario(),
			Trace:    makeRegressionMatchingTrace(),
		},
		{
			Scenario: makeRegressionMetricsScenario(),
			Trace:    makeRegressionMetricsTrace(),
		},
	})

	if !report.Passed {
		t.Fatalf("expected Passed=true, got report %#v", report)
	}
	if report.Summary.TotalScenarios != 2 {
		t.Fatalf("expected TotalScenarios=2, got %d", report.Summary.TotalScenarios)
	}
	if report.Summary.PassedScenarios != 2 || report.Summary.FailedScenarios != 0 {
		t.Fatalf("unexpected pass/fail counts: %#v", report.Summary)
	}
}

func TestRegressionSuiteOneFailureNonZeroExit(t *testing.T) {
	t.Parallel()

	failingTrace := makeRegressionMatchingTrace()
	failingTrace.FinalOutcome = "failed"

	report := RegressionRunner{Now: fixedRegressionNow}.Run([]RegressionCase{
		{
			Scenario: makeRegressionHappyScenario(),
			Trace:    failingTrace,
		},
	})

	if report.Passed {
		t.Fatal("expected Passed=false when one scenario fails")
	}
	if report.Summary.FailedScenarios != 1 {
		t.Fatalf("expected FailedScenarios=1, got %d", report.Summary.FailedScenarios)
	}
	reasons := report.Summary.FailedMetricsByScenario["sc-support-001"]
	if !containsString(reasons, string(DimFinalOutcome)) {
		t.Fatalf("expected final_outcome failure reason, got %v", reasons)
	}
}

func TestRegressionSuiteHardGateViolationCounted(t *testing.T) {
	t.Parallel()

	trace := makeRegressionHardGateTrace()
	trace.ToolCalls = append(trace.ToolCalls, TraceToolCall{ToolName: "send_email", Status: "executed"})

	report := RegressionRunner{Now: fixedRegressionNow}.Run([]RegressionCase{
		{
			Scenario: makeRegressionHardGateScenario(),
			Trace:    trace,
		},
	})

	if report.Summary.HardGateViolationCount == 0 {
		t.Fatalf("expected hard gate violations to be counted, got %#v", report.Summary)
	}
	if report.Scenarios[0].HardGateAssessment.FinalVerdict != VerdictFailedValidation {
		t.Fatalf("expected FinalVerdict=%q, got %q", VerdictFailedValidation, report.Scenarios[0].HardGateAssessment.FinalVerdict)
	}
}

func TestRegressionSuiteAggregateReportFormat(t *testing.T) {
	t.Parallel()

	report := RegressionRunner{Now: fixedRegressionNow}.Run([]RegressionCase{
		{
			Scenario: makeRegressionHappyScenario(),
			Trace:    makeRegressionMatchingTrace(),
		},
	})

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("json.Marshal(report): %v", err)
	}
	text := string(raw)
	for _, required := range []string{
		`"generated_at"`,
		`"summary"`,
		`"scenarios"`,
		`"scorecard"`,
		`"hard_gate_assessment"`,
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("expected report JSON to contain %s, got %s", required, text)
		}
	}
}

func TestRegressionSuiteBaselineComparisonRegressionDetected(t *testing.T) {
	t.Parallel()

	baseline := RegressionRunner{Now: fixedRegressionNow}.Run([]RegressionCase{
		{
			Scenario: makeRegressionHappyScenario(),
			Trace:    makeRegressionMatchingTrace(),
		},
	})

	regressedTrace := makeRegressionMatchingTrace()
	regressedTrace.FinalOutcome = "failed"

	current := RegressionRunner{Now: fixedRegressionNow}.Run([]RegressionCase{
		{
			Scenario: makeRegressionHappyScenario(),
			Trace:    regressedTrace,
		},
	})

	report := CompareToBaseline(current, baseline.ToBaselineSnapshot())

	if report.Baseline == nil || !report.Baseline.Regressed {
		t.Fatalf("expected regression delta, got %#v", report.Baseline)
	}
	if !containsString(report.Baseline.NewFailures, "sc-support-001") {
		t.Fatalf("expected new failure for sc-support-001, got %#v", report.Baseline)
	}
	if len(report.Baseline.ScoreRegressions) == 0 {
		t.Fatalf("expected score regression, got %#v", report.Baseline)
	}
}

func TestRegressionBaselineRoundTrip(t *testing.T) {
	t.Parallel()

	report := RegressionRunner{Now: fixedRegressionNow}.Run([]RegressionCase{
		{
			Scenario: makeRegressionHappyScenario(),
			Trace:    makeRegressionMatchingTrace(),
		},
	})

	dir := t.TempDir()
	path := filepath.Join(dir, "baseline.json")
	if err := SaveRegressionBaseline(path, report.ToBaselineSnapshot()); err != nil {
		t.Fatalf("SaveRegressionBaseline(): %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q): %v", path, err)
	}
	if !strings.HasSuffix(string(raw), "\n") {
		t.Fatalf("expected saved baseline to end with newline, got %q", string(raw))
	}

	loaded, err := LoadRegressionBaseline(path)
	if err != nil {
		t.Fatalf("LoadRegressionBaseline(): %v", err)
	}
	if loaded.Summary.TotalScenarios != 1 {
		t.Fatalf("expected TotalScenarios=1 after round trip, got %d", loaded.Summary.TotalScenarios)
	}
	if _, ok := loaded.Scenarios["sc-support-001"]; !ok {
		t.Fatalf("expected scenario snapshot for sc-support-001, got %#v", loaded.Scenarios)
	}
}

func TestRegressionFixtureSuite(t *testing.T) {
	t.Parallel()

	report := RegressionRunner{Now: fixedRegressionNow}.Run([]RegressionCase{
		{
			Scenario: makeRegressionHappyScenario(),
			Trace:    makeRegressionMatchingTrace(),
		},
		{
			Scenario: makeRegressionMetricsScenario(),
			Trace:    makeRegressionMetricsTrace(),
		},
	})

	if !report.Passed {
		t.Fatalf("fixture regression suite failed: %#v", report)
	}
}

func fixedRegressionNow() time.Time {
	return time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC)
}
