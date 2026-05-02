package eval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReviewPacketMarkdownExportContainsRequiredSections(t *testing.T) {
	t.Parallel()

	packet := makeReviewPacketFixture(t)
	markdown := packet.ToMarkdown()

	requiredSections := []string{
		"# Review Packet",
		"## Scenario",
		"## Run",
		"## Evaluation",
		"## Hard Gates",
		"## Metrics",
		"## Expected vs Actual",
		"### Final Outcome",
		"### Tool Calls",
		"### Contract Validation",
	}

	for _, section := range requiredSections {
		if !strings.Contains(markdown, section) {
			t.Fatalf("expected markdown to contain %q\n%s", section, markdown)
		}
	}
}

func TestReviewPacketJSONExportValidSchema(t *testing.T) {
	t.Parallel()

	packet := makeReviewPacketFixture(t)
	data, err := packet.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	var decoded ReviewPacket
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Scenario.ID != "sc-support-001" {
		t.Fatalf("expected scenario id sc-support-001, got %q", decoded.Scenario.ID)
	}
	if decoded.Run.RunID != "run-sample-001" {
		t.Fatalf("expected run id run-sample-001, got %q", decoded.Run.RunID)
	}
	if decoded.Evaluation.FinalVerdict != VerdictFailedValidation {
		t.Fatalf("expected final verdict %q, got %q", VerdictFailedValidation, decoded.Evaluation.FinalVerdict)
	}
	if len(decoded.Comparison.RequiredEvidence) == 0 {
		t.Fatal("expected required evidence comparison rows")
	}
}

func TestReviewPacketHardGateViolationVisibleInPacket(t *testing.T) {
	t.Parallel()

	packet := makeReviewPacketFixture(t)

	if len(packet.Evaluation.HardGateViolations) == 0 {
		t.Fatal("expected hard gate violations in packet")
	}

	markdown := packet.ToMarkdown()
	if !strings.Contains(markdown, "forbidden_tool_call") {
		t.Fatalf("expected markdown to contain forbidden_tool_call\n%s", markdown)
	}
	if !strings.Contains(markdown, "failed_validation") {
		t.Fatalf("expected markdown to contain failed_validation\n%s", markdown)
	}
}

func TestReviewPacketSampleFixtures(t *testing.T) {
	t.Parallel()

	packet := makeReviewPacketFixture(t)

	expectedMarkdown, err := os.ReadFile(filepath.Join("testdata", "packets", "sample_support_run.md"))
	if err != nil {
		t.Fatalf("read markdown fixture: %v", err)
	}
	if strings.TrimSpace(string(expectedMarkdown)) != strings.TrimSpace(packet.ToMarkdown()) {
		t.Fatalf("markdown fixture mismatch\nexpected:\n%s\nactual:\n%s", string(expectedMarkdown), packet.ToMarkdown())
	}

	expectedJSON, err := os.ReadFile(filepath.Join("testdata", "packets", "sample_support_run.json"))
	if err != nil {
		t.Fatalf("read json fixture: %v", err)
	}

	actualJSON, err := packet.ToJSON()
	if err != nil {
		t.Fatalf("packet.ToJSON() error = %v", err)
	}
	if strings.TrimSpace(string(expectedJSON)) != strings.TrimSpace(string(actualJSON)) {
		t.Fatalf("json fixture mismatch\nexpected:\n%s\nactual:\n%s", string(expectedJSON), string(actualJSON))
	}
}

func makeReviewPacketFixture(t *testing.T) ReviewPacket {
	t.Helper()

	scenario, err := LoadGoldenScenario(filepath.Join("testdata", "scenarios", "sc_support_happy_path.yaml"))
	if err != nil {
		t.Fatalf("LoadGoldenScenario() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join("testdata", "traces", "sample_support_run.json"))
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}

	var trace ActualRunTrace
	if err := json.Unmarshal(data, &trace); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	finalState, err := json.Marshal(map[string]any{
		"case.status":      "Closed",
		"case.last_action": "Customer emailed",
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	trace.FinalStateRaw = finalState
	trace.ScenarioID = scenario.ID
	if trace.StartedAt.IsZero() {
		trace.StartedAt = time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC)
	}

	result := Compare(*scenario, trace)
	metrics := ComputeMetrics(*scenario, trace, result)
	scorecard := DefaultScorecard(metrics)
	violations := EvaluateHardGates(*scenario, trace, result)
	assessment := ApplyHardGates(scorecard, violations)

	return BuildReviewPacket(*scenario, trace, result, assessment)
}
