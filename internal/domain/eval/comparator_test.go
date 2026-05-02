package eval_test

// F3-T1: Deterministic trace comparator — TDD tests.
// Tests cover: exact match, missing tool, extra tool, forbidden tool,
// final state mismatch, missing audit event, forbidden evidence.

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/eval"
)

// --- helpers ---

func makeHappyScenario() eval.GoldenScenario {
	return eval.GoldenScenario{
		ID:     "sc-test-001",
		Domain: "support",
		InputEvent: eval.ScenarioInputEvent{
			Type: "case.created",
		},
		Expected: eval.ScenarioExpected{
			FinalOutcome: "success",
			RequiredEvidence: []string{
				"case:CASE-001",
				"account:ACC-001",
			},
			ForbiddenEvidence: []string{"knowledge:FORBIDDEN-SRC"},
			PolicyDecisions: []eval.ExpectedPolicyDecision{
				{Action: "tool:create_task", ExpectedOutcome: "allow"},
			},
			ToolCalls: []eval.ExpectedToolCall{
				{ToolName: "create_task", Required: true},
				{ToolName: "add_case_note", Required: false},
			},
			ForbiddenToolCalls: []eval.ForbiddenToolCall{
				{ToolName: "send_email", Reason: "requires approval"},
			},
			ApprovalBehavior: &eval.ExpectedApprovalBehavior{
				Required:        false,
				ExpectedOutcome: "",
			},
			AuditEvents: []string{
				"agent.run.started",
				"tool.executed",
			},
			FinalState: map[string]any{
				"case.status": "In Progress",
			},
			ShouldAbstain: false,
		},
	}
}

func makeMatchingTrace() eval.ActualRunTrace {
	return eval.ActualRunTrace{
		RunID:        "run-001",
		WorkspaceID:  "ws-001",
		FinalOutcome: "success",
		EvidenceSources: []string{
			"case:CASE-001",
			"account:ACC-001",
		},
		PolicyDecisions: []eval.TracePolicyDecision{
			{Action: "tool:create_task", Outcome: "allow"},
		},
		ToolCalls: []eval.TraceToolCall{
			{ToolName: "create_task", Status: "executed"},
			{ToolName: "add_case_note", Status: "executed"},
		},
		AuditEvents: []eval.TraceAuditEvent{
			{ID: "evt-001", Action: "agent.run.started", Outcome: "success", ActorID: "run-001"},
			{ID: "evt-002", Action: "tool.executed", Outcome: "success", ActorID: "run-001"},
		},
		ContractValidation: eval.TraceContractValidation{
			MutatorsTraceable: true,
			PolicysTraceable:  true,
		},
	}
}

func setFinalState(trace *eval.ActualRunTrace, state map[string]any) {
	raw, _ := json.Marshal(state)
	trace.FinalStateRaw = raw
}

// --- tests ---

// TestComparatorExactMatch — all dimensions match, result must be clean.
func TestComparatorExactMatch(t *testing.T) {
	t.Parallel()

	scenario := makeHappyScenario()
	trace := makeMatchingTrace()
	setFinalState(&trace, map[string]any{"case.status": "In Progress"})

	result := eval.Compare(scenario, trace)

	if !result.Pass {
		t.Errorf("expected Pass=true, got mismatches: %v", result.Mismatches)
	}
	if len(result.Mismatches) != 0 {
		t.Errorf("expected 0 mismatches, got %d: %v", len(result.Mismatches), result.Mismatches)
	}
}

// TestComparatorMissingToolCall — required tool absent from actual trace.
func TestComparatorMissingToolCall(t *testing.T) {
	t.Parallel()

	scenario := makeHappyScenario()
	trace := makeMatchingTrace()
	setFinalState(&trace, map[string]any{"case.status": "In Progress"})

	// Remove the required create_task tool call.
	trace.ToolCalls = []eval.TraceToolCall{
		{ToolName: "add_case_note", Status: "executed"},
	}

	result := eval.Compare(scenario, trace)

	if result.Pass {
		t.Error("expected Pass=false when required tool call is missing")
	}
	if !hasMismatchDimension(result, eval.DimToolCalls) {
		t.Errorf("expected mismatch in dimension %q", eval.DimToolCalls)
	}
	mismatch := findMismatch(result, eval.DimToolCalls)
	if !strings.Contains(mismatch.Evidence, "create_task") {
		t.Errorf("mismatch evidence must mention the missing tool, got: %q", mismatch.Evidence)
	}
}

// TestComparatorExtraToolCall — actual has a tool not in expected list.
// Extra non-forbidden tool calls are reported as informational (no hard fail).
func TestComparatorExtraToolCall(t *testing.T) {
	t.Parallel()

	scenario := makeHappyScenario()
	trace := makeMatchingTrace()
	setFinalState(&trace, map[string]any{"case.status": "In Progress"})

	// Add an unexpected (but not forbidden) tool call.
	trace.ToolCalls = append(trace.ToolCalls, eval.TraceToolCall{
		ToolName: "lookup_account",
		Status:   "executed",
	})

	result := eval.Compare(scenario, trace)

	if !hasMismatchDimension(result, eval.DimExtraToolCalls) {
		t.Errorf("expected informational mismatch for extra tool call in dimension %q", eval.DimExtraToolCalls)
	}
	if !result.Pass {
		t.Error("expected Pass=true when only an informational extra tool mismatch exists")
	}
	mismatch := findMismatch(result, eval.DimExtraToolCalls)
	if !strings.Contains(mismatch.Evidence, "lookup_account") {
		t.Errorf("evidence must mention the extra tool, got: %q", mismatch.Evidence)
	}
}

// TestComparatorForbiddenToolCall — actual called a tool from the forbidden list.
func TestComparatorForbiddenToolCall(t *testing.T) {
	t.Parallel()

	scenario := makeHappyScenario()
	trace := makeMatchingTrace()
	setFinalState(&trace, map[string]any{"case.status": "In Progress"})

	// Add a forbidden tool call.
	trace.ToolCalls = append(trace.ToolCalls, eval.TraceToolCall{
		ToolName: "send_email",
		Status:   "executed",
	})

	result := eval.Compare(scenario, trace)

	if result.Pass {
		t.Error("expected Pass=false when forbidden tool was called")
	}
	if !hasMismatchDimension(result, eval.DimForbiddenToolCalls) {
		t.Errorf("expected mismatch in dimension %q", eval.DimForbiddenToolCalls)
	}
	mismatch := findMismatch(result, eval.DimForbiddenToolCalls)
	if !strings.Contains(mismatch.Evidence, "send_email") {
		t.Errorf("evidence must name the forbidden tool, got: %q", mismatch.Evidence)
	}
}

// TestComparatorFinalStateMismatch — actual state differs from expected.
func TestComparatorFinalStateMismatch(t *testing.T) {
	t.Parallel()

	scenario := makeHappyScenario()
	trace := makeMatchingTrace()
	setFinalState(&trace, map[string]any{"case.status": "New"}) // wrong value

	result := eval.Compare(scenario, trace)

	if result.Pass {
		t.Error("expected Pass=false when final state mismatches")
	}
	if !hasMismatchDimension(result, eval.DimFinalState) {
		t.Errorf("expected mismatch in dimension %q", eval.DimFinalState)
	}
	mismatch := findMismatch(result, eval.DimFinalState)
	if !strings.Contains(mismatch.Evidence, "case.status") {
		t.Errorf("evidence must name the mismatched field, got: %q", mismatch.Evidence)
	}
}

// TestComparatorMissingAuditEvent — required audit event not present in trace.
func TestComparatorMissingAuditEvent(t *testing.T) {
	t.Parallel()

	scenario := makeHappyScenario()
	trace := makeMatchingTrace()
	setFinalState(&trace, map[string]any{"case.status": "In Progress"})

	// Remove all audit events → both required events are absent.
	trace.AuditEvents = []eval.TraceAuditEvent{}

	result := eval.Compare(scenario, trace)

	if result.Pass {
		t.Error("expected Pass=false when required audit events are missing")
	}
	if !hasMismatchDimension(result, eval.DimAuditEvents) {
		t.Errorf("expected mismatch in dimension %q", eval.DimAuditEvents)
	}
}

// TestComparatorForbiddenEvidence — trace used a source in the forbidden list.
func TestComparatorForbiddenEvidence(t *testing.T) {
	t.Parallel()

	scenario := makeHappyScenario()
	trace := makeMatchingTrace()
	setFinalState(&trace, map[string]any{"case.status": "In Progress"})

	// Add forbidden evidence source.
	trace.EvidenceSources = append(trace.EvidenceSources, "knowledge:FORBIDDEN-SRC")

	result := eval.Compare(scenario, trace)

	if result.Pass {
		t.Error("expected Pass=false when forbidden evidence was used")
	}
	if !hasMismatchDimension(result, eval.DimForbiddenEvidence) {
		t.Errorf("expected mismatch in dimension %q", eval.DimForbiddenEvidence)
	}
	mismatch := findMismatch(result, eval.DimForbiddenEvidence)
	if !strings.Contains(mismatch.Evidence, "FORBIDDEN-SRC") {
		t.Errorf("evidence must name the forbidden source, got: %q", mismatch.Evidence)
	}
}

// TestComparatorFinalOutcomeMismatch — outcome differs from expected.
func TestComparatorFinalOutcomeMismatch(t *testing.T) {
	t.Parallel()

	scenario := makeHappyScenario()
	trace := makeMatchingTrace()
	setFinalState(&trace, map[string]any{"case.status": "In Progress"})

	trace.FinalOutcome = "failed"

	result := eval.Compare(scenario, trace)

	if result.Pass {
		t.Error("expected Pass=false when final outcome mismatches")
	}
	if !hasMismatchDimension(result, eval.DimFinalOutcome) {
		t.Errorf("expected mismatch in dimension %q", eval.DimFinalOutcome)
	}
}

// TestComparatorMissingRequiredEvidence — required source absent from trace.
func TestComparatorMissingRequiredEvidence(t *testing.T) {
	t.Parallel()

	scenario := makeHappyScenario()
	trace := makeMatchingTrace()
	setFinalState(&trace, map[string]any{"case.status": "In Progress"})

	// Remove a required evidence source.
	trace.EvidenceSources = []string{"case:CASE-001"} // account:ACC-001 missing

	result := eval.Compare(scenario, trace)

	if result.Pass {
		t.Error("expected Pass=false when required evidence is missing")
	}
	if !hasMismatchDimension(result, eval.DimRequiredEvidence) {
		t.Errorf("expected mismatch in dimension %q", eval.DimRequiredEvidence)
	}
}

// TestComparatorPolicyDecisionMismatch — policy outcome differs from expected.
func TestComparatorPolicyDecisionMismatch(t *testing.T) {
	t.Parallel()

	scenario := makeHappyScenario()
	trace := makeMatchingTrace()
	setFinalState(&trace, map[string]any{"case.status": "In Progress"})

	// Override policy decision outcome.
	trace.PolicyDecisions = []eval.TracePolicyDecision{
		{Action: "tool:create_task", Outcome: "deny"},
	}

	result := eval.Compare(scenario, trace)

	if result.Pass {
		t.Error("expected Pass=false when policy decision mismatches")
	}
	if !hasMismatchDimension(result, eval.DimPolicyDecisions) {
		t.Errorf("expected mismatch in dimension %q", eval.DimPolicyDecisions)
	}
}

// TestComparatorMissingApprovalEvent — scenario requires approval but trace has none.
func TestComparatorMissingApprovalEvent(t *testing.T) {
	t.Parallel()

	scenario := makeHappyScenario()
	scenario.Expected.ApprovalBehavior = &eval.ExpectedApprovalBehavior{
		Required:        true,
		ExpectedOutcome: "pending",
	}

	trace := makeMatchingTrace()
	setFinalState(&trace, map[string]any{"case.status": "In Progress"})

	result := eval.Compare(scenario, trace)

	if result.Pass {
		t.Error("expected Pass=false when required approval behavior is absent")
	}
	if !hasMismatchDimension(result, eval.DimApprovalBehavior) {
		t.Errorf("expected mismatch in dimension %q", eval.DimApprovalBehavior)
	}
}

// TestComparatorApprovalOutcomeMismatch — approval exists but status differs.
func TestComparatorApprovalOutcomeMismatch(t *testing.T) {
	t.Parallel()

	scenario := makeHappyScenario()
	scenario.Expected.ApprovalBehavior = &eval.ExpectedApprovalBehavior{
		Required:        true,
		ExpectedOutcome: "approved",
	}

	trace := makeMatchingTrace()
	trace.ApprovalEvents = []eval.TraceApprovalEvent{
		{ApprovalID: "ap-001", Action: "send_email", Status: "pending"},
	}
	setFinalState(&trace, map[string]any{"case.status": "In Progress"})

	result := eval.Compare(scenario, trace)

	if result.Pass {
		t.Error("expected Pass=false when approval outcome mismatches")
	}
	if !hasMismatchDimension(result, eval.DimApprovalBehavior) {
		t.Errorf("expected mismatch in dimension %q", eval.DimApprovalBehavior)
	}
}

// TestComparatorUnexpectedApprovalEvent — trace should not create approval events.
func TestComparatorUnexpectedApprovalEvent(t *testing.T) {
	t.Parallel()

	scenario := makeHappyScenario()
	trace := makeMatchingTrace()
	trace.ApprovalEvents = []eval.TraceApprovalEvent{
		{ApprovalID: "ap-001", Action: "send_email", Status: "pending"},
	}
	setFinalState(&trace, map[string]any{"case.status": "In Progress"})

	result := eval.Compare(scenario, trace)

	if result.Pass {
		t.Error("expected Pass=false when approval appears unexpectedly")
	}
	if !hasMismatchDimension(result, eval.DimApprovalBehavior) {
		t.Errorf("expected mismatch in dimension %q", eval.DimApprovalBehavior)
	}
}

// TestComparatorContractValidationMismatch — failed traceability invariants must fail the comparison.
func TestComparatorContractValidationMismatch(t *testing.T) {
	t.Parallel()

	scenario := makeHappyScenario()
	trace := makeMatchingTrace()
	trace.ContractValidation = eval.TraceContractValidation{
		MutatorsTraceable: false,
		PolicysTraceable:  false,
	}
	setFinalState(&trace, map[string]any{"case.status": "In Progress"})

	result := eval.Compare(scenario, trace)

	if result.Pass {
		t.Error("expected Pass=false when contract validation fails")
	}
	if !hasMismatchDimension(result, eval.DimContractValidation) {
		t.Errorf("expected mismatch in dimension %q", eval.DimContractValidation)
	}
}

// TestComparatorJSONRender — result JSON is valid and includes scenario_id.
func TestComparatorJSONRender(t *testing.T) {
	t.Parallel()

	scenario := makeHappyScenario()
	trace := makeMatchingTrace()
	setFinalState(&trace, map[string]any{"case.status": "In Progress"})

	result := eval.Compare(scenario, trace)

	raw, err := result.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error: %v", err)
	}
	if !json.Valid(raw) {
		t.Error("ToJSON() produced invalid JSON")
	}
	if !strings.Contains(string(raw), scenario.ID) {
		t.Errorf("JSON must include scenario ID %q", scenario.ID)
	}
}

// TestComparatorTextRender — text output contains dimension labels.
func TestComparatorTextRender(t *testing.T) {
	t.Parallel()

	scenario := makeHappyScenario()
	trace := makeMatchingTrace()
	setFinalState(&trace, map[string]any{"case.status": "In Progress"})
	trace.ToolCalls = []eval.TraceToolCall{} // force mismatch

	result := eval.Compare(scenario, trace)
	text := result.ToText()

	if !strings.Contains(text, string(eval.DimToolCalls)) {
		t.Errorf("text output must mention dimension %q", eval.DimToolCalls)
	}
}

// TestComparatorDeterministic — same inputs always produce the same result.
func TestComparatorDeterministic(t *testing.T) {
	t.Parallel()

	scenario := makeHappyScenario()
	trace := makeMatchingTrace()
	scenario.Expected.FinalState = map[string]any{
		"account.tier": "Gold",
		"case.status":  "In Progress",
	}
	setFinalState(&trace, map[string]any{
		"account.tier": "Silver",
		"case.status":  "New",
	})
	trace.ToolCalls = []eval.TraceToolCall{} // force reproducible mismatch set

	r1 := eval.Compare(scenario, trace)
	r2 := eval.Compare(scenario, trace)

	raw1, err := r1.ToJSON()
	if err != nil {
		t.Fatalf("first ToJSON() error: %v", err)
	}
	raw2, err := r2.ToJSON()
	if err != nil {
		t.Fatalf("second ToJSON() error: %v", err)
	}

	if string(raw1) != string(raw2) {
		t.Errorf("Compare() is not deterministic:\nfirst=%s\nsecond=%s", raw1, raw2)
	}
}

// --- assertion helpers ---

func hasMismatchDimension(r eval.ComparisonResult, dim eval.MismatchDimension) bool {
	for _, m := range r.Mismatches {
		if m.Dimension == dim {
			return true
		}
	}
	return false
}

func findMismatch(r eval.ComparisonResult, dim eval.MismatchDimension) eval.Mismatch {
	for _, m := range r.Mismatches {
		if m.Dimension == dim {
			return m
		}
	}
	return eval.Mismatch{}
}
