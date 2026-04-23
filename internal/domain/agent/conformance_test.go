package agent

import "testing"

func TestConformanceSafeDSLCore(t *testing.T) {
	t.Parallel()

	result := EvaluateWorkflowConformance(`WORKFLOW resolve_support_case
ON case.created
IF case.priority == "high":
  SET case.status = "triaged"
  NOTIFY salesperson WITH "review"
AGENT support_bot WITH case.id
`, "")

	if result.Profile != ConformanceProfileSafe {
		t.Fatalf("Profile = %q, want %q", result.Profile, ConformanceProfileSafe)
	}
	if detail := findConformanceDetail(result, "missing_spec_source"); detail == nil || detail.Severity != ConformanceSeverityWarning {
		t.Fatalf("missing_spec_source detail = %#v", detail)
	}
	if result.Graph == nil || len(result.Graph.Nodes) == 0 {
		t.Fatalf("Graph = %#v, want populated graph", result.Graph)
	}
}

func TestConformanceSafeCartaFull(t *testing.T) {
	t.Parallel()

	result := EvaluateWorkflowConformance(`WORKFLOW resolve_support_case
ON case.created
SET case.status = "triaged"
`, `CARTA resolve_support_case
BUDGET
  daily_tokens: 50000
AGENT search_knowledge
  GROUNDS
    min_sources: 2
  PERMIT send_reply
  DELEGATE TO HUMAN
    reason: "Escalate"
  INVARIANT
    never: "send_pii"
`)

	if result.Profile != ConformanceProfileSafe {
		t.Fatalf("Profile = %q, want %q; details=%#v", result.Profile, ConformanceProfileSafe, result.Details)
	}
	if result.Graph == nil {
		t.Fatal("Graph = nil, want populated graph")
	}
	if findSemanticNode(t, result.Graph, SemanticNodePermit, "PERMIT send_reply").Source != SemanticSourceCarta {
		t.Fatal("permit node source is not Carta")
	}
}

func TestConformanceSafeLegacySpecSource(t *testing.T) {
	t.Parallel()

	result := EvaluateWorkflowConformance(`WORKFLOW resolve_support_case
ON case.created
SET case.status = "triaged"
`, `BEHAVIOR resolve case
The workflow should resolve a case.`)

	if result.Profile != ConformanceProfileSafe {
		t.Fatalf("Profile = %q, want %q", result.Profile, ConformanceProfileSafe)
	}
	if detail := findConformanceDetail(result, "legacy_spec_source"); detail == nil || detail.Severity != ConformanceSeverityInfo {
		t.Fatalf("legacy_spec_source detail = %#v", detail)
	}
}

func TestConformanceInvalidDSL(t *testing.T) {
	t.Parallel()

	result := EvaluateWorkflowConformance(`WORKFLOW resolve_support_case
ON case.created
`, "")

	if result.Profile != ConformanceProfileInvalid {
		t.Fatalf("Profile = %q, want %q", result.Profile, ConformanceProfileInvalid)
	}
	if detail := findConformanceDetail(result, "invalid_dsl"); detail == nil || detail.Severity != ConformanceSeverityError {
		t.Fatalf("invalid_dsl detail = %#v", detail)
	}
}

func TestConformanceInvalidCarta(t *testing.T) {
	t.Parallel()

	result := EvaluateWorkflowConformance(`WORKFLOW resolve_support_case
ON case.created
SET case.status = "triaged"
`, `CARTA resolve_support_case
SKILL fetch_remote`)

	if result.Profile != ConformanceProfileInvalid {
		t.Fatalf("Profile = %q, want %q", result.Profile, ConformanceProfileInvalid)
	}
	if detail := findConformanceDetail(result, "invalid_carta"); detail == nil || detail.Severity != ConformanceSeverityError {
		t.Fatalf("invalid_carta detail = %#v", detail)
	}
}

func TestConformanceExtendedForUnsupportedGraphNode(t *testing.T) {
	t.Parallel()

	graph := NewWorkflowSemanticGraph("future")
	graph.AddNode(WorkflowSemanticNode{
		ID:     "future:1",
		Kind:   SemanticNodeKind("future"),
		Label:  "future construct",
		Source: SemanticSourceDSL,
	})

	result := EvaluateGraphConformance(graph)
	if result.Profile != ConformanceProfileExtended {
		t.Fatalf("Profile = %q, want %q", result.Profile, ConformanceProfileExtended)
	}
	if detail := findConformanceDetail(result, "unsupported_semantic_node"); detail == nil || detail.Severity != ConformanceSeverityWarning {
		t.Fatalf("unsupported_semantic_node detail = %#v", detail)
	}
}

func TestConformanceCallStatementIsExtended(t *testing.T) { // CLSF-54
	t.Parallel()

	result := EvaluateWorkflowConformance(`WORKFLOW x
ON case.created
CALL search WITH query AS result
`, "")

	if result.Profile != ConformanceProfileExtended {
		t.Fatalf("Profile = %q, want %q — CALL must be extended until runtime exists", result.Profile, ConformanceProfileExtended)
	}
	if detail := findConformanceDetail(result, "unsupported_semantic_node"); detail == nil {
		t.Fatal("expected unsupported_semantic_node detail for CALL statement")
	}
}

func TestConformanceApproveStatementIsExtended(t *testing.T) { // CLSF-54
	t.Parallel()

	result := EvaluateWorkflowConformance(`WORKFLOW x
ON case.created
APPROVE send_email role manager
`, "")

	if result.Profile != ConformanceProfileExtended {
		t.Fatalf("Profile = %q, want %q — APPROVE must be extended until runtime exists", result.Profile, ConformanceProfileExtended)
	}
	if detail := findConformanceDetail(result, "unsupported_semantic_node"); detail == nil {
		t.Fatal("expected unsupported_semantic_node detail for APPROVE statement")
	}
}

func TestConformanceMixedV0AndCallIsExtended(t *testing.T) { // CLSF-54
	t.Parallel()

	result := EvaluateWorkflowConformance(`WORKFLOW x
ON case.created
SET case.status = "open"
CALL search WITH query AS result
NOTIFY manager WITH "done"
`, "")

	if result.Profile != ConformanceProfileExtended {
		t.Fatalf("Profile = %q, want extended — one CALL promotes the whole workflow", result.Profile)
	}
}

func TestConformanceV0OnlyRemainsSafe(t *testing.T) { // CLSF-54 regression
	t.Parallel()

	result := EvaluateWorkflowConformance(`WORKFLOW x
ON case.created
SET case.status = "open"
NOTIFY manager WITH "done"
`, "")

	if result.Profile != ConformanceProfileSafe {
		t.Fatalf("Profile = %q, want safe — v0-only workflow must not regress", result.Profile)
	}
}

func findConformanceDetail(result ConformanceResult, code string) *ConformanceDetail {
	for _, detail := range result.Details {
		if detail.Code == code {
			return &detail
		}
	}
	return nil
}
