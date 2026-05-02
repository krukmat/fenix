package eval_test

import (
	"path/filepath"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/eval"
)

// F1-T1/F1-T2: Golden scenario schema validation and YAML loading.

func TestLoadGoldenScenario_HappyPath(t *testing.T) {
	t.Parallel()

	path := filepath.Join("testdata", "scenarios", "sc_support_happy_path.yaml")
	sc, err := eval.LoadGoldenScenario(path)
	if err != nil {
		t.Fatalf("LoadGoldenScenario(%q): %v", path, err)
	}

	if sc.ID == "" {
		t.Error("expected non-empty ID")
	}
	if sc.Domain != "support" {
		t.Errorf("expected domain=support, got %q", sc.Domain)
	}
	if sc.InputEvent.Type == "" {
		t.Error("expected non-empty input_event.type")
	}
	if len(sc.InitialState) == 0 {
		t.Error("expected non-empty initial_state")
	}
	if len(sc.Expected.ToolCalls) == 0 {
		t.Error("expected at least one expected_tool_call")
	}
	if len(sc.Expected.ForbiddenToolCalls) == 0 {
		t.Error("expected at least one forbidden_tool_call")
	}
	if len(sc.Expected.PolicyDecisions) == 0 {
		t.Error("expected at least one expected_policy_decision")
	}
	if sc.Expected.FinalOutcome == "" {
		t.Error("expected non-empty final_outcome")
	}
	if len(sc.Expected.AuditEvents) == 0 {
		t.Error("expected at least one expected_audit_event")
	}
	if sc.Thresholds.MaxLatencyMs == 0 {
		t.Error("expected non-zero max_latency_ms threshold")
	}
}

func TestLoadGoldenScenario_PolicyDenial(t *testing.T) {
	t.Parallel()

	path := filepath.Join("testdata", "scenarios", "sc_support_policy_denial.yaml")
	sc, err := eval.LoadGoldenScenario(path)
	if err != nil {
		t.Fatalf("LoadGoldenScenario(%q): %v", path, err)
	}

	if sc.ID == "" {
		t.Error("expected non-empty ID")
	}
	if len(sc.Expected.ForbiddenToolCalls) == 0 {
		t.Error("expected at least one forbidden_tool_call for denial scenario")
	}

	hasDeny := false
	for _, pd := range sc.Expected.PolicyDecisions {
		if pd.ExpectedOutcome == "deny" {
			hasDeny = true
			break
		}
	}
	if !hasDeny {
		t.Error("policy denial scenario must have at least one expected_policy_decision with outcome=deny")
	}
}

func TestLoadGoldenScenario_WeakEvidenceAbstention(t *testing.T) {
	t.Parallel()

	path := filepath.Join("testdata", "scenarios", "sc_support_weak_evidence_abstention.yaml")
	sc, err := eval.LoadGoldenScenario(path)
	if err != nil {
		t.Fatalf("LoadGoldenScenario(%q): %v", path, err)
	}
	if !sc.Expected.ShouldAbstain {
		t.Error("weak evidence scenario must have should_abstain=true")
	}
	if sc.Expected.AbstainReason == "" {
		t.Error("abstain_reason must be set when should_abstain=true")
	}
}

func TestLoadGoldenScenario_SensitiveMutationApproval(t *testing.T) {
	t.Parallel()

	path := filepath.Join("testdata", "scenarios", "sc_support_sensitive_mutation_approval.yaml")
	sc, err := eval.LoadGoldenScenario(path)
	if err != nil {
		t.Fatalf("LoadGoldenScenario(%q): %v", path, err)
	}
	if sc.Expected.ApprovalBehavior == nil {
		t.Fatal("sensitive mutation scenario must define approval_behavior")
	}
	if !sc.Expected.ApprovalBehavior.Required {
		t.Error("approval_behavior.required must be true for sensitive mutation scenario")
	}

	hasRequireApproval := false
	for _, pd := range sc.Expected.PolicyDecisions {
		if pd.ExpectedOutcome == "require_approval" {
			hasRequireApproval = true
			break
		}
	}
	if !hasRequireApproval {
		t.Error("sensitive mutation scenario must have a policy decision with outcome=require_approval")
	}
}

func TestLoadGoldenScenario_ToolFailureHandoff(t *testing.T) {
	t.Parallel()

	path := filepath.Join("testdata", "scenarios", "sc_support_tool_failure_handoff.yaml")
	sc, err := eval.LoadGoldenScenario(path)
	if err != nil {
		t.Fatalf("LoadGoldenScenario(%q): %v", path, err)
	}
	if sc.Expected.FinalOutcome != "escalated" {
		t.Errorf("tool failure scenario must have final_outcome=escalated, got %q", sc.Expected.FinalOutcome)
	}
}

func TestLoadGoldenScenario_SalesBriefIncompleteContext(t *testing.T) {
	t.Parallel()

	path := filepath.Join("testdata", "scenarios", "sc_sales_brief_incomplete_context.yaml")
	sc, err := eval.LoadGoldenScenario(path)
	if err != nil {
		t.Fatalf("LoadGoldenScenario(%q): %v", path, err)
	}
	if sc.Domain != "sales" {
		t.Errorf("expected domain=sales, got %q", sc.Domain)
	}
}

func TestLoadGoldenScenario_WorkflowActivationBlocked(t *testing.T) {
	t.Parallel()

	path := filepath.Join("testdata", "scenarios", "sc_workflow_activation_blocked.yaml")
	sc, err := eval.LoadGoldenScenario(path)
	if err != nil {
		t.Fatalf("LoadGoldenScenario(%q): %v", path, err)
	}
	if sc.Expected.FinalOutcome != "blocked" {
		t.Errorf("workflow activation blocked scenario must have final_outcome=blocked, got %q", sc.Expected.FinalOutcome)
	}
}

// --- Validation unit tests ---

func TestValidateGoldenScenario_MissingID(t *testing.T) {
	t.Parallel()

	sc := eval.GoldenScenario{
		Domain:     "support",
		InputEvent: eval.ScenarioInputEvent{Type: "case.created"},
	}
	if err := sc.Validate(); err == nil {
		t.Error("expected validation error for missing ID")
	}
}

func TestValidateGoldenScenario_InvalidDomain(t *testing.T) {
	t.Parallel()

	sc := eval.GoldenScenario{
		ID:         "sc-001",
		Domain:     "unknown_domain",
		InputEvent: eval.ScenarioInputEvent{Type: "case.created"},
	}
	if err := sc.Validate(); err == nil {
		t.Error("expected validation error for invalid domain")
	}
}

func TestValidateGoldenScenario_MissingInputEventType(t *testing.T) {
	t.Parallel()

	sc := eval.GoldenScenario{
		ID:     "sc-001",
		Domain: "support",
	}
	if err := sc.Validate(); err == nil {
		t.Error("expected validation error for missing input_event.type")
	}
}

func TestValidateGoldenScenario_InvalidPolicyOutcome(t *testing.T) {
	t.Parallel()

	sc := eval.GoldenScenario{
		ID:         "sc-001",
		Domain:     "support",
		InputEvent: eval.ScenarioInputEvent{Type: "case.created"},
		Expected: eval.ScenarioExpected{
			PolicyDecisions: []eval.ExpectedPolicyDecision{
				{Action: "tool:send_email", ExpectedOutcome: "maybe"},
			},
		},
	}
	if err := sc.Validate(); err == nil {
		t.Error("expected validation error for invalid policy outcome")
	}
}

func TestValidateGoldenScenario_Valid(t *testing.T) {
	t.Parallel()

	sc := eval.GoldenScenario{
		ID:         "sc-001",
		Domain:     "support",
		InputEvent: eval.ScenarioInputEvent{Type: "case.created"},
	}
	if err := sc.Validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestLoadGoldenScenario_FileNotFound(t *testing.T) {
	t.Parallel()

	_, err := eval.LoadGoldenScenario("testdata/scenarios/nonexistent.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
