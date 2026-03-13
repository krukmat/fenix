package agent

import "testing"

func TestRunProtocolJudgeChecks_FlagsDispatchWithoutContractSelector(t *testing.T) {
	t.Parallel()

	program, err := ParseDSL("WORKFLOW route_case\nON case.created\nDISPATCH TO support_agent WITH {\"case_id\":\"case-1\"}")
	if err != nil {
		t.Fatalf("ParseDSL() error = %v", err)
	}

	violations, warnings := RunProtocolJudgeChecks(program)
	if len(violations) != 1 {
		t.Fatalf("len(violations) = %d, want 1", len(violations))
	}
	if violations[0].Code != "dispatch_contract_missing" {
		t.Fatalf("Code = %q, want dispatch_contract_missing", violations[0].Code)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v, want none", warnings)
	}
}

func TestRunProtocolJudgeChecks_WarnsOnInlineDispatchDSL(t *testing.T) {
	t.Parallel()

	program, err := ParseDSL("WORKFLOW route_case\nON case.created\nDISPATCH TO support_agent WITH {\"dsl_source\":\"WORKFLOW x\\nON y\\nSET case.status = \\\"open\\\"\"}")
	if err != nil {
		t.Fatalf("ParseDSL() error = %v", err)
	}

	violations, warnings := RunProtocolJudgeChecks(program)
	if len(violations) != 0 {
		t.Fatalf("violations = %#v, want none", violations)
	}
	if len(warnings) != 1 {
		t.Fatalf("len(warnings) = %d, want 1", len(warnings))
	}
	if warnings[0].Code != "dispatch_inline_dsl_portability" {
		t.Fatalf("Code = %q, want dispatch_inline_dsl_portability", warnings[0].Code)
	}
}

func TestRunProtocolJudgeChecks_FlagsAmbiguousDispatchContract(t *testing.T) {
	t.Parallel()

	program, err := ParseDSL("WORKFLOW route_case\nON case.created\nDISPATCH TO support_agent WITH {\"workflow_name\":\"x\",\"dsl_source\":\"WORKFLOW x\\nON y\\nSET case.status = \\\"open\\\"\"}")
	if err != nil {
		t.Fatalf("ParseDSL() error = %v", err)
	}

	violations, _ := RunProtocolJudgeChecks(program)
	if len(violations) != 1 {
		t.Fatalf("len(violations) = %d, want 1", len(violations))
	}
	if violations[0].Code != "dispatch_contract_ambiguous" {
		t.Fatalf("Code = %q, want dispatch_contract_ambiguous", violations[0].Code)
	}
}

func TestRunProtocolJudgeChecks_FlagsInvalidSurfaceConfidence(t *testing.T) {
	t.Parallel()

	program, err := ParseDSL("WORKFLOW surface_case\nON case.created\nSURFACE case TO salesperson.view WITH {\"value\":\"review\",\"confidence\":1.5}")
	if err != nil {
		t.Fatalf("ParseDSL() error = %v", err)
	}

	violations, warnings := RunProtocolJudgeChecks(program)
	if len(violations) != 1 {
		t.Fatalf("len(violations) = %d, want 1", len(violations))
	}
	if violations[0].Code != "surface_confidence_invalid" {
		t.Fatalf("Code = %q, want surface_confidence_invalid", violations[0].Code)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v, want none", warnings)
	}
}

func TestRunProtocolJudgeChecks_WarnsOnAmbiguousSurfaceView(t *testing.T) {
	t.Parallel()

	program, err := ParseDSL("WORKFLOW surface_case\nON case.created\nSURFACE case TO salesperson WITH {\"value\":\"review\"}")
	if err != nil {
		t.Fatalf("ParseDSL() error = %v", err)
	}

	violations, warnings := RunProtocolJudgeChecks(program)
	if len(violations) != 0 {
		t.Fatalf("violations = %#v, want none", violations)
	}
	if len(warnings) != 1 {
		t.Fatalf("len(warnings) = %d, want 1", len(warnings))
	}
	if warnings[0].Code != "surface_view_ambiguous" {
		t.Fatalf("Code = %q, want surface_view_ambiguous", warnings[0].Code)
	}
}
