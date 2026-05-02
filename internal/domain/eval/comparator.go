package eval

// F3-T1: Deterministic trace comparator.
// Compare(GoldenScenario, ActualRunTrace) → ComparisonResult with per-dimension mismatch evidence.
// No LLM, no randomness — same inputs always produce the same output.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
)

// MismatchDimension identifies which comparison axis produced a mismatch.
type MismatchDimension string

const (
	DimFinalOutcome       MismatchDimension = "final_outcome"
	DimRequiredEvidence   MismatchDimension = "required_evidence"
	DimForbiddenEvidence  MismatchDimension = "forbidden_evidence"
	DimPolicyDecisions    MismatchDimension = "policy_decisions"
	DimToolCalls          MismatchDimension = "tool_calls"
	DimExtraToolCalls     MismatchDimension = "extra_tool_calls"
	DimForbiddenToolCalls MismatchDimension = "forbidden_tool_calls"
	DimApprovalBehavior   MismatchDimension = "approval_behavior"
	DimFinalState         MismatchDimension = "final_state"
	DimAuditEvents        MismatchDimension = "audit_events"
	DimContractValidation MismatchDimension = "contract_validation"
)

// fmtStateField and fmtPolicyAction are goconst-required format string constants.
const (
	fmtStateField  = "%s=%v"
	fmtPolicyField = "%s → %s"
	fmtAbsentValue = "absent: %s"
)

// Mismatch captures one failed check with explicit expected/actual evidence.
type Mismatch struct {
	Dimension MismatchDimension `json:"dimension"`
	Expected  string            `json:"expected"`
	Actual    string            `json:"actual"`
	Evidence  string            `json:"evidence"` // human-readable summary
}

// ComparatorResult is the output of Compare. Pass is true only when Mismatches is empty.
type ComparatorResult struct {
	ScenarioID string     `json:"scenario_id"`
	RunID      string     `json:"run_id"`
	Pass       bool       `json:"pass"`
	Mismatches []Mismatch `json:"mismatches"`
}

// ComparisonResult is kept as a compatibility alias while callers migrate to ComparatorResult.
type ComparisonResult = ComparatorResult

// ToJSON returns the result as canonical JSON bytes.
func (r ComparatorResult) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// ToText returns a human-readable multi-line summary.
func (r ComparatorResult) ToText() string {
	var buf bytes.Buffer
	status := "PASS"
	if !r.Pass {
		status = "FAIL"
	}
	fmt.Fprintf(&buf, "[%s] scenario=%s run=%s mismatches=%d\n", status, r.ScenarioID, r.RunID, len(r.Mismatches))
	for _, m := range r.Mismatches {
		fmt.Fprintf(&buf, "  [%s] expected=%q actual=%q | %s\n", m.Dimension, m.Expected, m.Actual, m.Evidence)
	}
	return buf.String()
}

// Compare runs all comparison dimensions and returns a ComparisonResult.
// Deterministic: same inputs always produce the same result.
func Compare(scenario GoldenScenario, trace ActualRunTrace) ComparatorResult {
	var mismatches []Mismatch

	mismatches = append(mismatches, compareFinalOutcome(scenario.Expected, trace)...)
	mismatches = append(mismatches, compareRequiredEvidence(scenario.Expected, trace)...)
	mismatches = append(mismatches, compareForbiddenEvidence(scenario.Expected, trace)...)
	mismatches = append(mismatches, comparePolicyDecisions(scenario.Expected, trace)...)
	mismatches = append(mismatches, compareToolCalls(scenario.Expected, trace)...)
	mismatches = append(mismatches, compareApprovalBehavior(scenario.Expected, trace)...)
	mismatches = append(mismatches, compareFinalState(scenario.Expected, trace)...)
	mismatches = append(mismatches, compareAuditEvents(scenario.Expected, trace)...)
	mismatches = append(mismatches, compareContractValidation(trace)...)
	sortMismatches(mismatches)

	pass := true
	for _, m := range mismatches {
		if m.Dimension != DimExtraToolCalls { // extra tool calls are informational only
			pass = false
			break
		}
	}

	return ComparatorResult{
		ScenarioID: scenario.ID,
		RunID:      trace.RunID,
		Pass:       pass,
		Mismatches: mismatches,
	}
}

func compareFinalOutcome(expected ScenarioExpected, trace ActualRunTrace) []Mismatch {
	if expected.FinalOutcome == "" {
		return nil
	}
	if trace.FinalOutcome != expected.FinalOutcome {
		return []Mismatch{{
			Dimension: DimFinalOutcome,
			Expected:  expected.FinalOutcome,
			Actual:    trace.FinalOutcome,
			Evidence:  fmt.Sprintf("final_outcome: expected %q, got %q", expected.FinalOutcome, trace.FinalOutcome),
		}}
	}
	return nil
}

func compareRequiredEvidence(expected ScenarioExpected, trace ActualRunTrace) []Mismatch {
	actual := stringSet(trace.EvidenceSources)
	var out []Mismatch
	for _, req := range expected.RequiredEvidence {
		if _, ok := actual[req]; !ok {
			out = append(out, Mismatch{
				Dimension: DimRequiredEvidence,
				Expected:  req,
				Actual:    "",
				Evidence:  fmt.Sprintf("required evidence %q not found in trace evidence sources", req),
			})
		}
	}
	return out
}

func compareForbiddenEvidence(expected ScenarioExpected, trace ActualRunTrace) []Mismatch {
	actual := stringSet(trace.EvidenceSources)
	var out []Mismatch
	for _, forbidden := range expected.ForbiddenEvidence {
		if _, ok := actual[forbidden]; ok {
			out = append(out, Mismatch{
				Dimension: DimForbiddenEvidence,
				Expected:  fmt.Sprintf(fmtAbsentValue, forbidden),
				Actual:    forbidden,
				Evidence:  fmt.Sprintf("forbidden evidence source %q was used in trace", forbidden),
			})
		}
	}
	return out
}

func comparePolicyDecisions(expected ScenarioExpected, trace ActualRunTrace) []Mismatch {
	actualMap := make(map[string]string, len(trace.PolicyDecisions))
	for _, pd := range trace.PolicyDecisions {
		actualMap[pd.Action] = pd.Outcome
	}

	var out []Mismatch
	for _, exp := range expected.PolicyDecisions {
		actual, found := actualMap[exp.Action]
		if !found {
			out = append(out, Mismatch{
				Dimension: DimPolicyDecisions,
				Expected:  fmt.Sprintf(fmtPolicyField, exp.Action, exp.ExpectedOutcome),
				Actual:    "missing",
				Evidence:  fmt.Sprintf("expected policy decision for action %q not found in trace", exp.Action),
			})
			continue
		}
		if actual != exp.ExpectedOutcome {
			out = append(out, Mismatch{
				Dimension: DimPolicyDecisions,
				Expected:  fmt.Sprintf(fmtPolicyField, exp.Action, exp.ExpectedOutcome),
				Actual:    fmt.Sprintf(fmtPolicyField, exp.Action, actual),
				Evidence:  fmt.Sprintf("policy decision for %q: expected %q, got %q", exp.Action, exp.ExpectedOutcome, actual),
			})
		}
	}
	return out
}

func compareToolCalls(expected ScenarioExpected, trace ActualRunTrace) []Mismatch {
	actualNames := stringSet(toolCallNames(trace.ToolCalls))
	forbiddenSet := forbiddenToolSet(expected.ForbiddenToolCalls)
	expectedNames := stringSet(expectedToolNames(expected.ToolCalls))

	var out []Mismatch
	out = append(out, missingRequiredTools(expected.ToolCalls, actualNames)...)
	out = append(out, executedForbiddenTools(trace.ToolCalls, forbiddenSet)...)
	out = append(out, extraToolCalls(trace.ToolCalls, expectedNames, forbiddenSet)...)
	return out
}

func toolCallNames(calls []TraceToolCall) []string {
	names := make([]string, 0, len(calls))
	for _, tc := range calls {
		names = append(names, tc.ToolName)
	}
	return names
}

func expectedToolNames(calls []ExpectedToolCall) []string {
	names := make([]string, 0, len(calls))
	for _, exp := range calls {
		names = append(names, exp.ToolName)
	}
	return names
}

func forbiddenToolSet(forbidden []ForbiddenToolCall) map[string]string {
	out := make(map[string]string, len(forbidden))
	for _, f := range forbidden {
		out[f.ToolName] = f.Reason
	}
	return out
}

func missingRequiredTools(expected []ExpectedToolCall, actualNames map[string]struct{}) []Mismatch {
	var out []Mismatch
	for _, exp := range expected {
		if !exp.Required {
			continue
		}
		if _, ok := actualNames[exp.ToolName]; !ok {
			out = append(out, Mismatch{
				Dimension: DimToolCalls,
				Expected:  exp.ToolName,
				Actual:    "absent",
				Evidence:  fmt.Sprintf("required tool call %q was not executed", exp.ToolName),
			})
		}
	}
	return out
}

func executedForbiddenTools(actual []TraceToolCall, forbiddenSet map[string]string) []Mismatch {
	var out []Mismatch
	for _, tc := range actual {
		if reason, forbidden := forbiddenSet[tc.ToolName]; forbidden {
			out = append(out, Mismatch{
				Dimension: DimForbiddenToolCalls,
				Expected:  fmt.Sprintf(fmtAbsentValue, tc.ToolName),
				Actual:    tc.ToolName,
				Evidence:  fmt.Sprintf("forbidden tool %q was executed: %s", tc.ToolName, reason),
			})
		}
	}
	return out
}

func extraToolCalls(actual []TraceToolCall, expectedNames map[string]struct{}, forbiddenSet map[string]string) []Mismatch {
	var out []Mismatch
	for _, tc := range actual {
		_, inExpected := expectedNames[tc.ToolName]
		_, inForbidden := forbiddenSet[tc.ToolName]
		if !inExpected && !inForbidden {
			out = append(out, Mismatch{
				Dimension: DimExtraToolCalls,
				Expected:  "not expected",
				Actual:    tc.ToolName,
				Evidence:  fmt.Sprintf("unexpected (but not forbidden) tool call %q observed", tc.ToolName),
			})
		}
	}
	return out
}

func compareApprovalBehavior(expected ScenarioExpected, trace ActualRunTrace) []Mismatch {
	if expected.ApprovalBehavior == nil {
		return nil
	}

	var out []Mismatch
	out = append(out, compareApprovalPresence(*expected.ApprovalBehavior, trace.ApprovalEvents)...)
	out = append(out, compareApprovalOutcome(*expected.ApprovalBehavior, trace.ApprovalEvents)...)
	return out
}

func compareApprovalPresence(expected ExpectedApprovalBehavior, events []TraceApprovalEvent) []Mismatch {
	hasApproval := len(events) > 0
	switch {
	case expected.Required && !hasApproval:
		return []Mismatch{{
			Dimension: DimApprovalBehavior,
			Expected:  "approval required",
			Actual:    "no approval events",
			Evidence:  "scenario requires an approval request but trace contains none",
		}}
	case !expected.Required && hasApproval:
		return []Mismatch{{
			Dimension: DimApprovalBehavior,
			Expected:  "no approval required",
			Actual:    fmt.Sprintf("%d approval event(s)", len(events)),
			Evidence:  "trace contains approval events even though scenario does not require approval",
		}}
	default:
		return nil
	}
}

func compareApprovalOutcome(expected ExpectedApprovalBehavior, events []TraceApprovalEvent) []Mismatch {
	if expected.ExpectedOutcome == "" || len(events) == 0 {
		return nil
	}
	if !hasApprovalOutcome(events, expected.ExpectedOutcome) {
		actualStatuses := joinApprovalStatuses(events)
		return []Mismatch{{
			Dimension: DimApprovalBehavior,
			Expected:  expected.ExpectedOutcome,
			Actual:    actualStatuses,
			Evidence:  fmt.Sprintf("expected an approval outcome %q but observed %s", expected.ExpectedOutcome, actualStatuses),
		}}
	}
	return nil
}

func compareFinalState(expected ScenarioExpected, trace ActualRunTrace) []Mismatch {
	if len(expected.FinalState) == 0 {
		return nil
	}

	var actualState map[string]any
	if len(trace.FinalStateRaw) > 0 {
		if err := json.Unmarshal(trace.FinalStateRaw, &actualState); err != nil {
			return []Mismatch{{
				Dimension: DimFinalState,
				Expected:  fmt.Sprintf("%v", expected.FinalState),
				Actual:    "unparseable",
				Evidence:  fmt.Sprintf("could not parse trace final_state JSON: %v", err),
			}}
		}
	}

	var out []Mismatch
	keys := sortedMapKeys(expected.FinalState)
	for _, key := range keys {
		out = append(out, compareFinalStateField(key, expected.FinalState[key], actualState)...)
	}
	return out
}

func compareFinalStateField(key string, expectedValue any, actualState map[string]any) []Mismatch {
	actualValue, found := actualState[key]
	if !found {
		return []Mismatch{{
			Dimension: DimFinalState,
			Expected:  fmt.Sprintf(fmtStateField, key, expectedValue),
			Actual:    "missing",
			Evidence:  fmt.Sprintf("expected final state field %q not present in trace", key),
		}}
	}
	if reflect.DeepEqual(expectedValue, actualValue) {
		return nil
	}
	return []Mismatch{{
		Dimension: DimFinalState,
		Expected:  fmt.Sprintf(fmtStateField, key, expectedValue),
		Actual:    fmt.Sprintf(fmtStateField, key, actualValue),
		Evidence:  fmt.Sprintf("final state field %q: expected %v, got %v", key, expectedValue, actualValue),
	}}
}

func compareContractValidation(trace ActualRunTrace) []Mismatch {
	var out []Mismatch
	if !trace.ContractValidation.MutatorsTraceable {
		out = append(out, Mismatch{
			Dimension: DimContractValidation,
			Expected:  "mutators_traceable=true",
			Actual:    "mutators_traceable=false",
			Evidence:  "trace contract validation reported untraceable executed mutators",
		})
	}
	if !trace.ContractValidation.PolicysTraceable {
		out = append(out, Mismatch{
			Dimension: DimContractValidation,
			Expected:  "policys_traceable=true",
			Actual:    "policys_traceable=false",
			Evidence:  "trace contract validation reported policy decisions without audit traceability",
		})
	}
	return out
}

func compareAuditEvents(expected ScenarioExpected, trace ActualRunTrace) []Mismatch {
	actual := make(map[string]struct{}, len(trace.AuditEvents))
	for _, ae := range trace.AuditEvents {
		actual[ae.Action] = struct{}{}
	}

	var out []Mismatch
	for _, req := range expected.AuditEvents {
		if _, ok := actual[req]; !ok {
			out = append(out, Mismatch{
				Dimension: DimAuditEvents,
				Expected:  req,
				Actual:    "absent",
				Evidence:  fmt.Sprintf("expected audit event %q not found in trace", req),
			})
		}
	}
	return out
}

func hasApprovalOutcome(events []TraceApprovalEvent, want string) bool {
	for _, event := range events {
		if event.Status == want {
			return true
		}
	}
	return false
}

func joinApprovalStatuses(events []TraceApprovalEvent) string {
	statuses := make([]string, 0, len(events))
	for _, event := range events {
		statuses = append(statuses, event.Status)
	}
	sort.Strings(statuses)
	return fmt.Sprintf("%v", statuses)
}

func sortedMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortMismatches(items []Mismatch) {
	sort.Slice(items, func(i, j int) bool {
		left := items[i]
		right := items[j]
		if left.Dimension != right.Dimension {
			return left.Dimension < right.Dimension
		}
		if left.Expected != right.Expected {
			return left.Expected < right.Expected
		}
		if left.Actual != right.Actual {
			return left.Actual < right.Actual
		}
		return left.Evidence < right.Evidence
	})
}

// stringSet converts a slice of strings to a set for O(1) membership testing.
func stringSet(ss []string) map[string]struct{} {
	out := make(map[string]struct{}, len(ss))
	for _, s := range ss {
		out[s] = struct{}{}
	}
	return out
}
