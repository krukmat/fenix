package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"
)

const regressionScoreDeltaFormat = "%s: %.2f -> %.2f"

// RegressionCase binds one golden scenario to the actual run trace to evaluate.
type RegressionCase struct {
	Scenario GoldenScenario `json:"scenario"`
	Trace    ActualRunTrace `json:"trace"`
}

// RegressionRunner evaluates deterministic scenario fixtures as a regression suite.
type RegressionRunner struct {
	Now     func() time.Time
	Weights ScorecardWeights
}

// RegressionScenarioResult captures the full deterministic outcome for one scenario.
type RegressionScenarioResult struct {
	ScenarioID         string              `json:"scenario_id"`
	Title              string              `json:"title"`
	Passed             bool                `json:"passed"`
	Comparator         ComparatorResult    `json:"comparator"`
	Metrics            Metrics             `json:"metrics"`
	Scorecard          Scorecard           `json:"scorecard"`
	HardGateViolations []HardGateViolation `json:"hard_gate_violations,omitempty"`
	HardGateAssessment HardGateAssessment  `json:"hard_gate_assessment"`
	FailureReasons     []string            `json:"failure_reasons,omitempty"`
}

// RegressionSummary aggregates the suite-level regression outcome.
type RegressionSummary struct {
	TotalScenarios          int                 `json:"total_scenarios"`
	PassedScenarios         int                 `json:"passed_scenarios"`
	FailedScenarios         int                 `json:"failed_scenarios"`
	HardGateViolationCount  int                 `json:"hard_gate_violation_count"`
	AverageScore            float64             `json:"average_score"`
	MinScore                float64             `json:"min_score"`
	MaxScore                float64             `json:"max_score"`
	VerdictDistribution     map[Verdict]int     `json:"verdict_distribution"`
	FailedScenarioIDs       []string            `json:"failed_scenario_ids,omitempty"`
	FailedMetricsByScenario map[string][]string `json:"failed_metrics_by_scenario,omitempty"`
}

// RegressionDelta describes how a current report differs from a stored baseline.
type RegressionDelta struct {
	BaselineScenarios   int      `json:"baseline_scenarios"`
	CurrentScenarios    int      `json:"current_scenarios"`
	NewFailures         []string `json:"new_failures,omitempty"`
	ResolvedFailures    []string `json:"resolved_failures,omitempty"`
	ScoreRegressions    []string `json:"score_regressions,omitempty"`
	VerdictRegressions  []string `json:"verdict_regressions,omitempty"`
	HardGateRegressions []string `json:"hard_gate_regressions,omitempty"`
	Regressed           bool     `json:"regressed"`
}

// RegressionReport is the suite-level output suitable for CI and baseline storage.
type RegressionReport struct {
	GeneratedAt time.Time                  `json:"generated_at"`
	Passed      bool                       `json:"passed"`
	Summary     RegressionSummary          `json:"summary"`
	Scenarios   []RegressionScenarioResult `json:"scenarios"`
	Baseline    *RegressionDelta           `json:"baseline,omitempty"`
}

// RegressionBaseline stores the compact snapshot used for future comparisons.
type RegressionBaseline struct {
	GeneratedAt time.Time                             `json:"generated_at"`
	Summary     RegressionSummary                     `json:"summary"`
	Scenarios   map[string]RegressionBaselineScenario `json:"scenarios"`
}

// RegressionBaselineScenario stores comparison-relevant values for one scenario.
type RegressionBaselineScenario struct {
	Passed                 bool    `json:"passed"`
	Score                  float64 `json:"score"`
	Verdict                Verdict `json:"verdict"`
	HardGateViolationCount int     `json:"hard_gate_violation_count"`
}

// Run executes the deterministic regression suite over the provided fixtures.
func (r RegressionRunner) Run(cases []RegressionCase) RegressionReport {
	now := time.Now().UTC()
	if r.Now != nil {
		now = r.Now().UTC()
	}

	results := make([]RegressionScenarioResult, 0, len(cases))
	for _, item := range cases {
		results = append(results, r.runScenario(item))
	}
	sortRegressionScenarioResults(results)

	summary := buildRegressionSummary(results)
	return RegressionReport{
		GeneratedAt: now,
		Passed:      summary.FailedScenarios == 0,
		Summary:     summary,
		Scenarios:   results,
	}
}

// CompareToBaseline attaches a deterministic baseline diff to the current report.
func CompareToBaseline(current RegressionReport, baseline RegressionBaseline) RegressionReport {
	current.Baseline = compareRegressionBaseline(current, baseline)
	return current
}

// ToBaselineSnapshot reduces a report to the fields needed for later comparisons.
func (r RegressionReport) ToBaselineSnapshot() RegressionBaseline {
	scenarios := make(map[string]RegressionBaselineScenario, len(r.Scenarios))
	for _, item := range r.Scenarios {
		scenarios[item.ScenarioID] = RegressionBaselineScenario{
			Passed:                 item.Passed,
			Score:                  item.Scorecard.TotalScore,
			Verdict:                item.HardGateAssessment.FinalVerdict,
			HardGateViolationCount: len(item.HardGateViolations),
		}
	}
	return RegressionBaseline{
		GeneratedAt: r.GeneratedAt,
		Summary:     r.Summary,
		Scenarios:   scenarios,
	}
}

// SaveRegressionBaseline writes a baseline snapshot as indented JSON.
func SaveRegressionBaseline(path string, baseline RegressionBaseline) error {
	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal regression baseline: %w", err)
	}
	writeErr := os.WriteFile(path, append(data, '\n'), 0o600)
	if writeErr != nil {
		return fmt.Errorf("write regression baseline %q: %w", path, writeErr)
	}
	return nil
}

// LoadRegressionBaseline reads a baseline snapshot from JSON on disk.
func LoadRegressionBaseline(path string) (RegressionBaseline, error) {
	data, err := os.ReadFile(path) //nolint:gosec // internal deterministic fixture path
	if err != nil {
		return RegressionBaseline{}, fmt.Errorf("read regression baseline %q: %w", path, err)
	}

	var baseline RegressionBaseline
	parseErr := json.Unmarshal(data, &baseline)
	if parseErr != nil {
		return RegressionBaseline{}, fmt.Errorf("parse regression baseline %q: %w", path, parseErr)
	}
	if baseline.Scenarios == nil {
		baseline.Scenarios = map[string]RegressionBaselineScenario{}
	}
	return baseline, nil
}

func (r RegressionRunner) runScenario(item RegressionCase) RegressionScenarioResult {
	result := Compare(item.Scenario, item.Trace)
	metrics := ComputeMetrics(item.Scenario, item.Trace, result)
	scorecard := NewScorecard(metrics, r.scorecardWeights())
	violations := EvaluateHardGates(item.Scenario, item.Trace, result)
	assessment := ApplyHardGates(scorecard, violations)

	return RegressionScenarioResult{
		ScenarioID:         item.Scenario.ID,
		Title:              item.Scenario.Title,
		Passed:             result.Pass && assessment.FinalVerdict != VerdictFailedValidation,
		Comparator:         result,
		Metrics:            metrics,
		Scorecard:          scorecard,
		HardGateViolations: cloneHardGateViolations(violations),
		HardGateAssessment: assessment,
		FailureReasons:     buildFailureReasons(result, violations),
	}
}

func (r RegressionRunner) scorecardWeights() ScorecardWeights {
	if r.Weights == (ScorecardWeights{}) {
		return DefaultScorecardWeights()
	}
	return r.Weights
}

func buildRegressionSummary(results []RegressionScenarioResult) RegressionSummary {
	summary := RegressionSummary{
		TotalScenarios:          len(results),
		VerdictDistribution:     make(map[Verdict]int, len(results)),
		FailedMetricsByScenario: make(map[string][]string),
	}
	if len(results) == 0 {
		return summary
	}

	populateRegressionSummary(&summary, results)
	return summary
}

func buildFailureReasons(result ComparatorResult, violations []HardGateViolation) []string {
	reasons := make([]string, 0, len(result.Mismatches)+len(violations))
	seen := make(map[string]struct{}, len(result.Mismatches)+len(violations))

	for _, mismatch := range result.Mismatches {
		if mismatch.Dimension == DimExtraToolCalls {
			continue
		}
		reason := string(mismatch.Dimension)
		if _, ok := seen[reason]; ok {
			continue
		}
		seen[reason] = struct{}{}
		reasons = append(reasons, reason)
	}

	for _, violation := range violations {
		reason := "hard_gate:" + violation.Gate
		if _, ok := seen[reason]; ok {
			continue
		}
		seen[reason] = struct{}{}
		reasons = append(reasons, reason)
	}

	sort.Strings(reasons)
	return reasons
}

func compareRegressionBaseline(current RegressionReport, baseline RegressionBaseline) *RegressionDelta {
	delta := &RegressionDelta{
		BaselineScenarios: baseline.Summary.TotalScenarios,
		CurrentScenarios:  current.Summary.TotalScenarios,
	}

	for _, item := range current.Scenarios {
		previous, ok := baseline.Scenarios[item.ScenarioID]
		if !ok {
			appendNewFailureIfNeeded(delta, item)
			continue
		}
		applyRegressionBaselineDiff(delta, item, previous)
	}

	appendResolvedFailures(delta, current.Scenarios, baseline.Scenarios)
	finalizeRegressionDelta(delta)
	return delta
}

func populateRegressionSummary(summary *RegressionSummary, results []RegressionScenarioResult) {
	minScore := results[0].Scorecard.TotalScore
	maxScore := results[0].Scorecard.TotalScore
	totalScore := 0.0

	for _, item := range results {
		totalScore += item.Scorecard.TotalScore
		minScore, maxScore = updateRegressionScoreBounds(minScore, maxScore, item.Scorecard.TotalScore)
		summary.VerdictDistribution[item.HardGateAssessment.FinalVerdict]++
		summary.HardGateViolationCount += len(item.HardGateViolations)
		recordRegressionScenarioOutcome(summary, item)
	}

	sort.Strings(summary.FailedScenarioIDs)
	summary.AverageScore = safeRatio(totalScore, float64(len(results)))
	summary.MinScore = minScore
	summary.MaxScore = maxScore

	if len(summary.FailedMetricsByScenario) == 0 {
		summary.FailedMetricsByScenario = nil
	}
}

func updateRegressionScoreBounds(minScore, maxScore, score float64) (float64, float64) {
	if score < minScore {
		minScore = score
	}
	if score > maxScore {
		maxScore = score
	}
	return minScore, maxScore
}

func recordRegressionScenarioOutcome(summary *RegressionSummary, item RegressionScenarioResult) {
	if item.Passed {
		summary.PassedScenarios++
		return
	}
	summary.FailedScenarios++
	summary.FailedScenarioIDs = append(summary.FailedScenarioIDs, item.ScenarioID)
	summary.FailedMetricsByScenario[item.ScenarioID] = append([]string(nil), item.FailureReasons...)
}

func appendNewFailureIfNeeded(delta *RegressionDelta, item RegressionScenarioResult) {
	if !item.Passed {
		delta.NewFailures = append(delta.NewFailures, item.ScenarioID)
	}
}

func appendResolvedFailures(
	delta *RegressionDelta,
	current []RegressionScenarioResult,
	baseline map[string]RegressionBaselineScenario,
) {
	for scenarioID, previous := range baseline {
		if _, ok := findScenarioResult(current, scenarioID); ok {
			continue
		}
		if !previous.Passed {
			delta.ResolvedFailures = append(delta.ResolvedFailures, scenarioID)
		}
	}
}

func applyRegressionBaselineDiff(
	delta *RegressionDelta,
	item RegressionScenarioResult,
	previous RegressionBaselineScenario,
) {
	appendFailureTransitions(delta, item, previous)
	appendScoreRegression(delta, item, previous)
	appendVerdictRegression(delta, item, previous)
	appendHardGateRegression(delta, item, previous)
}

func appendFailureTransitions(
	delta *RegressionDelta,
	item RegressionScenarioResult,
	previous RegressionBaselineScenario,
) {
	if previous.Passed && !item.Passed {
		delta.NewFailures = append(delta.NewFailures, item.ScenarioID)
	}
	if !previous.Passed && item.Passed {
		delta.ResolvedFailures = append(delta.ResolvedFailures, item.ScenarioID)
	}
}

func appendScoreRegression(
	delta *RegressionDelta,
	item RegressionScenarioResult,
	previous RegressionBaselineScenario,
) {
	if item.Scorecard.TotalScore < previous.Score {
		delta.ScoreRegressions = append(delta.ScoreRegressions,
			fmt.Sprintf(regressionScoreDeltaFormat, item.ScenarioID, previous.Score, item.Scorecard.TotalScore))
	}
}

func appendVerdictRegression(
	delta *RegressionDelta,
	item RegressionScenarioResult,
	previous RegressionBaselineScenario,
) {
	currentVerdict := item.HardGateAssessment.FinalVerdict
	if verdictRank(currentVerdict) > verdictRank(previous.Verdict) {
		delta.VerdictRegressions = append(delta.VerdictRegressions,
			fmt.Sprintf("%s: %s -> %s", item.ScenarioID, previous.Verdict, currentVerdict))
	}
}

func appendHardGateRegression(
	delta *RegressionDelta,
	item RegressionScenarioResult,
	previous RegressionBaselineScenario,
) {
	currentCount := len(item.HardGateViolations)
	if currentCount > previous.HardGateViolationCount {
		delta.HardGateRegressions = append(delta.HardGateRegressions,
			fmt.Sprintf("%s: %d -> %d", item.ScenarioID, previous.HardGateViolationCount, currentCount))
	}
}

func finalizeRegressionDelta(delta *RegressionDelta) {
	sort.Strings(delta.NewFailures)
	sort.Strings(delta.ResolvedFailures)
	sort.Strings(delta.ScoreRegressions)
	sort.Strings(delta.VerdictRegressions)
	sort.Strings(delta.HardGateRegressions)

	delta.Regressed = len(delta.NewFailures) > 0 ||
		len(delta.ScoreRegressions) > 0 ||
		len(delta.VerdictRegressions) > 0 ||
		len(delta.HardGateRegressions) > 0
}

func sortRegressionScenarioResults(items []RegressionScenarioResult) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].ScenarioID != items[j].ScenarioID {
			return items[i].ScenarioID < items[j].ScenarioID
		}
		return items[i].Title < items[j].Title
	})
}

func verdictRank(v Verdict) int {
	switch v {
	case VerdictPass:
		return 0
	case VerdictPassWithWarnings:
		return 1
	case VerdictRequiresReview:
		return 2
	case VerdictFail:
		return 3
	case VerdictFailedValidation:
		return 4
	default:
		return 5
	}
}

func findScenarioResult(items []RegressionScenarioResult, scenarioID string) (RegressionScenarioResult, bool) {
	for _, item := range items {
		if item.ScenarioID == scenarioID {
			return item, true
		}
	}
	return RegressionScenarioResult{}, false
}
