package eval

import (
	"encoding/json"
	"math"
	"sort"
)

// Verdict is the scorecard classification for a deterministic eval run.
type Verdict string

const (
	VerdictPass             Verdict = "pass"
	VerdictPassWithWarnings Verdict = "pass_with_warnings"
	VerdictRequiresReview   Verdict = "requires_review"
	VerdictFail             Verdict = "fail"
	VerdictFailedValidation Verdict = "failed_validation"
)

// Metrics contains the reproducible quantitative measurements for one scenario/run comparison.
type Metrics struct {
	OutcomeAccuracy         float64 `json:"outcome_accuracy"`
	ToolCallPrecision       float64 `json:"tool_call_precision"`
	ToolCallRecall          float64 `json:"tool_call_recall"`
	ToolCallF1              float64 `json:"tool_call_f1"`
	ForbiddenToolViolations int     `json:"forbidden_tool_violations"`
	PolicyCompliance        float64 `json:"policy_compliance"`
	ApprovalAccuracy        float64 `json:"approval_accuracy"`
	EvidenceCoverage        float64 `json:"evidence_coverage"`
	ForbiddenEvidenceCount  int     `json:"forbidden_evidence_count"`
	StateMutationAccuracy   float64 `json:"state_mutation_accuracy"`
	AuditCompleteness       float64 `json:"audit_completeness"`
	ContractValidity        float64 `json:"contract_validity"`
	AbstentionAccuracy      float64 `json:"abstention_accuracy"`
	LatencyCompliance       float64 `json:"latency_compliance"`
	ToolBudgetCompliance    float64 `json:"tool_budget_compliance"`
}

// ScorecardWeights configures the weighted scorecard dimensions.
type ScorecardWeights struct {
	FinalOutcome        float64 `json:"final_outcome"`
	ToolCorrectness     float64 `json:"tool_correctness"`
	PolicyCompliance    float64 `json:"policy_compliance"`
	EvidenceGrounding   float64 `json:"evidence_grounding"`
	ApprovalCorrectness float64 `json:"approval_correctness"`
	StateMutation       float64 `json:"state_mutation"`
	AuditCompleteness   float64 `json:"audit_completeness"`
	ContractValidity    float64 `json:"contract_validity"`
}

// Scorecard is the weighted aggregate over the deterministic metrics.
type Scorecard struct {
	Metrics    Metrics          `json:"metrics"`
	Weights    ScorecardWeights `json:"weights"`
	TotalScore float64          `json:"total_score"`
	Verdict    Verdict          `json:"verdict"`
}

// DefaultScorecardWeights returns the F4 default weighting.
func DefaultScorecardWeights() ScorecardWeights {
	return ScorecardWeights{
		FinalOutcome:        20,
		ToolCorrectness:     15,
		PolicyCompliance:    20,
		EvidenceGrounding:   15,
		ApprovalCorrectness: 10,
		StateMutation:       10,
		AuditCompleteness:   5,
		ContractValidity:    5,
	}
}

// ComputeMetrics derives all deterministic F4 metrics from the structured scenario, trace, and comparison result.
func ComputeMetrics(scenario GoldenScenario, trace ActualRunTrace, result ComparatorResult) Metrics {
	actualState := decodeFinalState(trace.FinalStateRaw)
	return Metrics{
		OutcomeAccuracy:         outcomeAccuracy(result),
		ToolCallPrecision:       toolCallPrecision(scenario.Expected.ToolCalls, trace.ToolCalls),
		ToolCallRecall:          toolCallRecall(scenario.Expected.ToolCalls, trace.ToolCalls),
		ToolCallF1:              toolCallF1(scenario.Expected.ToolCalls, trace.ToolCalls),
		ForbiddenToolViolations: forbiddenToolViolations(scenario.Expected.ForbiddenToolCalls, trace.ToolCalls),
		PolicyCompliance:        policyCompliance(scenario.Expected.PolicyDecisions, trace.PolicyDecisions),
		ApprovalAccuracy:        approvalAccuracy(scenario.Expected.ApprovalBehavior, trace.ApprovalEvents),
		EvidenceCoverage:        evidenceCoverage(scenario.Expected.RequiredEvidence, trace.EvidenceSources),
		ForbiddenEvidenceCount:  forbiddenEvidenceCount(scenario.Expected.ForbiddenEvidence, trace.EvidenceSources),
		StateMutationAccuracy:   stateMutationAccuracy(scenario.Expected.FinalState, actualState),
		AuditCompleteness:       auditCompleteness(scenario.Expected.AuditEvents, trace.AuditEvents),
		ContractValidity:        contractValidity(trace.ContractValidation),
		AbstentionAccuracy:      abstentionAccuracy(scenario.Expected, trace),
		LatencyCompliance:       latencyCompliance(scenario.Thresholds.MaxLatencyMs, trace.LatencyMs),
		ToolBudgetCompliance:    toolBudgetCompliance(scenario.Thresholds.MaxToolCalls, trace.ToolCalls),
	}
}

// NewScorecard computes the weighted aggregate score and verdict for a metrics set.
func NewScorecard(metrics Metrics, weights ScorecardWeights) Scorecard {
	score := weightedScore(metrics, weights)
	return Scorecard{
		Metrics:    metrics,
		Weights:    weights,
		TotalScore: score,
		Verdict:    ComputeVerdict(score),
	}
}

// DefaultScorecard computes the scorecard using the F4 default weights.
func DefaultScorecard(metrics Metrics) Scorecard {
	return NewScorecard(metrics, DefaultScorecardWeights())
}

// ComputeVerdict maps a numeric score into the F4 verdict bands.
func ComputeVerdict(score float64) Verdict {
	switch {
	case score >= 90:
		return VerdictPass
	case score >= 75:
		return VerdictPassWithWarnings
	case score >= 60:
		return VerdictRequiresReview
	default:
		return VerdictFail
	}
}

func outcomeAccuracy(result ComparatorResult) float64 {
	if hasMismatchInDimension(result.Mismatches, DimFinalOutcome) {
		return 0
	}
	return 1
}

func toolCallPrecision(expected []ExpectedToolCall, actual []TraceToolCall) float64 {
	actualTotal := len(actual)
	if actualTotal == 0 {
		return ratioWhenZeroExpected(len(expected))
	}
	return safeRatio(float64(matchedExpectedToolCalls(expected, actual)), float64(actualTotal))
}

func toolCallRecall(expected []ExpectedToolCall, actual []TraceToolCall) float64 {
	expectedTotal := len(expected)
	if expectedTotal == 0 {
		return 1
	}
	return safeRatio(float64(matchedExpectedToolCalls(expected, actual)), float64(expectedTotal))
}

func toolCallF1(expected []ExpectedToolCall, actual []TraceToolCall) float64 {
	precision := toolCallPrecision(expected, actual)
	recall := toolCallRecall(expected, actual)
	if precision == 0 && recall == 0 {
		return 0
	}
	return safeRatio(2*precision*recall, precision+recall)
}

func matchedExpectedToolCalls(expected []ExpectedToolCall, actual []TraceToolCall) int {
	expectedCounts := toolNameCounts(expectedToolNames(expected))
	actualCounts := toolNameCounts(toolCallNames(actual))
	return matchedCounts(expectedCounts, actualCounts)
}

func forbiddenToolViolations(forbidden []ForbiddenToolCall, actual []TraceToolCall) int {
	forbiddenSet := forbiddenToolSet(forbidden)
	return countMatchingToolCalls(actual, forbiddenSet)
}

func policyCompliance(expected []ExpectedPolicyDecision, actual []TracePolicyDecision) float64 {
	if len(expected) == 0 {
		return 1
	}
	actualMap := make(map[string]string, len(actual))
	for _, policy := range actual {
		actualMap[policy.Action] = policy.Outcome
	}
	matched := 0
	for _, policy := range expected {
		if actualMap[policy.Action] == policy.ExpectedOutcome {
			matched++
		}
	}
	return safeRatio(float64(matched), float64(len(expected)))
}

func approvalAccuracy(expected *ExpectedApprovalBehavior, actual []TraceApprovalEvent) float64 {
	if expected == nil {
		if len(actual) == 0 {
			return 1
		}
		return 0
	}
	if expected.Required != (len(actual) > 0) {
		return 0
	}
	if expected.ExpectedOutcome == "" || hasApprovalOutcome(actual, expected.ExpectedOutcome) {
		return 1
	}
	return 0
}

func evidenceCoverage(required []string, actual []string) float64 {
	if len(required) == 0 {
		return 1
	}
	actualSet := stringSet(actual)
	matched := 0
	for _, id := range required {
		if _, ok := actualSet[id]; ok {
			matched++
		}
	}
	return safeRatio(float64(matched), float64(len(required)))
}

func forbiddenEvidenceCount(forbidden []string, actual []string) int {
	forbiddenSet := stringSet(forbidden)
	actualSet := stringSet(actual)
	count := 0
	for source := range forbiddenSet {
		if _, ok := actualSet[source]; ok {
			count++
		}
	}
	return count
}

func stateMutationAccuracy(expected map[string]any, actual map[string]any) float64 {
	if len(expected) == 0 {
		return 1
	}
	matched := 0
	for key, expectedValue := range expected {
		if reflectEqualMapValue(actual, key, expectedValue) {
			matched++
		}
	}
	return safeRatio(float64(matched), float64(len(expected)))
}

func auditCompleteness(required []string, actual []TraceAuditEvent) float64 {
	if len(required) == 0 {
		return 1
	}
	present := 0
	actualSet := make(map[string]struct{}, len(actual))
	for _, event := range actual {
		actualSet[event.Action] = struct{}{}
	}
	for _, action := range required {
		if _, ok := actualSet[action]; ok {
			present++
		}
	}
	return safeRatio(float64(present), float64(len(required)))
}

func contractValidity(validation TraceContractValidation) float64 {
	validCount := 0
	if validation.MutatorsTraceable {
		validCount++
	}
	if validation.PolicysTraceable {
		validCount++
	}
	return safeRatio(float64(validCount), 2)
}

func abstentionAccuracy(expected ScenarioExpected, trace ActualRunTrace) float64 {
	actuallyAbstained := trace.FinalOutcome == "abstained"
	if expected.ShouldAbstain == actuallyAbstained {
		return 1
	}
	return 0
}

func latencyCompliance(maxLatencyMs int, actualLatencyMs *int64) float64 {
	if maxLatencyMs <= 0 {
		return 1
	}
	if actualLatencyMs == nil {
		return 0
	}
	if *actualLatencyMs <= int64(maxLatencyMs) {
		return 1
	}
	return 0
}

func toolBudgetCompliance(maxToolCalls int, actual []TraceToolCall) float64 {
	if maxToolCalls <= 0 || len(actual) <= maxToolCalls {
		return 1
	}
	return 0
}

func weightedScore(metrics Metrics, weights ScorecardWeights) float64 {
	score := 0.0
	score += metrics.OutcomeAccuracy * weights.FinalOutcome
	score += metrics.ToolCallF1 * weights.ToolCorrectness
	score += metrics.PolicyCompliance * weights.PolicyCompliance
	score += metrics.EvidenceCoverage * weights.EvidenceGrounding
	score += metrics.ApprovalAccuracy * weights.ApprovalCorrectness
	score += metrics.StateMutationAccuracy * weights.StateMutation
	score += metrics.AuditCompleteness * weights.AuditCompleteness
	score += metrics.ContractValidity * weights.ContractValidity
	return roundToTwoDecimals(score)
}

func decodeFinalState(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

func reflectEqualMapValue(actual map[string]any, key string, expected any) bool {
	value, ok := actual[key]
	return ok && valuesEqual(expected, value)
}

func valuesEqual(expected, actual any) bool {
	expectedJSON, expectedErr := json.Marshal(expected)
	actualJSON, actualErr := json.Marshal(actual)
	return expectedErr == nil && actualErr == nil && string(expectedJSON) == string(actualJSON)
}

func safeRatio(numerator, denominator float64) float64 {
	if denominator == 0 {
		return 0
	}
	return numerator / denominator
}

func ratioWhenZeroExpected(expectedTotal int) float64 {
	if expectedTotal == 0 {
		return 1
	}
	return 0
}

func toolNameCounts(names []string) map[string]int {
	counts := make(map[string]int, len(names))
	for _, name := range names {
		counts[name]++
	}
	return counts
}

func matchedCounts(expectedCounts, actualCounts map[string]int) int {
	keys := sortedToolNames(expectedCounts)
	matched := 0
	for _, key := range keys {
		matched += minInt(expectedCounts[key], actualCounts[key])
	}
	return matched
}

func sortedToolNames(items map[string]int) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func countMatchingToolCalls(actual []TraceToolCall, forbidden map[string]string) int {
	count := 0
	for _, toolCall := range actual {
		if _, ok := forbidden[toolCall.ToolName]; ok {
			count++
		}
	}
	return count
}

func hasMismatchInDimension(mismatches []Mismatch, dimension MismatchDimension) bool {
	for _, mismatch := range mismatches {
		if mismatch.Dimension == dimension {
			return true
		}
	}
	return false
}

func roundToTwoDecimals(value float64) float64 {
	return math.Round(value*100) / 100
}
