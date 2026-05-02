package eval_test

import (
	"encoding/json"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/eval"
)

func TestOutcomeAccuracyMatch(t *testing.T) {
	t.Parallel()

	scenario := makeMetricsScenario()
	trace := makeMetricsTrace()
	setMetricsFinalState(&trace, map[string]any{"case.status": "In Progress", "priority": "high"})

	metrics := eval.ComputeMetrics(scenario, trace, eval.Compare(scenario, trace))

	if metrics.OutcomeAccuracy != 1 {
		t.Fatalf("expected outcome accuracy 1, got %v", metrics.OutcomeAccuracy)
	}
}

func TestOutcomeAccuracyMismatch(t *testing.T) {
	t.Parallel()

	scenario := makeMetricsScenario()
	trace := makeMetricsTrace()
	trace.FinalOutcome = "failed"
	setMetricsFinalState(&trace, map[string]any{"case.status": "In Progress", "priority": "high"})

	metrics := eval.ComputeMetrics(scenario, trace, eval.Compare(scenario, trace))

	if metrics.OutcomeAccuracy != 0 {
		t.Fatalf("expected outcome accuracy 0, got %v", metrics.OutcomeAccuracy)
	}
}

func TestToolCallF1Perfect(t *testing.T) {
	t.Parallel()

	scenario := makeMetricsScenario()
	trace := makeMetricsTrace()
	setMetricsFinalState(&trace, map[string]any{"case.status": "In Progress", "priority": "high"})

	metrics := eval.ComputeMetrics(scenario, trace, eval.Compare(scenario, trace))

	if metrics.ToolCallPrecision != 1 || metrics.ToolCallRecall != 1 || metrics.ToolCallF1 != 1 {
		t.Fatalf("expected perfect tool metrics, got precision=%v recall=%v f1=%v",
			metrics.ToolCallPrecision, metrics.ToolCallRecall, metrics.ToolCallF1)
	}
}

func TestToolCallF1Partial(t *testing.T) {
	t.Parallel()

	scenario := makeMetricsScenario()
	trace := makeMetricsTrace()
	trace.ToolCalls = []eval.TraceToolCall{
		{ToolName: "create_task", Status: "executed"},
		{ToolName: "lookup_account", Status: "executed"},
	}
	setMetricsFinalState(&trace, map[string]any{"case.status": "In Progress", "priority": "high"})

	metrics := eval.ComputeMetrics(scenario, trace, eval.Compare(scenario, trace))

	if metrics.ToolCallPrecision != 0.5 {
		t.Fatalf("expected precision 0.5, got %v", metrics.ToolCallPrecision)
	}
	if metrics.ToolCallRecall != 0.5 {
		t.Fatalf("expected recall 0.5, got %v", metrics.ToolCallRecall)
	}
	if metrics.ToolCallF1 != 0.5 {
		t.Fatalf("expected f1 0.5, got %v", metrics.ToolCallF1)
	}
}

func TestToolCallF1ZeroExpected(t *testing.T) {
	t.Parallel()

	scenario := makeMetricsScenario()
	scenario.Expected.ToolCalls = nil
	trace := makeMetricsTrace()
	trace.ToolCalls = nil
	setMetricsFinalState(&trace, map[string]any{"case.status": "In Progress", "priority": "high"})

	metrics := eval.ComputeMetrics(scenario, trace, eval.Compare(scenario, trace))

	if metrics.ToolCallPrecision != 1 || metrics.ToolCallRecall != 1 || metrics.ToolCallF1 != 1 {
		t.Fatalf("expected zero-expected-tools metrics to default to 1, got precision=%v recall=%v f1=%v",
			metrics.ToolCallPrecision, metrics.ToolCallRecall, metrics.ToolCallF1)
	}
}

func TestForbiddenToolViolations(t *testing.T) {
	t.Parallel()

	scenario := makeMetricsScenario()
	trace := makeMetricsTrace()
	trace.ToolCalls = append(trace.ToolCalls,
		eval.TraceToolCall{ToolName: "send_email", Status: "executed"},
		eval.TraceToolCall{ToolName: "delete_case", Status: "executed"},
	)
	setMetricsFinalState(&trace, map[string]any{"case.status": "In Progress", "priority": "high"})

	metrics := eval.ComputeMetrics(scenario, trace, eval.Compare(scenario, trace))

	if metrics.ForbiddenToolViolations != 2 {
		t.Fatalf("expected 2 forbidden tool violations, got %d", metrics.ForbiddenToolViolations)
	}
}

func TestComputeMetricsCoreFields(t *testing.T) {
	t.Parallel()

	scenario := makeMetricsScenario()
	trace := makeMetricsTrace()
	setMetricsFinalState(&trace, map[string]any{"case.status": "In Progress", "priority": "low"})

	metrics := eval.ComputeMetrics(scenario, trace, eval.Compare(scenario, trace))

	if metrics.PolicyCompliance != 1 {
		t.Fatalf("expected policy compliance 1, got %v", metrics.PolicyCompliance)
	}
	if metrics.ApprovalAccuracy != 1 {
		t.Fatalf("expected approval accuracy 1, got %v", metrics.ApprovalAccuracy)
	}
	if metrics.EvidenceCoverage != 1 {
		t.Fatalf("expected evidence coverage 1, got %v", metrics.EvidenceCoverage)
	}
	if metrics.ForbiddenEvidenceCount != 0 {
		t.Fatalf("expected forbidden evidence count 0, got %d", metrics.ForbiddenEvidenceCount)
	}
	if metrics.StateMutationAccuracy != 0.5 {
		t.Fatalf("expected state mutation accuracy 0.5, got %v", metrics.StateMutationAccuracy)
	}
	if metrics.AuditCompleteness != 1 {
		t.Fatalf("expected audit completeness 1, got %v", metrics.AuditCompleteness)
	}
	if metrics.ContractValidity != 1 {
		t.Fatalf("expected contract validity 1, got %v", metrics.ContractValidity)
	}
	if metrics.AbstentionAccuracy != 1 {
		t.Fatalf("expected abstention accuracy 1, got %v", metrics.AbstentionAccuracy)
	}
	if metrics.LatencyCompliance != 1 {
		t.Fatalf("expected latency compliance 1, got %v", metrics.LatencyCompliance)
	}
	if metrics.ToolBudgetCompliance != 1 {
		t.Fatalf("expected tool budget compliance 1, got %v", metrics.ToolBudgetCompliance)
	}
}

func TestScorecardVerdicts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		score    float64
		expected eval.Verdict
	}{
		{name: "pass", score: 90, expected: eval.VerdictPass},
		{name: "pass_with_warnings", score: 75, expected: eval.VerdictPassWithWarnings},
		{name: "requires_review", score: 60, expected: eval.VerdictRequiresReview},
		{name: "fail", score: 59.99, expected: eval.VerdictFail},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := eval.ComputeVerdict(tc.score); got != tc.expected {
				t.Fatalf("expected verdict %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestScorecardConfigurableWeights(t *testing.T) {
	t.Parallel()

	metrics := eval.Metrics{
		OutcomeAccuracy:       1,
		ToolCallF1:            0.5,
		PolicyCompliance:      1,
		EvidenceCoverage:      1,
		ApprovalAccuracy:      1,
		StateMutationAccuracy: 1,
		AuditCompleteness:     1,
		ContractValidity:      1,
	}
	weights := eval.ScorecardWeights{
		FinalOutcome:        50,
		ToolCorrectness:     50,
		PolicyCompliance:    0,
		EvidenceGrounding:   0,
		ApprovalCorrectness: 0,
		StateMutation:       0,
		AuditCompleteness:   0,
		ContractValidity:    0,
	}

	scorecard := eval.NewScorecard(metrics, weights)

	if scorecard.TotalScore != 75 {
		t.Fatalf("expected configurable weighted score 75, got %v", scorecard.TotalScore)
	}
	if scorecard.Verdict != eval.VerdictPassWithWarnings {
		t.Fatalf("expected verdict %q, got %q", eval.VerdictPassWithWarnings, scorecard.Verdict)
	}
}

func TestDefaultScorecard(t *testing.T) {
	t.Parallel()

	metrics := eval.Metrics{
		OutcomeAccuracy:       1,
		ToolCallF1:            1,
		PolicyCompliance:      1,
		EvidenceCoverage:      1,
		ApprovalAccuracy:      1,
		StateMutationAccuracy: 1,
		AuditCompleteness:     1,
		ContractValidity:      1,
	}

	scorecard := eval.DefaultScorecard(metrics)

	if scorecard.TotalScore != 100 {
		t.Fatalf("expected default scorecard score 100, got %v", scorecard.TotalScore)
	}
	if scorecard.Verdict != eval.VerdictPass {
		t.Fatalf("expected verdict %q, got %q", eval.VerdictPass, scorecard.Verdict)
	}
}

func makeMetricsScenario() eval.GoldenScenario {
	return eval.GoldenScenario{
		ID:     "sc-metrics-001",
		Domain: "support",
		InputEvent: eval.ScenarioInputEvent{
			Type: "case.created",
		},
		Expected: eval.ScenarioExpected{
			FinalOutcome:      "success",
			RequiredEvidence:  []string{"case:CASE-001", "account:ACC-001"},
			ForbiddenEvidence: []string{"knowledge:FORBIDDEN-SRC"},
			PolicyDecisions: []eval.ExpectedPolicyDecision{
				{Action: "tool:create_task", ExpectedOutcome: "allow"},
			},
			ToolCalls: []eval.ExpectedToolCall{
				{ToolName: "create_task", Required: true},
				{ToolName: "add_case_note", Required: true},
			},
			ForbiddenToolCalls: []eval.ForbiddenToolCall{
				{ToolName: "send_email", Reason: "requires approval"},
				{ToolName: "delete_case", Reason: "destructive"},
			},
			ApprovalBehavior: &eval.ExpectedApprovalBehavior{
				Required:        true,
				ExpectedOutcome: "approved",
			},
			AuditEvents: []string{"agent.run.started", "tool.executed"},
			FinalState: map[string]any{
				"case.status": "In Progress",
				"priority":    "high",
			},
			ShouldAbstain: false,
		},
		Thresholds: eval.ScenarioThresholds{
			MaxLatencyMs: 5000,
			MaxToolCalls: 3,
		},
	}
}

func makeMetricsTrace() eval.ActualRunTrace {
	latency := int64(4500)
	return eval.ActualRunTrace{
		RunID:        "run-metrics-001",
		WorkspaceID:  "ws-001",
		FinalOutcome: "success",
		EvidenceSources: []string{
			"case:CASE-001",
			"account:ACC-001",
		},
		PolicyDecisions: []eval.TracePolicyDecision{
			{Action: "tool:create_task", Outcome: "allow"},
		},
		ApprovalEvents: []eval.TraceApprovalEvent{
			{ApprovalID: "ap-001", Action: "send_email", Status: "approved"},
		},
		ToolCalls: []eval.TraceToolCall{
			{ToolName: "create_task", Status: "executed"},
			{ToolName: "add_case_note", Status: "executed"},
		},
		AuditEvents: []eval.TraceAuditEvent{
			{ID: "evt-001", Action: "agent.run.started", Outcome: "success", ActorID: "run-metrics-001"},
			{ID: "evt-002", Action: "tool.executed", Outcome: "success", ActorID: "run-metrics-001"},
		},
		ContractValidation: eval.TraceContractValidation{
			MutatorsTraceable: true,
			PolicysTraceable:  true,
		},
		LatencyMs: &latency,
	}
}

func setMetricsFinalState(trace *eval.ActualRunTrace, state map[string]any) {
	raw, _ := json.Marshal(state)
	trace.FinalStateRaw = raw
}
