package agent

import "testing"

func TestRunInitialSpecDSLChecks_NoViolationsWhenBehaviorIsCovered(t *testing.T) {
	t.Parallel()

	spec := ParsePartialSpec(`BEHAVIOR resolve_support_case`)
	program, err := ParseAndValidateDSL("WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"resolved\"")
	if err != nil {
		t.Fatalf("ParseAndValidateDSL() error = %v", err)
	}

	violations, warnings := RunInitialSpecDSLChecks(spec, program)
	if len(violations) != 0 {
		t.Fatalf("Violations = %#v, want none", violations)
	}
	if len(warnings) != 0 {
		t.Fatalf("Warnings = %#v, want none", warnings)
	}
}

func TestRunInitialSpecDSLChecks_DetectsMissingBehaviorCoverage(t *testing.T) {
	t.Parallel()

	spec := ParsePartialSpec(`BEHAVIOR notify_salesperson`)
	program, err := ParseAndValidateDSL("WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"resolved\"")
	if err != nil {
		t.Fatalf("ParseAndValidateDSL() error = %v", err)
	}

	violations, _ := RunInitialSpecDSLChecks(spec, program)
	if len(violations) != 1 {
		t.Fatalf("len(Violations) = %d, want 1", len(violations))
	}
	got := violations[0]
	if got.Code != "behavior_no_coverage" {
		t.Fatalf("Code = %q, want behavior_no_coverage", got.Code)
	}
	if got.CheckID != judgeCheckBehaviorCoverage {
		t.Fatalf("CheckID = %d, want %d", got.CheckID, judgeCheckBehaviorCoverage)
	}
}

func TestRunInitialSpecDSLChecks_UsesStatementCoverage(t *testing.T) {
	t.Parallel()

	spec := ParsePartialSpec(`BEHAVIOR notify_salesperson`)
	program, err := ParseAndValidateDSL("WORKFLOW resolve_support_case\nON case.created\nNOTIFY salesperson WITH \"review\"")
	if err != nil {
		t.Fatalf("ParseAndValidateDSL() error = %v", err)
	}

	violations, _ := RunInitialSpecDSLChecks(spec, program)
	if len(violations) != 0 {
		t.Fatalf("Violations = %#v, want none", violations)
	}
}

func TestRunCartaPermitChecks_FlagsToolWithoutPermit(t *testing.T) {
	t.Parallel()

	carta := &CartaSummary{
		Permits: []CartaPermit{{Tool: "send_reply"}},
	}
	program, err := ParseAndValidateDSL("WORKFLOW resolve_support_case\nON case.created\nNOTIFY salesperson WITH \"review\"")
	if err != nil {
		t.Fatalf("ParseAndValidateDSL() error = %v", err)
	}

	violations := RunCartaPermitChecks(carta, program)
	if len(violations) != 1 {
		t.Fatalf("len(Violations) = %d, want 1", len(violations))
	}
	got := violations[0]
	if got.Code != "tool_not_permitted" {
		t.Fatalf("Code = %q, want tool_not_permitted", got.Code)
	}
	if got.CheckID != CartaCheckPermit {
		t.Fatalf("CheckID = %d, want %d", got.CheckID, CartaCheckPermit)
	}
}

func TestRunCartaPermitChecks_AllowsPermittedTool(t *testing.T) {
	t.Parallel()

	carta := &CartaSummary{
		Permits: []CartaPermit{{Tool: "CREATE_TASK"}},
	}
	program, err := ParseAndValidateDSL("WORKFLOW resolve_support_case\nON case.created\nNOTIFY salesperson WITH \"review\"")
	if err != nil {
		t.Fatalf("ParseAndValidateDSL() error = %v", err)
	}

	violations := RunCartaPermitChecks(carta, program)
	if len(violations) != 0 {
		t.Fatalf("Violations = %#v, want none", violations)
	}
}

func TestRunCartaPermitChecks_IgnoresNonToolStatementsAndRecurseIf(t *testing.T) {
	t.Parallel()

	carta := &CartaSummary{
		Permits: []CartaPermit{{Tool: "update_case"}},
	}
	program, err := ParseAndValidateDSL("WORKFLOW resolve_support_case\nON case.created\nIF case.priority == \"high\":\n  NOTIFY contact WITH \"done\"\nWAIT 5 minutes")
	if err != nil {
		t.Fatalf("ParseAndValidateDSL() error = %v", err)
	}

	violations := RunCartaPermitChecks(carta, program)
	if len(violations) != 1 {
		t.Fatalf("len(Violations) = %d, want 1", len(violations))
	}
	if violations[0].Location != "NOTIFY contact" {
		t.Fatalf("Location = %q, want NOTIFY contact", violations[0].Location)
	}
}

func TestRunCartaCoverageChecks_NoOpWithoutSpecBehaviors(t *testing.T) {
	t.Parallel()

	violations := RunCartaCoverageChecks(&CartaSummary{}, nil)
	if len(violations) != 0 {
		t.Fatalf("Violations = %#v, want none", violations)
	}
}

func TestRunCartaCoverageChecks_FlagsBehaviorWithoutPermitOrDelegate(t *testing.T) {
	t.Parallel()

	carta := &CartaSummary{
		Permits: []CartaPermit{{Tool: "send_reply"}},
	}
	spec := &SpecSummary{
		Behaviors: []SpecBehavior{{Name: "escalate_unresolved", Line: 7}},
	}

	violations := RunCartaCoverageChecks(carta, spec)
	if len(violations) != 1 {
		t.Fatalf("len(Violations) = %d, want 1", len(violations))
	}
	got := violations[0]
	if got.Code != "behavior_no_permit_or_delegate" {
		t.Fatalf("Code = %q, want behavior_no_permit_or_delegate", got.Code)
	}
	if got.CheckID != CartaCheckCoverage {
		t.Fatalf("CheckID = %d, want %d", got.CheckID, CartaCheckCoverage)
	}
}

func TestRunCartaCoverageChecks_AllowsCoveredBehaviorByPermit(t *testing.T) {
	t.Parallel()

	carta := &CartaSummary{
		Permits: []CartaPermit{{Tool: "send_reply"}},
	}
	spec := &SpecSummary{
		Behaviors: []SpecBehavior{{Name: "send_reply_to_contact", Line: 3}},
	}

	violations := RunCartaCoverageChecks(carta, spec)
	if len(violations) != 0 {
		t.Fatalf("Violations = %#v, want none", violations)
	}
}

func TestRunCartaCoverageChecks_AllowsAnyBehaviorWhenDelegateExists(t *testing.T) {
	t.Parallel()

	carta := &CartaSummary{
		Delegates: []CartaDelegate{{Reason: "Escalate to human"}},
	}
	spec := &SpecSummary{
		Behaviors: []SpecBehavior{{Name: "escalate_unresolved", Line: 4}},
	}

	violations := RunCartaCoverageChecks(carta, spec)
	if len(violations) != 0 {
		t.Fatalf("Violations = %#v, want none", violations)
	}
}

func TestRunCartaGroundsPresenceCheck_WarnsWhenGroundsMissing(t *testing.T) {
	t.Parallel()

	warnings := RunCartaGroundsPresenceCheck(&CartaSummary{})
	if len(warnings) != 1 {
		t.Fatalf("len(Warnings) = %d, want 1", len(warnings))
	}
	got := warnings[0]
	if got.Code != "carta_missing_grounds" {
		t.Fatalf("Code = %q, want carta_missing_grounds", got.Code)
	}
	if got.CheckID != CartaCheckGrounds {
		t.Fatalf("CheckID = %d, want %d", got.CheckID, CartaCheckGrounds)
	}
}

func TestRunCartaGroundsPresenceCheck_NoWarningWhenGroundsExist(t *testing.T) {
	t.Parallel()

	warnings := RunCartaGroundsPresenceCheck(&CartaSummary{
		Grounds: &CartaGrounds{MinSources: 2},
	})
	if len(warnings) != 0 {
		t.Fatalf("Warnings = %#v, want none", warnings)
	}
}

func TestRunCartaSpecDSLChecks_CombinesViolationsAndWarnings(t *testing.T) {
	t.Parallel()

	carta := &CartaSummary{
		Permits: []CartaPermit{{Tool: "send_reply"}},
	}
	program, err := ParseAndValidateDSL("WORKFLOW resolve_support_case\nON case.created\nNOTIFY salesperson WITH \"review\"")
	if err != nil {
		t.Fatalf("ParseAndValidateDSL() error = %v", err)
	}
	spec := &SpecSummary{
		Behaviors: []SpecBehavior{{Name: "escalate_unresolved", Line: 5}},
	}

	violations, warnings := RunCartaSpecDSLChecks(carta, program, spec)
	if len(violations) != 2 {
		t.Fatalf("len(Violations) = %d, want 2", len(violations))
	}
	if len(warnings) != 1 {
		t.Fatalf("len(Warnings) = %d, want 1", len(warnings))
	}
}
