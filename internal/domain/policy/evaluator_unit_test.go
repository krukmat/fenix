// Traces: FR-071
// Unit tests for pure functions in evaluator.go — no DB, no external deps.
package policy

import (
	"context"
	"testing"
)

func newTestCtx() context.Context { return context.Background() }

// --- matchesConditions ---

func TestMatchesConditions_EmptyConditions(t *testing.T) {
	if !matchesConditions(nil, nil) {
		t.Fatal("empty conditions should always match")
	}
	if !matchesConditions(map[string]any{}, map[string]string{"k": "v"}) {
		t.Fatal("empty conditions map should always match")
	}
}

func TestMatchesConditions_AllMatch(t *testing.T) {
	conditions := map[string]any{"env": "prod", "tenant": "acme"}
	attrs := map[string]string{"env": "prod", "tenant": "acme", "extra": "ignored"}
	if !matchesConditions(conditions, attrs) {
		t.Fatal("expected match when all conditions satisfied")
	}
}

func TestMatchesConditions_OneKeyMismatch(t *testing.T) {
	conditions := map[string]any{"env": "prod"}
	attrs := map[string]string{"env": "staging"}
	if matchesConditions(conditions, attrs) {
		t.Fatal("expected no match when attr value differs")
	}
}

func TestMatchesConditions_MissingKey(t *testing.T) {
	conditions := map[string]any{"env": "prod"}
	attrs := map[string]string{"other": "prod"}
	if matchesConditions(conditions, attrs) {
		t.Fatal("expected no match when key is absent from attrs")
	}
}

func TestMatchesConditions_NilAttrs(t *testing.T) {
	conditions := map[string]any{"env": "prod"}
	if matchesConditions(conditions, nil) {
		t.Fatal("expected no match when attrs is nil and conditions non-empty")
	}
}

func TestMatchesConditions_NonStringExpected(t *testing.T) {
	conditions := map[string]any{"count": 42} // non-string expected value
	attrs := map[string]string{"count": "42"}
	if matchesConditions(conditions, attrs) {
		t.Fatal("expected no match when condition value is non-string")
	}
}

// --- parsePolicyRules ---

func TestParsePolicyRules_EmptyString(t *testing.T) {
	rules, err := parsePolicyRules("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rules != nil {
		t.Fatalf("expected nil rules, got %v", rules)
	}
}

func TestParsePolicyRules_WhitespaceOnly(t *testing.T) {
	rules, err := parsePolicyRules("   ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rules != nil {
		t.Fatalf("expected nil rules, got %v", rules)
	}
}

func TestParsePolicyRules_DocFormat(t *testing.T) {
	raw := `{"rules":[{"id":"r1","resource":"tools","action":"*","effect":"allow","priority":1}]}`
	rules, err := parsePolicyRules(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 || rules[0].ID != "r1" {
		t.Fatalf("expected 1 rule with id=r1, got %v", rules)
	}
}

func TestParsePolicyRules_ArrayFormat(t *testing.T) {
	raw := `[{"id":"r2","resource":"api","action":"read","effect":"deny","priority":5}]`
	rules, err := parsePolicyRules(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 || rules[0].ID != "r2" {
		t.Fatalf("expected 1 rule with id=r2, got %v", rules)
	}
}

func TestParsePolicyRules_InvalidJSON(t *testing.T) {
	_, err := parsePolicyRules(`not-valid-json`)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// --- resolveDeterministicRule ---

func TestResolveDeterministicRule_HigherPriorityWins(t *testing.T) {
	rules := []policyRule{
		{ID: "low", Effect: "allow", Priority: 1},
		{ID: "high", Effect: "allow", Priority: 10},
	}
	got := resolveDeterministicRule(rules)
	if got.ID != "high" {
		t.Fatalf("expected high-priority rule, got %q", got.ID)
	}
}

func TestResolveDeterministicRule_DenyWinsAtSamePriority(t *testing.T) {
	rules := []policyRule{
		{ID: "allow_rule", Effect: "allow", Priority: 5},
		{ID: "deny_rule", Effect: "deny", Priority: 5},
	}
	got := resolveDeterministicRule(rules)
	if got.ID != "deny_rule" {
		t.Fatalf("expected deny rule to win at same priority, got %q", got.ID)
	}
}

func TestResolveDeterministicRule_IDTiebreak(t *testing.T) {
	rules := []policyRule{
		{ID: "b_rule", Effect: "allow", Priority: 5},
		{ID: "a_rule", Effect: "allow", Priority: 5},
	}
	got := resolveDeterministicRule(rules)
	if got.ID != "a_rule" {
		t.Fatalf("expected lexicographically first ID to win, got %q", got.ID)
	}
}

// --- candidateToolActions ---

func TestCandidateToolActions_Empty(t *testing.T) {
	got := candidateToolActions("")
	if got != nil {
		t.Fatalf("expected nil for empty input, got %v", got)
	}
}

func TestCandidateToolActions_NoPrefix(t *testing.T) {
	got := candidateToolActions("update_case")
	if len(got) != 1 || got[0] != "update_case" {
		t.Fatalf("expected [update_case], got %v", got)
	}
}

func TestCandidateToolActions_ToolsPrefix(t *testing.T) {
	got := candidateToolActions("tools:update_case")
	if len(got) != 2 {
		t.Fatalf("expected 2 candidates, got %v", got)
	}
	if got[0] != "tools:update_case" || got[1] != "update_case" {
		t.Fatalf("unexpected candidates: %v", got)
	}
}

func TestCandidateToolActions_ToolsPrefixOnly(t *testing.T) {
	// Input is trimmed before processing: "tools:  " → "tools:"
	// After stripping prefix: "" (empty) → no second element appended
	got := candidateToolActions("tools:  ")
	if len(got) != 1 || got[0] != "tools:" {
		t.Fatalf("unexpected candidates for tools-prefix-only: %v", got)
	}
}

// --- resolvePolicyOutcome ---

func TestResolvePolicyOutcome_Allow(t *testing.T) {
	outcome := resolvePolicyOutcome(true)
	if string(outcome) != "success" {
		t.Fatalf("expected success, got %q", outcome)
	}
}

func TestResolvePolicyOutcome_Deny(t *testing.T) {
	outcome := resolvePolicyOutcome(false)
	if string(outcome) != "denied" {
		t.Fatalf("expected denied, got %q", outcome)
	}
}

// --- EvaluatePolicyDecision with conditions ---

func TestEvaluatePolicyDecision_ConditionMatch_Allows(t *testing.T) {
	db := setupPolicyTestDB(t)
	workspaceID, userID := seedWorkspaceUserRole(t, db, `{}`)

	policyJSON := `{"rules":[
		{"id":"allow_prod","resource":"tools","action":"*","effect":"allow","priority":1,
		 "conditions":{"env":"prod"}}
	]}`
	seedActivePolicyVersion(t, db, workspaceID, 1, policyJSON)

	engine := NewPolicyEngine(db, nil, nil)
	decision, err := engine.EvaluatePolicyDecision(
		newTestCtx(), workspaceID, userID, "tools", "update_case",
		map[string]string{"env": "prod"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !decision.Allow {
		t.Fatal("expected allow when condition matches")
	}
}

func TestEvaluatePolicyDecision_ConditionMismatch_Denies(t *testing.T) {
	db := setupPolicyTestDB(t)
	workspaceID, userID := seedWorkspaceUserRole(t, db, `{}`)

	policyJSON := `{"rules":[
		{"id":"allow_prod","resource":"tools","action":"*","effect":"allow","priority":1,
		 "conditions":{"env":"prod"}}
	]}`
	seedActivePolicyVersion(t, db, workspaceID, 1, policyJSON)

	engine := NewPolicyEngine(db, nil, nil)
	decision, err := engine.EvaluatePolicyDecision(
		newTestCtx(), workspaceID, userID, "tools", "update_case",
		map[string]string{"env": "staging"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Allow {
		t.Fatal("expected deny when condition does not match")
	}
}
