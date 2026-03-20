package handlers

import (
	"context"
	"encoding/json"
	"math"
	"slices"

	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
)

const (
	paritySeverityHigh   = "high"
	paritySeverityMedium = "medium"
	paritySeverityLow    = "low"
	shadowPendingAction  = "pending_approval"
)

type insightsShadowComparison struct {
	PrimaryRunID         string                     `json:"primary_run_id"`
	ShadowRunID          string                     `json:"shadow_run_id"`
	EffectiveShadowRunID string                     `json:"effective_shadow_run_id"`
	Matched              bool                       `json:"matched"`
	Differences          []insightsShadowDifference `json:"differences"`
}

type insightsShadowDifference struct {
	Check    string `json:"check"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

func buildInsightsShadowComparisonFromRuns(
	primaryRun *agent.Run,
	shadowRun *agent.Run,
	effectiveShadow *agent.Run,
) insightsShadowComparison {
	report := insightsShadowComparison{
		Matched:     true,
		Differences: make([]insightsShadowDifference, 0),
	}
	if primaryRun != nil {
		report.PrimaryRunID = primaryRun.ID
	}
	if shadowRun != nil {
		report.ShadowRunID = shadowRun.ID
	}
	if effectiveShadow != nil {
		report.EffectiveShadowRunID = effectiveShadow.ID
	}
	if primaryRun == nil || effectiveShadow == nil {
		addParityDifference(&report, "comparison_input", paritySeverityHigh, "primary or effective shadow run is missing")
		return report
	}

	compareRunStatus(primaryRun, effectiveShadow, &report)
	compareOutputFields(primaryRun, effectiveShadow, &report)
	compareToolCalls(primaryRun, effectiveShadow, &report)
	compareCosts(primaryRun, effectiveShadow, &report)
	return report
}

func buildInsightsShadowComparison(
	ctx context.Context,
	orch *agent.Orchestrator,
	workspaceID string,
	primaryRun *agent.Run,
	shadowRun *agent.Run,
) insightsShadowComparison {
	return buildInsightsShadowComparisonFromRuns(primaryRun, shadowRun, resolveEffectiveShadowRun(ctx, orch, workspaceID, shadowRun))
}

func resolveEffectiveShadowRun(ctx context.Context, orch *agent.Orchestrator, workspaceID string, shadowRun *agent.Run) *agent.Run {
	if shadowRun == nil {
		return nil
	}
	childID := extractChildRunID(shadowRun.Output)
	if childID == "" || orch == nil {
		return shadowRun
	}
	childRun, err := orch.GetAgentRun(ctx, workspaceID, childID)
	if err != nil || childRun == nil {
		return shadowRun
	}
	return childRun
}

func extractChildRunID(raw json.RawMessage) string {
	decoded, ok := decodeChildRunPayload(raw)
	if !ok {
		return ""
	}
	if runID, found := extractChildRunIDFromMap(decoded); found {
		return runID
	}
	return extractChildRunIDFromStatements(decoded["statements"])
}

func decodeChildRunPayload(raw json.RawMessage) (map[string]any, bool) {
	if len(raw) == 0 || !json.Valid(raw) {
		return nil, false
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, false
	}
	return decoded, true
}

func extractChildRunIDFromStatements(value any) string {
	statements, _ := value.([]any)
	for _, statement := range statements {
		statementMap, ok := statement.(map[string]any)
		if !ok {
			continue
		}
		outputMap, _ := statementMap["output"].(map[string]any)
		if runID, found := extractChildRunIDFromMap(outputMap); found {
			return runID
		}
	}
	return ""
}

func extractChildRunIDFromMap(decoded map[string]any) (string, bool) {
	if len(decoded) == 0 {
		return "", false
	}
	runID, _ := decoded["run_id"].(string)
	_, hasAgent := decoded["agent_id"]
	if !hasAgent || runID == "" {
		return "", false
	}
	return runID, true
}

func compareRunStatus(primary, shadow *agent.Run, report *insightsShadowComparison) {
	if primary.Status != shadow.Status {
		addParityDifference(report, "status", paritySeverityHigh, "run status differs between Go and shadow execution")
	}
}

func compareOutputFields(primary, shadow *agent.Run, report *insightsShadowComparison) {
	primaryOutput := decodeJSONMap(primary.Output)
	shadowOutput := decodeJSONMap(shadow.Output)
	compareOutputString("action", primaryOutput, shadowOutput, paritySeverityHigh, report)
	compareOutputString("confidence", primaryOutput, shadowOutput, paritySeverityMedium, report)
	compareStringSliceField("evidence_ids", primaryOutput, shadowOutput, paritySeverityMedium, report)
	compareApprovalMarkers(primaryOutput, shadowOutput, report)
}

func compareOutputString(check string, primary, shadow map[string]any, severity string, report *insightsShadowComparison) {
	left, _ := primary[check].(string)
	right, _ := shadow[check].(string)
	if left != right {
		addParityDifference(report, check, severity, "output field differs between Go and shadow execution")
	}
}

func compareStringSliceField(check string, primary, shadow map[string]any, severity string, report *insightsShadowComparison) {
	left := normalizeStringSlice(primary[check])
	right := normalizeStringSlice(shadow[check])
	if !slices.Equal(left, right) {
		addParityDifference(report, check, severity, "output array differs between Go and shadow execution")
	}
}

func compareApprovalMarkers(primary, shadow map[string]any, report *insightsShadowComparison) {
	leftPending := hasPendingApproval(primary)
	rightPending := hasPendingApproval(shadow)
	if leftPending != rightPending {
		addParityDifference(report, "approval", paritySeverityHigh, "approval behavior differs between Go and shadow execution")
	}
}

func compareToolCalls(primary, shadow *agent.Run, report *insightsShadowComparison) {
	left := extractToolNames(primary.ToolCalls)
	right := extractToolNames(shadow.ToolCalls)
	if !slices.Equal(left, right) {
		addParityDifference(report, "tool_calls", paritySeverityHigh, "tool call sequence differs between Go and shadow execution")
	}
}

func compareCosts(primary, shadow *agent.Run, report *insightsShadowComparison) {
	if primary.TotalCost == nil || shadow.TotalCost == nil {
		return
	}
	if math.Abs(*primary.TotalCost-*shadow.TotalCost) > 0.000001 {
		addParityDifference(report, "cost", paritySeverityLow, "total cost differs between Go and shadow execution")
	}
}

func addParityDifference(report *insightsShadowComparison, check, severity, message string) {
	report.Matched = false
	report.Differences = append(report.Differences, insightsShadowDifference{
		Check:    check,
		Severity: severity,
		Message:  message,
	})
}

func decodeJSONMap(raw json.RawMessage) map[string]any {
	if len(raw) == 0 || !json.Valid(raw) {
		return map[string]any{}
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return map[string]any{}
	}
	return decoded
}

func extractToolNames(raw json.RawMessage) []string {
	if len(raw) == 0 || !json.Valid(raw) {
		return nil
	}
	var decoded []map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil
	}
	names := make([]string, 0, len(decoded))
	for _, item := range decoded {
		name, _ := item["tool_name"].(string)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

func normalizeStringSlice(value any) []string {
	rawItems, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(rawItems))
	for _, item := range rawItems {
		if current, isString := item.(string); isString && current != "" {
			out = append(out, current)
		}
	}
	slices.Sort(out)
	return out
}

func hasPendingApproval(decoded map[string]any) bool {
	action, _ := decoded["action"].(string)
	if action == shadowPendingAction {
		return true
	}
	_, hasApprovalID := decoded["approval_id"]
	return hasApprovalID
}
