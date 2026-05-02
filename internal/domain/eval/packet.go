package eval

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ReviewPacketStatus classifies deterministic packet checks.
type ReviewPacketStatus string

const (
	ReviewPacketStatusPass ReviewPacketStatus = "pass"
	ReviewPacketStatusFail ReviewPacketStatus = "fail"
	ReviewPacketStatusInfo ReviewPacketStatus = "info"
)

const (
	packetValuePresent = "present"
	packetValueAbsent  = "absent"
	packetFmtInt       = "%d"
)

// ReviewPacket is the human-readable deterministic projection of one eval result.
type ReviewPacket struct {
	Scenario   ReviewPacketScenario   `json:"scenario"`
	Run        ReviewPacketRun        `json:"run"`
	Evaluation ReviewPacketEvaluation `json:"evaluation"`
	Comparison ReviewPacketComparison `json:"comparison"`
}

// ReviewPacketScenario carries stable scenario metadata.
type ReviewPacketScenario struct {
	ID          string             `json:"id"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Domain      string             `json:"domain"`
	Tags        []string           `json:"tags,omitempty"`
	InputEvent  string             `json:"input_event"`
	Thresholds  ScenarioThresholds `json:"thresholds"`
}

// ReviewPacketRun carries stable actual-run metadata.
type ReviewPacketRun struct {
	RunID              string                  `json:"run_id"`
	WorkspaceID        string                  `json:"workspace_id"`
	AgentDefinitionID  string                  `json:"agent_definition_id"`
	ScenarioID         string                  `json:"scenario_id,omitempty"`
	TriggerType        string                  `json:"trigger_type"`
	FinalOutcome       string                  `json:"final_outcome"`
	LatencyMs          *int64                  `json:"latency_ms,omitempty"`
	TotalTokens        *int64                  `json:"total_tokens,omitempty"`
	TotalCost          *float64                `json:"total_cost,omitempty"`
	Retries            int                     `json:"retries"`
	StartedAt          time.Time               `json:"started_at"`
	CompletedAt        *time.Time              `json:"completed_at,omitempty"`
	ContractValidation TraceContractValidation `json:"contract_validation"`
}

// ReviewPacketEvaluation contains deterministic score, verdict, and gate outcome.
type ReviewPacketEvaluation struct {
	ComparatorPass     bool                `json:"comparator_pass"`
	MismatchCount      int                 `json:"mismatch_count"`
	ScorecardVerdict   Verdict             `json:"scorecard_verdict"`
	FinalVerdict       Verdict             `json:"final_verdict"`
	TotalScore         float64             `json:"total_score"`
	HardGateFailed     bool                `json:"hard_gate_failed"`
	HardGateViolations []HardGateViolation `json:"hard_gate_violations,omitempty"`
	DeniedActions      []ReviewPacketDeniedAction `json:"denied_actions,omitempty"`
	Metrics            Metrics             `json:"metrics"`
	Mismatches         []Mismatch          `json:"mismatches,omitempty"`
}

type ReviewPacketDeniedAction struct {
	ActorID   string    `json:"actor_id"`
	Action    string    `json:"action"`
	Target    string    `json:"target,omitempty"`
	Policy    string    `json:"policy,omitempty"`
	Reason    string    `json:"reason,omitempty"`
	Outcome   string    `json:"outcome"`
	Timestamp time.Time `json:"timestamp"`
}

// ReviewPacketComparison groups expected-vs-actual deterministic checks.
type ReviewPacketComparison struct {
	FinalOutcome       []ReviewPacketCheck `json:"final_outcome"`
	PolicyDecisions    []ReviewPacketCheck `json:"policy_decisions"`
	RequiredEvidence   []ReviewPacketCheck `json:"required_evidence"`
	ForbiddenEvidence  []ReviewPacketCheck `json:"forbidden_evidence"`
	ToolCalls          []ReviewPacketCheck `json:"tool_calls"`
	ApprovalBehavior   []ReviewPacketCheck `json:"approval_behavior,omitempty"`
	FinalState         []ReviewPacketCheck `json:"final_state"`
	AuditEvents        []ReviewPacketCheck `json:"audit_events"`
	ContractValidation []ReviewPacketCheck `json:"contract_validation"`
}

// ReviewPacketCheck is one expected-vs-actual deterministic comparison row.
type ReviewPacketCheck struct {
	Dimension string             `json:"dimension"`
	Label     string             `json:"label"`
	Expected  string             `json:"expected"`
	Actual    string             `json:"actual"`
	Status    ReviewPacketStatus `json:"status"`
	Evidence  string             `json:"evidence,omitempty"`
}

// BuildReviewPacket assembles the deterministic review packet for one scenario/run.
func BuildReviewPacket(
	scenario GoldenScenario,
	trace ActualRunTrace,
	result ComparatorResult,
	assessment HardGateAssessment,
) ReviewPacket {
	return ReviewPacket{
		Scenario: ReviewPacketScenario{
			ID:          scenario.ID,
			Title:       scenario.Title,
			Description: scenario.Description,
			Domain:      scenario.Domain,
			Tags:        append([]string(nil), scenario.Tags...),
			InputEvent:  scenario.InputEvent.Type,
			Thresholds:  scenario.Thresholds,
		},
		Run: ReviewPacketRun{
			RunID:              trace.RunID,
			WorkspaceID:        trace.WorkspaceID,
			AgentDefinitionID:  trace.AgentDefinitionID,
			ScenarioID:         trace.ScenarioID,
			TriggerType:        trace.TriggerType,
			FinalOutcome:       trace.FinalOutcome,
			LatencyMs:          trace.LatencyMs,
			TotalTokens:        trace.TotalTokens,
			TotalCost:          trace.TotalCost,
			Retries:            trace.Retries,
			StartedAt:          trace.StartedAt,
			CompletedAt:        trace.CompletedAt,
			ContractValidation: trace.ContractValidation,
		},
		Evaluation: ReviewPacketEvaluation{
			ComparatorPass:     result.Pass,
			MismatchCount:      len(result.Mismatches),
			ScorecardVerdict:   assessment.Scorecard.Verdict,
			FinalVerdict:       assessment.FinalVerdict,
			TotalScore:         assessment.Scorecard.TotalScore,
			HardGateFailed:     len(assessment.Violations) > 0,
			HardGateViolations: cloneHardGateViolations(assessment.Violations),
			DeniedActions:      buildDeniedActionSummaries(trace),
			Metrics:            assessment.Scorecard.Metrics,
			Mismatches:         append([]Mismatch(nil), result.Mismatches...),
		},
		Comparison: ReviewPacketComparison{
			FinalOutcome:       buildFinalOutcomeChecks(scenario.Expected, trace),
			PolicyDecisions:    buildPolicyDecisionChecks(scenario.Expected, trace),
			RequiredEvidence:   buildRequiredEvidenceChecks(scenario.Expected, trace),
			ForbiddenEvidence:  buildForbiddenEvidenceChecks(scenario.Expected, trace),
			ToolCalls:          buildToolCallChecks(scenario.Expected, trace),
			ApprovalBehavior:   buildApprovalChecks(scenario.Expected, trace),
			FinalState:         buildFinalStateChecks(scenario.Expected, trace),
			AuditEvents:        buildAuditEventChecks(scenario.Expected, trace),
			ContractValidation: buildContractValidationChecks(trace),
		},
	}
}

// ToJSON returns the packet as indented JSON.
func (p ReviewPacket) ToJSON() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

// ToMarkdown returns the packet as human-readable markdown.
func (p ReviewPacket) ToMarkdown() string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "# Review Packet\n\n")
	writeScenarioMarkdown(&buf, p.Scenario)
	writeRunMarkdown(&buf, p.Run)
	writeEvaluationMarkdown(&buf, p.Evaluation)
	writeExpectedActualMarkdown(&buf, p.Comparison)

	return buf.String()
}

func buildFinalOutcomeChecks(expected ScenarioExpected, trace ActualRunTrace) []ReviewPacketCheck {
	if expected.FinalOutcome == "" {
		return nil
	}
	status := ReviewPacketStatusPass
	evidence := fmt.Sprintf("trace.final_outcome=%s", trace.FinalOutcome)
	if trace.FinalOutcome != expected.FinalOutcome {
		status = ReviewPacketStatusFail
		evidence = fmt.Sprintf("expected final_outcome %q, got %q", expected.FinalOutcome, trace.FinalOutcome)
	}
	return []ReviewPacketCheck{{
		Dimension: string(DimFinalOutcome),
		Label:     "final_outcome",
		Expected:  expected.FinalOutcome,
		Actual:    trace.FinalOutcome,
		Status:    status,
		Evidence:  evidence,
	}}
}

func buildPolicyDecisionChecks(expected ScenarioExpected, trace ActualRunTrace) []ReviewPacketCheck {
	actual := make(map[string]string, len(trace.PolicyDecisions))
	for _, item := range trace.PolicyDecisions {
		actual[item.Action] = item.Outcome
	}

	checks := make([]ReviewPacketCheck, 0, len(expected.PolicyDecisions))
	for _, item := range expected.PolicyDecisions {
		actualOutcome, ok := actual[item.Action]
		status := ReviewPacketStatusPass
		evidence := fmt.Sprintf("trace.policy_decisions[%s]=%s", item.Action, actualOutcome)
		if !ok {
			actualOutcome = textContractActualMissing
			status = ReviewPacketStatusFail
			evidence = fmt.Sprintf("expected policy decision %q missing", item.Action)
		} else if actualOutcome != item.ExpectedOutcome {
			status = ReviewPacketStatusFail
			evidence = fmt.Sprintf("expected policy outcome %q, got %q", item.ExpectedOutcome, actualOutcome)
		}
		checks = append(checks, ReviewPacketCheck{
			Dimension: string(DimPolicyDecisions),
			Label:     item.Action,
			Expected:  item.ExpectedOutcome,
			Actual:    actualOutcome,
			Status:    status,
			Evidence:  evidence,
		})
	}
	return checks
}

func buildRequiredEvidenceChecks(expected ScenarioExpected, trace ActualRunTrace) []ReviewPacketCheck {
	actual := stringSet(trace.EvidenceSources)
	checks := make([]ReviewPacketCheck, 0, len(expected.RequiredEvidence))
	for _, item := range expected.RequiredEvidence {
		status := ReviewPacketStatusPass
		actualValue := packetValuePresent
		evidence := fmt.Sprintf("trace.evidence_sources contains %q", item)
		if _, ok := actual[item]; !ok {
			status = ReviewPacketStatusFail
			actualValue = textContractActualMissing
			evidence = fmt.Sprintf("required evidence %q not found", item)
		}
		checks = append(checks, ReviewPacketCheck{
			Dimension: string(DimRequiredEvidence),
			Label:     item,
			Expected:  packetValuePresent,
			Actual:    actualValue,
			Status:    status,
			Evidence:  evidence,
		})
	}
	return checks
}

func buildForbiddenEvidenceChecks(expected ScenarioExpected, trace ActualRunTrace) []ReviewPacketCheck {
	actual := stringSet(trace.EvidenceSources)
	checks := make([]ReviewPacketCheck, 0, len(expected.ForbiddenEvidence))
	for _, item := range expected.ForbiddenEvidence {
		status := ReviewPacketStatusPass
		actualValue := packetValueAbsent
		evidence := fmt.Sprintf("trace.evidence_sources excludes %q", item)
		if _, ok := actual[item]; ok {
			status = ReviewPacketStatusFail
			actualValue = packetValuePresent
			evidence = fmt.Sprintf("forbidden evidence %q was used", item)
		}
		checks = append(checks, ReviewPacketCheck{
			Dimension: string(DimForbiddenEvidence),
			Label:     item,
			Expected:  packetValueAbsent,
			Actual:    actualValue,
			Status:    status,
			Evidence:  evidence,
		})
	}
	return checks
}

func buildToolCallChecks(expected ScenarioExpected, trace ActualRunTrace) []ReviewPacketCheck {
	expectedSet := stringSet(expectedToolNames(expected.ToolCalls))
	forbiddenSet := forbiddenToolSet(expected.ForbiddenToolCalls)
	actualCounts := toolNameCounts(toolCallNames(trace.ToolCalls))

	checks := make([]ReviewPacketCheck, 0, len(expected.ToolCalls)+len(trace.ToolCalls))
	checks = append(checks, buildExpectedToolCallChecks(expected.ToolCalls, actualCounts)...)
	checks = append(checks, buildObservedToolCallChecks(trace.ToolCalls, expectedSet, forbiddenSet)...)

	sortReviewPacketChecks(checks)
	return checks
}

func buildApprovalChecks(expected ScenarioExpected, trace ActualRunTrace) []ReviewPacketCheck {
	if expected.ApprovalBehavior == nil {
		return nil
	}

	checks := make([]ReviewPacketCheck, 0, 2)
	approvalCount := len(trace.ApprovalEvents)
	presenceStatus := ReviewPacketStatusPass
	presenceActual := packetValueAbsent
	if approvalCount > 0 {
		presenceActual = packetValuePresent
	}
	presenceEvidence := fmt.Sprintf("trace.approval_events=%d", approvalCount)
	if expected.ApprovalBehavior.Required != (approvalCount > 0) {
		presenceStatus = ReviewPacketStatusFail
		presenceEvidence = fmt.Sprintf("expected approval presence=%t, got %d event(s)", expected.ApprovalBehavior.Required, approvalCount)
	}
	checks = append(checks, ReviewPacketCheck{
		Dimension: string(DimApprovalBehavior),
		Label:     "approval_presence",
		Expected:  boolToPresence(expected.ApprovalBehavior.Required),
		Actual:    presenceActual,
		Status:    presenceStatus,
		Evidence:  presenceEvidence,
	})

	if expected.ApprovalBehavior.ExpectedOutcome == "" {
		return checks
	}

	outcomeStatus := ReviewPacketStatusPass
	outcomeActual := joinApprovalStatuses(trace.ApprovalEvents)
	if outcomeActual == "[]" {
		outcomeActual = "none"
	}
	outcomeEvidence := fmt.Sprintf("trace.approval_statuses=%s", outcomeActual)
	if !hasApprovalOutcome(trace.ApprovalEvents, expected.ApprovalBehavior.ExpectedOutcome) {
		outcomeStatus = ReviewPacketStatusFail
		outcomeEvidence = fmt.Sprintf("expected approval outcome %q, got %s", expected.ApprovalBehavior.ExpectedOutcome, outcomeActual)
	}
	checks = append(checks, ReviewPacketCheck{
		Dimension: string(DimApprovalBehavior),
		Label:     "approval_outcome",
		Expected:  expected.ApprovalBehavior.ExpectedOutcome,
		Actual:    outcomeActual,
		Status:    outcomeStatus,
		Evidence:  outcomeEvidence,
	})

	return checks
}

func buildFinalStateChecks(expected ScenarioExpected, trace ActualRunTrace) []ReviewPacketCheck {
	actualState := decodeFinalState(trace.FinalStateRaw)
	keys := sortedMapKeys(expected.FinalState)
	checks := make([]ReviewPacketCheck, 0, len(keys))
	for _, key := range keys {
		expectedValue := marshalValue(expected.FinalState[key])
		actualValue, ok := actualState[key]
		status := ReviewPacketStatusPass
		actualText := textContractActualMissing
		evidence := fmt.Sprintf("trace.final_state[%s]=%s", key, expectedValue)
		if ok {
			actualText = marshalValue(actualValue)
		}
		if !reflectEqualMapValue(actualState, key, expected.FinalState[key]) {
			status = ReviewPacketStatusFail
			evidence = fmt.Sprintf("expected final_state[%s]=%s, got %s", key, expectedValue, actualText)
		}
		checks = append(checks, ReviewPacketCheck{
			Dimension: string(DimFinalState),
			Label:     key,
			Expected:  expectedValue,
			Actual:    actualText,
			Status:    status,
			Evidence:  evidence,
		})
	}
	return checks
}

func buildAuditEventChecks(expected ScenarioExpected, trace ActualRunTrace) []ReviewPacketCheck {
	actual := make(map[string]struct{}, len(trace.AuditEvents))
	for _, item := range trace.AuditEvents {
		actual[item.Action] = struct{}{}
	}
	checks := make([]ReviewPacketCheck, 0, len(expected.AuditEvents))
	for _, item := range expected.AuditEvents {
		status := ReviewPacketStatusPass
		actualValue := packetValuePresent
		evidence := fmt.Sprintf("trace.audit_events contains %q", item)
		if _, ok := actual[item]; !ok {
			status = ReviewPacketStatusFail
			actualValue = textContractActualMissing
			evidence = fmt.Sprintf("required audit event %q not found", item)
		}
		checks = append(checks, ReviewPacketCheck{
			Dimension: string(DimAuditEvents),
			Label:     item,
			Expected:  packetValuePresent,
			Actual:    actualValue,
			Status:    status,
			Evidence:  evidence,
		})
	}
	return checks
}

func buildContractValidationChecks(trace ActualRunTrace) []ReviewPacketCheck {
	return []ReviewPacketCheck{
		{
			Dimension: string(DimContractValidation),
			Label:     "mutators_traceable",
			Expected:  "true",
			Actual:    fmt.Sprintf("%t", trace.ContractValidation.MutatorsTraceable),
			Status:    boolToPacketStatus(trace.ContractValidation.MutatorsTraceable),
			Evidence:  "trace.contract_validation.mutators_traceable",
		},
		{
			Dimension: string(DimContractValidation),
			Label:     "policys_traceable",
			Expected:  "true",
			Actual:    fmt.Sprintf("%t", trace.ContractValidation.PolicysTraceable),
			Status:    boolToPacketStatus(trace.ContractValidation.PolicysTraceable),
			Evidence:  "trace.contract_validation.policys_traceable",
		},
	}
}

func writeScenarioMarkdown(buf *bytes.Buffer, scenario ReviewPacketScenario) {
	fmt.Fprintf(buf, "## Scenario\n\n")
	fmt.Fprintf(buf, "- Scenario ID: `%s`\n", scenario.ID)
	fmt.Fprintf(buf, "- Title: %s\n", safeMarkdownText(scenario.Title))
	fmt.Fprintf(buf, "- Domain: `%s`\n", scenario.Domain)
	fmt.Fprintf(buf, "- Input event: `%s`\n", scenario.InputEvent)
	if len(scenario.Tags) > 0 {
		fmt.Fprintf(buf, "- Tags: `%s`\n", strings.Join(scenario.Tags, "`, `"))
	}
	if scenario.Description != "" {
		fmt.Fprintf(buf, "- Description: %s\n", safeMarkdownText(scenario.Description))
	}
	fmt.Fprintf(buf, "- Thresholds: min_score=%d, max_latency_ms=%d, max_tool_calls=%d, max_retries=%d\n\n",
		scenario.Thresholds.MinScore,
		scenario.Thresholds.MaxLatencyMs,
		scenario.Thresholds.MaxToolCalls,
		scenario.Thresholds.MaxRetries,
	)
}

func writeRunMarkdown(buf *bytes.Buffer, run ReviewPacketRun) {
	fmt.Fprintf(buf, "## Run\n\n")
	fmt.Fprintf(buf, "- Run ID: `%s`\n", run.RunID)
	fmt.Fprintf(buf, "- Workspace ID: `%s`\n", run.WorkspaceID)
	if run.AgentDefinitionID != "" {
		fmt.Fprintf(buf, "- Agent definition ID: `%s`\n", run.AgentDefinitionID)
	}
	if run.ScenarioID != "" {
		fmt.Fprintf(buf, "- Trace scenario ID: `%s`\n", run.ScenarioID)
	}
	if run.TriggerType != "" {
		fmt.Fprintf(buf, "- Trigger type: `%s`\n", run.TriggerType)
	}
	fmt.Fprintf(buf, "- Final outcome: `%s`\n", run.FinalOutcome)
	fmt.Fprintf(buf, "- Retries: %d\n", run.Retries)
	writeOptionalRunMetricsMarkdown(buf, run)
	fmt.Fprintf(buf, "\n")
}

func writeOptionalRunMetricsMarkdown(buf *bytes.Buffer, run ReviewPacketRun) {
	if run.LatencyMs != nil {
		fmt.Fprintf(buf, "- Latency ms: %d\n", *run.LatencyMs)
	}
	if run.TotalTokens != nil {
		fmt.Fprintf(buf, "- Total tokens: %d\n", *run.TotalTokens)
	}
	if run.TotalCost != nil {
		fmt.Fprintf(buf, "- Total cost: %.6f\n", *run.TotalCost)
	}
	if !run.StartedAt.IsZero() {
		fmt.Fprintf(buf, "- Started at: `%s`\n", run.StartedAt.UTC().Format(time.RFC3339))
	}
	if run.CompletedAt != nil {
		fmt.Fprintf(buf, "- Completed at: `%s`\n", run.CompletedAt.UTC().Format(time.RFC3339))
	}
}

func writeEvaluationMarkdown(buf *bytes.Buffer, evaluation ReviewPacketEvaluation) {
	fmt.Fprintf(buf, "## Evaluation\n\n")
	fmt.Fprintf(buf, "- Comparator pass: `%t`\n", evaluation.ComparatorPass)
	fmt.Fprintf(buf, "- Total score: `%.2f`\n", evaluation.TotalScore)
	fmt.Fprintf(buf, "- Scorecard verdict: `%s`\n", evaluation.ScorecardVerdict)
	fmt.Fprintf(buf, "- Final verdict: `%s`\n", evaluation.FinalVerdict)
	fmt.Fprintf(buf, "- Mismatch count: `%d`\n", evaluation.MismatchCount)
	fmt.Fprintf(buf, "- Hard gate failed: `%t`\n\n", evaluation.HardGateFailed)

	fmt.Fprintf(buf, "## Hard Gates\n\n")
	if len(evaluation.HardGateViolations) == 0 {
		fmt.Fprintf(buf, "_None_\n\n")
	} else {
		writeHardGateTable(buf, evaluation.HardGateViolations)
	}

	fmt.Fprintf(buf, "## Denied Actions\n\n")
	if len(evaluation.DeniedActions) == 0 {
		fmt.Fprintf(buf, "_None_\n\n")
	} else {
		writeDeniedActionsTable(buf, evaluation.DeniedActions)
	}

	fmt.Fprintf(buf, "## Metrics\n\n")
	writeMetricTable(buf, evaluation.Metrics)
}

func writeExpectedActualMarkdown(buf *bytes.Buffer, comparison ReviewPacketComparison) {
	fmt.Fprintf(buf, "## Expected vs Actual\n\n")
	writeCheckSection(buf, "Final Outcome", comparison.FinalOutcome)
	writeCheckSection(buf, "Policy Decisions", comparison.PolicyDecisions)
	writeCheckSection(buf, "Required Evidence", comparison.RequiredEvidence)
	writeCheckSection(buf, "Forbidden Evidence", comparison.ForbiddenEvidence)
	writeCheckSection(buf, "Tool Calls", comparison.ToolCalls)
	writeCheckSection(buf, "Approval Behavior", comparison.ApprovalBehavior)
	writeCheckSection(buf, "Final State", comparison.FinalState)
	writeCheckSection(buf, "Audit Events", comparison.AuditEvents)
	writeCheckSection(buf, "Contract Validation", comparison.ContractValidation)
}

func buildExpectedToolCallChecks(expected []ExpectedToolCall, actualCounts map[string]int) []ReviewPacketCheck {
	checks := make([]ReviewPacketCheck, 0, len(expected))
	for _, item := range expected {
		actualValue := packetValueAbsent
		status := ReviewPacketStatusPass
		evidence := fmt.Sprintf("tool %q observed %d time(s)", item.ToolName, actualCounts[item.ToolName])
		if actualCounts[item.ToolName] > 0 {
			actualValue = packetValuePresent
		}
		if item.Required && actualCounts[item.ToolName] == 0 {
			status = ReviewPacketStatusFail
			evidence = fmt.Sprintf("required tool call %q was not executed", item.ToolName)
		}
		checks = append(checks, ReviewPacketCheck{
			Dimension: string(DimToolCalls),
			Label:     item.ToolName,
			Expected:  requiredToolExpectation(item),
			Actual:    actualValue,
			Status:    status,
			Evidence:  evidence,
		})
	}
	return checks
}

func buildObservedToolCallChecks(
	actual []TraceToolCall,
	expectedSet map[string]struct{},
	forbiddenSet map[string]string,
) []ReviewPacketCheck {
	checks := make([]ReviewPacketCheck, 0, len(actual))
	for _, item := range actual {
		if check, ok := buildForbiddenToolCallCheck(item, forbiddenSet); ok {
			checks = append(checks, check)
			continue
		}
		if _, ok := expectedSet[item.ToolName]; ok {
			continue
		}
		checks = append(checks, ReviewPacketCheck{
			Dimension: string(DimExtraToolCalls),
			Label:     item.ToolName,
			Expected:  "not expected",
			Actual:    item.Status,
			Status:    ReviewPacketStatusInfo,
			Evidence:  fmt.Sprintf("unexpected but non-forbidden tool %q observed", item.ToolName),
		})
	}
	return checks
}

func buildForbiddenToolCallCheck(
	item TraceToolCall,
	forbiddenSet map[string]string,
) (ReviewPacketCheck, bool) {
	if _, ok := forbiddenSet[item.ToolName]; !ok {
		return ReviewPacketCheck{}, false
	}
	return ReviewPacketCheck{
		Dimension: string(DimForbiddenToolCalls),
		Label:     item.ToolName,
		Expected:  packetValueAbsent,
		Actual:    item.Status,
		Status:    ReviewPacketStatusFail,
		Evidence:  fmt.Sprintf("forbidden tool %q executed", item.ToolName),
	}, true
}

func writeHardGateTable(buf *bytes.Buffer, violations []HardGateViolation) {
	fmt.Fprintf(buf, "| Gate | Severity | Expected | Actual | Evidence |\n")
	fmt.Fprintf(buf, "| --- | --- | --- | --- | --- |\n")
	for _, item := range violations {
		fmt.Fprintf(buf, "| `%s` | `%s` | %s | %s | %s |\n",
			item.Gate,
			item.Severity,
			markdownCell(item.Expected),
			markdownCell(item.Actual),
			markdownCell(item.Evidence),
		)
	}
	fmt.Fprintf(buf, "\n")
}

func writeMetricTable(buf *bytes.Buffer, metrics Metrics) {
	rows := []struct {
		name  string
		value string
	}{
		{name: "outcome_accuracy", value: formatFloat(metrics.OutcomeAccuracy)},
		{name: "tool_call_precision", value: formatFloat(metrics.ToolCallPrecision)},
		{name: "tool_call_recall", value: formatFloat(metrics.ToolCallRecall)},
		{name: "tool_call_f1", value: formatFloat(metrics.ToolCallF1)},
		{name: "forbidden_tool_violations", value: fmt.Sprintf(packetFmtInt, metrics.ForbiddenToolViolations)},
		{name: "policy_compliance", value: formatFloat(metrics.PolicyCompliance)},
		{name: "approval_accuracy", value: formatFloat(metrics.ApprovalAccuracy)},
		{name: "evidence_coverage", value: formatFloat(metrics.EvidenceCoverage)},
		{name: "forbidden_evidence_count", value: fmt.Sprintf(packetFmtInt, metrics.ForbiddenEvidenceCount)},
		{name: "state_mutation_accuracy", value: formatFloat(metrics.StateMutationAccuracy)},
		{name: "audit_completeness", value: formatFloat(metrics.AuditCompleteness)},
		{name: "contract_validity", value: formatFloat(metrics.ContractValidity)},
		{name: "abstention_accuracy", value: formatFloat(metrics.AbstentionAccuracy)},
		{name: "latency_compliance", value: formatFloat(metrics.LatencyCompliance)},
		{name: "tool_budget_compliance", value: formatFloat(metrics.ToolBudgetCompliance)},
	}

	fmt.Fprintf(buf, "| Metric | Value |\n")
	fmt.Fprintf(buf, "| --- | --- |\n")
	for _, row := range rows {
		fmt.Fprintf(buf, "| `%s` | `%s` |\n", row.name, row.value)
	}
	fmt.Fprintf(buf, "\n")
}

func writeCheckSection(buf *bytes.Buffer, title string, checks []ReviewPacketCheck) {
	if len(checks) == 0 {
		return
	}
	fmt.Fprintf(buf, "### %s\n\n", title)
	fmt.Fprintf(buf, "| Label | Expected | Actual | Status | Evidence |\n")
	fmt.Fprintf(buf, "| --- | --- | --- | --- | --- |\n")
	for _, item := range checks {
		fmt.Fprintf(buf, "| `%s` | %s | %s | `%s` | %s |\n",
			item.Label,
			markdownCell(item.Expected),
			markdownCell(item.Actual),
			item.Status,
			markdownCell(item.Evidence),
		)
	}
	fmt.Fprintf(buf, "\n")
}

func sortReviewPacketChecks(items []ReviewPacketCheck) {
	sort.Slice(items, func(i, j int) bool {
		left := items[i]
		right := items[j]
		if left.Dimension != right.Dimension {
			return left.Dimension < right.Dimension
		}
		if left.Label != right.Label {
			return left.Label < right.Label
		}
		if left.Expected != right.Expected {
			return left.Expected < right.Expected
		}
		return left.Actual < right.Actual
	})
}

type packetAuditMetadata struct {
	Action    string `json:"action,omitempty"`
	Policy    string `json:"policy,omitempty"`
	PolicyID  string `json:"policy_id,omitempty"`
	PolicySet string `json:"policy_set,omitempty"`
	Reason    string `json:"reason,omitempty"`
	ErrorCode string `json:"error_code,omitempty"`
	Target    string `json:"target,omitempty"`
	ToolName  string `json:"tool_name,omitempty"`
}

type packetAuditDetails struct {
	Metadata packetAuditMetadata `json:"metadata"`
}

func buildDeniedActionSummaries(trace ActualRunTrace) []ReviewPacketDeniedAction {
	out := make([]ReviewPacketDeniedAction, 0, len(trace.AuditEvents))
	for _, event := range trace.AuditEvents {
		if event.Outcome != "denied" {
			continue
		}
		details := decodeAuditDetails(event.Details)
		policy := firstNonEmpty(details.Metadata.Policy, details.Metadata.PolicyID, details.Metadata.PolicySet)
		target := firstNonEmpty(details.Metadata.Target, derefString(event.EntityID), details.Metadata.ToolName)
		reason := firstNonEmpty(details.Metadata.Reason, details.Metadata.ErrorCode)
		action := firstNonEmpty(details.Metadata.Action, event.Action)
		out = append(out, ReviewPacketDeniedAction{
			ActorID:   event.ActorID,
			Action:    action,
			Target:    target,
			Policy:    policy,
			Reason:    reason,
			Outcome:   event.Outcome,
			Timestamp: event.At,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Timestamp.Equal(out[j].Timestamp) {
			return out[i].Timestamp.Before(out[j].Timestamp)
		}
		if out[i].Action != out[j].Action {
			return out[i].Action < out[j].Action
		}
		return out[i].Target < out[j].Target
	})
	return out
}

func decodeAuditDetails(raw json.RawMessage) packetAuditDetails {
	if len(raw) == 0 {
		return packetAuditDetails{}
	}
	var details packetAuditDetails
	if err := json.Unmarshal(raw, &details); err != nil {
		return packetAuditDetails{}
	}
	return details
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func writeDeniedActionsTable(buf *bytes.Buffer, items []ReviewPacketDeniedAction) {
	fmt.Fprintf(buf, "| Actor | Action | Target | Policy | Reason | Outcome | Timestamp |\n")
	fmt.Fprintf(buf, "| --- | --- | --- | --- | --- | --- | --- |\n")
	for _, item := range items {
		fmt.Fprintf(
			buf,
			"| `%s` | `%s` | `%s` | `%s` | %s | `%s` | `%s` |\n",
			item.ActorID,
			item.Action,
			item.Target,
			item.Policy,
			safeMarkdownText(item.Reason),
			item.Outcome,
			item.Timestamp.UTC().Format(time.RFC3339),
		)
	}
	fmt.Fprintf(buf, "\n")
}

func requiredToolExpectation(item ExpectedToolCall) string {
	if item.Required {
		return "present"
	}
	return "optional"
}

func boolToPresence(value bool) string {
	if value {
		return "present"
	}
	return "absent"
}

func boolToPacketStatus(value bool) ReviewPacketStatus {
	if value {
		return ReviewPacketStatusPass
	}
	return ReviewPacketStatusFail
}

func marshalValue(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(data)
}

func markdownCell(value string) string {
	replacer := strings.NewReplacer("|", "\\|", "\n", "<br>")
	return replacer.Replace(safeMarkdownText(value))
}

func safeMarkdownText(value string) string {
	if value == "" {
		return ""
	}
	return strings.TrimSpace(value)
}

func formatFloat(value float64) string {
	return fmt.Sprintf("%.2f", value)
}
