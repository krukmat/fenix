package eval

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	reportJSONIndent   = "  "
	reportFallbackDash = "—"
	reportOutcomeDeny  = "deny"
)

type GovernanceReviewCase struct {
	Trace  ActualRunTrace           `json:"trace"`
	Result RegressionScenarioResult `json:"result"`
}

type GovernanceMetricsReport struct {
	GeneratedAt time.Time                   `json:"generated_at"`
	Summary     GovernanceMetricsSummary    `json:"summary"`
	Runs        []GovernanceRunMetric       `json:"runs"`
	ByWorkflow  []GovernanceDimensionMetric `json:"by_workflow,omitempty"`
	ByScenario  []GovernanceDimensionMetric `json:"by_scenario,omitempty"`
	ByActor     []GovernanceDimensionMetric `json:"by_actor,omitempty"`
	ByOutcome   []GovernanceDimensionMetric `json:"by_outcome,omitempty"`
	ByTool      []GovernanceToolMetric      `json:"by_tool,omitempty"`
}

type GovernanceMetricsSummary struct {
	TotalRuns              int     `json:"total_runs"`
	PassedRuns             int     `json:"passed_runs"`
	FailedRuns             int     `json:"failed_runs"`
	PassRate               float64 `json:"pass_rate"`
	PolicyDenialCount      int     `json:"policy_denial_count"`
	HardGateViolationCount int     `json:"hard_gate_violation_count"`
	ApprovalRequestCount   int     `json:"approval_request_count"`
	TotalToolCalls         int     `json:"total_tool_calls"`
	AverageLatencyMs       float64 `json:"average_latency_ms"`
	AverageRetries         float64 `json:"average_retries"`
	TotalCost              float64 `json:"total_cost"`
}

type GovernanceRunMetric struct {
	RunID                string   `json:"run_id"`
	ScenarioID           string   `json:"scenario_id,omitempty"`
	WorkflowID           string   `json:"workflow_id,omitempty"`
	ActorID              string   `json:"actor_id,omitempty"`
	TriggerType          string   `json:"trigger_type"`
	FinalOutcome         string   `json:"final_outcome"`
	Passed               bool     `json:"passed"`
	FinalVerdict         Verdict  `json:"final_verdict"`
	TotalCost            *float64 `json:"total_cost,omitempty"`
	LatencyMs            *int64   `json:"latency_ms,omitempty"`
	Retries              int      `json:"retries"`
	ToolCallCount        int      `json:"tool_call_count"`
	PolicyDenialCount    int      `json:"policy_denial_count"`
	ApprovalRequestCount int      `json:"approval_request_count"`
	HardGateViolations   int      `json:"hard_gate_violations"`
}

type GovernanceDimensionMetric struct {
	Key                string  `json:"key"`
	Runs               int     `json:"runs"`
	PassedRuns         int     `json:"passed_runs"`
	FailedRuns         int     `json:"failed_runs"`
	PassRate           float64 `json:"pass_rate"`
	PolicyDenialCount  int     `json:"policy_denial_count"`
	HardGateViolations int     `json:"hard_gate_violations"`
	AverageLatencyMs   float64 `json:"average_latency_ms"`
	AverageRetries     float64 `json:"average_retries"`
	TotalCost          float64 `json:"total_cost"`
}

type GovernanceToolMetric struct {
	ToolName              string `json:"tool_name"`
	Calls                 int    `json:"calls"`
	DeniedPolicyDecisions int    `json:"denied_policy_decisions"`
}

func BuildGovernanceMetricsReport(now time.Time, cases []GovernanceReviewCase) GovernanceMetricsReport {
	report := GovernanceMetricsReport{
		GeneratedAt: now.UTC(),
		Runs:        make([]GovernanceRunMetric, 0, len(cases)),
	}

	workflowAgg := map[string]*governanceAggregator{}
	scenarioAgg := map[string]*governanceAggregator{}
	actorAgg := map[string]*governanceAggregator{}
	outcomeAgg := map[string]*governanceAggregator{}
	toolAgg := map[string]*GovernanceToolMetric{}

	var latencyTotal int64
	var latencyCount int
	var retryTotal int

	for _, item := range cases {
		run := buildGovernanceRunMetric(item)
		report.Runs = append(report.Runs, run)
		accumulateGovernanceSummary(&report.Summary, run)

		if run.LatencyMs != nil {
			latencyTotal += *run.LatencyMs
			latencyCount++
		}
		retryTotal += run.Retries

		addGovernanceDimension(workflowAgg, normalizeGovernanceKey(run.WorkflowID, "unassigned_workflow"), run)
		addGovernanceDimension(scenarioAgg, normalizeGovernanceKey(run.ScenarioID, "unspecified_scenario"), run)
		addGovernanceDimension(actorAgg, normalizeGovernanceKey(run.ActorID, "unknown_actor"), run)
		addGovernanceDimension(outcomeAgg, normalizeGovernanceKey(run.FinalOutcome, "unknown_outcome"), run)
		addGovernanceToolMetrics(toolAgg, item.Trace)
	}

	report.Summary.PassRate = governancePassRate(report.Summary.PassedRuns, report.Summary.TotalRuns)
	report.Summary.AverageLatencyMs = governanceAverageInt64(latencyTotal, latencyCount)
	report.Summary.AverageRetries = governanceAverageInt(float64(retryTotal), report.Summary.TotalRuns)
	report.ByWorkflow = finalizeGovernanceDimensions(workflowAgg)
	report.ByScenario = finalizeGovernanceDimensions(scenarioAgg)
	report.ByActor = finalizeGovernanceDimensions(actorAgg)
	report.ByOutcome = finalizeGovernanceDimensions(outcomeAgg)
	report.ByTool = finalizeGovernanceToolMetrics(toolAgg)
	sortGovernanceRunMetrics(report.Runs)
	return report
}

func (r GovernanceMetricsReport) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", reportJSONIndent)
}

func (r GovernanceMetricsReport) ToMarkdown() string {
	var buf bytes.Buffer

	buf.WriteString("# Governance Metrics Report\n\n")
	fmt.Fprintf(&buf, "- Generated at: `%s`\n", r.GeneratedAt.Format(time.RFC3339))
	fmt.Fprintf(&buf, "- Runs: `%d`\n", r.Summary.TotalRuns)
	fmt.Fprintf(&buf, "- Pass rate: `%.2f`\n", r.Summary.PassRate)
	fmt.Fprintf(&buf, "- Policy denials: `%d`\n", r.Summary.PolicyDenialCount)
	fmt.Fprintf(&buf, "- Hard gate violations: `%d`\n", r.Summary.HardGateViolationCount)
	fmt.Fprintf(&buf, "- Avg latency ms: `%.2f`\n", r.Summary.AverageLatencyMs)
	fmt.Fprintf(&buf, "- Avg retries: `%.2f`\n", r.Summary.AverageRetries)
	fmt.Fprintf(&buf, "- Total cost: `%.4f`\n\n", r.Summary.TotalCost)

	if len(r.Runs) > 0 {
		buf.WriteString("## Runs\n\n")
		buf.WriteString("| Run | Scenario | Workflow | Actor | Outcome | Verdict | Cost | Latency | Retries | Policy Denials | Hard Gates |\n")
		buf.WriteString("|---|---|---|---|---|---|---:|---:|---:|---:|---:|\n")
		for _, run := range r.Runs {
			fmt.Fprintf(&buf, "| `%s` | `%s` | `%s` | `%s` | `%s` | `%s` | `%s` | `%s` | `%d` | `%d` | `%d` |\n",
				run.RunID,
				orFallback(run.ScenarioID, reportFallbackDash),
				orFallback(run.WorkflowID, reportFallbackDash),
				orFallback(run.ActorID, reportFallbackDash),
				run.FinalOutcome,
				run.FinalVerdict,
				formatOptionalCost(run.TotalCost),
				formatOptionalLatency(run.LatencyMs),
				run.Retries,
				run.PolicyDenialCount,
				run.HardGateViolations,
			)
		}
		buf.WriteString("\n")
	}

	writeGovernanceDimensionTable(&buf, "By Workflow", r.ByWorkflow)
	writeGovernanceDimensionTable(&buf, "By Scenario", r.ByScenario)
	writeGovernanceDimensionTable(&buf, "By Actor", r.ByActor)
	writeGovernanceDimensionTable(&buf, "By Outcome", r.ByOutcome)

	if len(r.ByTool) > 0 {
		buf.WriteString("## By Tool\n\n")
		buf.WriteString("| Tool | Calls | Denied Policy Decisions |\n")
		buf.WriteString("|---|---:|---:|\n")
		for _, item := range r.ByTool {
			fmt.Fprintf(&buf, "| `%s` | `%d` | `%d` |\n", item.ToolName, item.Calls, item.DeniedPolicyDecisions)
		}
		buf.WriteString("\n")
	}

	return buf.String()
}

type governanceAggregator struct {
	Key                string
	Runs               int
	PassedRuns         int
	FailedRuns         int
	PolicyDenials      int
	HardGateViolations int
	LatencyTotal       int64
	LatencyCount       int
	RetryTotal         int
	TotalCost          float64
}

func buildGovernanceRunMetric(item GovernanceReviewCase) GovernanceRunMetric {
	return GovernanceRunMetric{
		RunID:                item.Trace.RunID,
		ScenarioID:           item.Trace.ScenarioID,
		WorkflowID:           governanceWorkflowID(item.Trace),
		ActorID:              governanceActorID(item.Trace),
		TriggerType:          item.Trace.TriggerType,
		FinalOutcome:         item.Trace.FinalOutcome,
		Passed:               item.Result.Passed,
		FinalVerdict:         item.Result.HardGateAssessment.FinalVerdict,
		TotalCost:            item.Trace.TotalCost,
		LatencyMs:            item.Trace.LatencyMs,
		Retries:              item.Trace.Retries,
		ToolCallCount:        len(item.Trace.ToolCalls),
		PolicyDenialCount:    governancePolicyDenials(item.Trace),
		ApprovalRequestCount: len(item.Trace.ApprovalEvents),
		HardGateViolations:   len(item.Result.HardGateViolations),
	}
}

func accumulateGovernanceSummary(summary *GovernanceMetricsSummary, run GovernanceRunMetric) {
	summary.TotalRuns++
	if run.Passed {
		summary.PassedRuns++
	} else {
		summary.FailedRuns++
	}
	summary.PolicyDenialCount += run.PolicyDenialCount
	summary.HardGateViolationCount += run.HardGateViolations
	summary.ApprovalRequestCount += run.ApprovalRequestCount
	summary.TotalToolCalls += run.ToolCallCount
	if run.TotalCost != nil {
		summary.TotalCost += *run.TotalCost
	}
}

func addGovernanceDimension(store map[string]*governanceAggregator, key string, run GovernanceRunMetric) {
	item := store[key]
	if item == nil {
		item = &governanceAggregator{Key: key}
		store[key] = item
	}

	item.Runs++
	if run.Passed {
		item.PassedRuns++
	} else {
		item.FailedRuns++
	}
	item.PolicyDenials += run.PolicyDenialCount
	item.HardGateViolations += run.HardGateViolations
	item.RetryTotal += run.Retries
	if run.LatencyMs != nil {
		item.LatencyTotal += *run.LatencyMs
		item.LatencyCount++
	}
	if run.TotalCost != nil {
		item.TotalCost += *run.TotalCost
	}
}

func addGovernanceToolMetrics(store map[string]*GovernanceToolMetric, trace ActualRunTrace) {
	for _, call := range trace.ToolCalls {
		item := store[call.ToolName]
		if item == nil {
			item = &GovernanceToolMetric{ToolName: call.ToolName}
			store[call.ToolName] = item
		}
		item.Calls++
	}
	for _, decision := range trace.PolicyDecisions {
		if decision.Outcome != reportOutcomeDeny {
			continue
		}
		toolName := strings.TrimPrefix(decision.Action, "tool:")
		item := store[toolName]
		if item == nil {
			item = &GovernanceToolMetric{ToolName: toolName}
			store[toolName] = item
		}
		item.DeniedPolicyDecisions++
	}
}

func finalizeGovernanceDimensions(store map[string]*governanceAggregator) []GovernanceDimensionMetric {
	if len(store) == 0 {
		return nil
	}
	out := make([]GovernanceDimensionMetric, 0, len(store))
	for _, item := range store {
		out = append(out, GovernanceDimensionMetric{
			Key:                item.Key,
			Runs:               item.Runs,
			PassedRuns:         item.PassedRuns,
			FailedRuns:         item.FailedRuns,
			PassRate:           governancePassRate(item.PassedRuns, item.Runs),
			PolicyDenialCount:  item.PolicyDenials,
			HardGateViolations: item.HardGateViolations,
			AverageLatencyMs:   governanceAverageInt64(item.LatencyTotal, item.LatencyCount),
			AverageRetries:     governanceAverageInt(float64(item.RetryTotal), item.Runs),
			TotalCost:          item.TotalCost,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Runs != out[j].Runs {
			return out[i].Runs > out[j].Runs
		}
		return out[i].Key < out[j].Key
	})
	return out
}

func finalizeGovernanceToolMetrics(store map[string]*GovernanceToolMetric) []GovernanceToolMetric {
	if len(store) == 0 {
		return nil
	}
	out := make([]GovernanceToolMetric, 0, len(store))
	for _, item := range store {
		out = append(out, *item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Calls != out[j].Calls {
			return out[i].Calls > out[j].Calls
		}
		return out[i].ToolName < out[j].ToolName
	})
	return out
}

func governanceWorkflowID(trace ActualRunTrace) string {
	return firstNonEmptyJSONValue(trace.Output, "workflow_id", firstNonEmptyJSONValue(trace.InputEvent, "workflow_id", ""))
}

func governanceActorID(trace ActualRunTrace) string {
	for _, event := range trace.AuditEvents {
		if event.ActorID != "" {
			return event.ActorID
		}
	}
	return ""
}

func governancePolicyDenials(trace ActualRunTrace) int {
	total := 0
	for _, decision := range trace.PolicyDecisions {
		if decision.Outcome == "deny" {
			total++
		}
	}
	return total
}

func firstNonEmptyJSONValue(raw json.RawMessage, key, fallback string) string {
	if len(raw) == 0 {
		return fallback
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return fallback
	}
	value, ok := data[key]
	if !ok {
		return fallback
	}
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return fallback
	}
	return text
}

func governancePassRate(passed, total int) float64 {
	return governanceAverageInt(float64(passed), total)
}

func governanceAverageInt64(total int64, count int) float64 {
	if count == 0 {
		return 0
	}
	return float64(total) / float64(count)
}

func governanceAverageInt(total float64, count int) float64 {
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func normalizeGovernanceKey(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func sortGovernanceRunMetrics(items []GovernanceRunMetric) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].ScenarioID != items[j].ScenarioID {
			return items[i].ScenarioID < items[j].ScenarioID
		}
		return items[i].RunID < items[j].RunID
	})
}

func writeGovernanceDimensionTable(buf *bytes.Buffer, title string, items []GovernanceDimensionMetric) {
	if len(items) == 0 {
		return
	}
	fmt.Fprintf(buf, "## %s\n\n", title)
	buf.WriteString("| Key | Runs | Passed | Failed | Pass Rate | Policy Denials | Hard Gates | Avg Latency | Avg Retries | Total Cost |\n")
	buf.WriteString("|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, item := range items {
		fmt.Fprintf(buf, "| `%s` | `%d` | `%d` | `%d` | `%.2f` | `%d` | `%d` | `%.2f` | `%.2f` | `%.4f` |\n",
			item.Key,
			item.Runs,
			item.PassedRuns,
			item.FailedRuns,
			item.PassRate,
			item.PolicyDenialCount,
			item.HardGateViolations,
			item.AverageLatencyMs,
			item.AverageRetries,
			item.TotalCost,
		)
	}
	buf.WriteString("\n")
}

func formatOptionalCost(value *float64) string {
	if value == nil {
		return "—"
	}
	return fmt.Sprintf("%.4f", *value)
}

func formatOptionalLatency(value *int64) string {
	if value == nil {
		return "—"
	}
	return fmt.Sprintf("%d", *value)
}

func orFallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
