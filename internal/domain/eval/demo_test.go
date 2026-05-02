package eval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type reviewPacketDemoFixture struct {
	ScenarioFixture string         `json:"scenario_fixture"`
	Trace           ActualRunTrace `json:"trace"`
}

func TestSupportCaseDemoBuildsPassingReviewPacket(t *testing.T) {
	t.Parallel()

	demo := loadReviewPacketDemoFixture(t)

	scenario, err := LoadGoldenScenario(filepath.Join("testdata", demo.ScenarioFixture))
	if err != nil {
		t.Fatalf("LoadGoldenScenario() error = %v", err)
	}

	result := Compare(*scenario, demo.Trace)
	metrics := ComputeMetrics(*scenario, demo.Trace, result)
	scorecard := DefaultScorecard(metrics)
	violations := EvaluateHardGates(*scenario, demo.Trace, result)
	assessment := ApplyHardGates(scorecard, violations)
	packet := BuildReviewPacket(*scenario, demo.Trace, result, assessment)

	if !result.Pass {
		t.Fatalf("expected comparator pass, got mismatches %#v", result.Mismatches)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no hard gate violations, got %#v", violations)
	}
	if assessment.FinalVerdict != VerdictPass {
		t.Fatalf("expected final verdict %q, got %q", VerdictPass, assessment.FinalVerdict)
	}
	if packet.Run.FinalOutcome != "awaiting_approval" {
		t.Fatalf("expected demo final outcome awaiting_approval, got %q", packet.Run.FinalOutcome)
	}

	assertDemoPacketFixtures(t, packet)
}

func loadReviewPacketDemoFixture(t *testing.T) reviewPacketDemoFixture {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", "demo", "support_case_demo.json"))
	if err != nil {
		t.Fatalf("read demo fixture: %v", err)
	}

	var fixture reviewPacketDemoFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatalf("parse demo fixture: %v", err)
	}
	return fixture
}

func assertDemoPacketFixtures(t *testing.T, packet ReviewPacket) {
	t.Helper()

	expectedMarkdown, err := os.ReadFile(filepath.Join("testdata", "packets", "demo_support_run.md"))
	if err != nil {
		t.Fatalf("read markdown packet fixture: %v", err)
	}
	if strings.TrimSpace(string(expectedMarkdown)) != strings.TrimSpace(packet.ToMarkdown()) {
		t.Fatalf("markdown packet fixture mismatch\nexpected:\n%s\nactual:\n%s", string(expectedMarkdown), packet.ToMarkdown())
	}

	expectedJSON, err := os.ReadFile(filepath.Join("testdata", "packets", "demo_support_run.json"))
	if err != nil {
		t.Fatalf("read json packet fixture: %v", err)
	}
	actualJSON, err := packet.ToJSON()
	if err != nil {
		t.Fatalf("packet.ToJSON() error = %v", err)
	}
	if strings.TrimSpace(string(expectedJSON)) != strings.TrimSpace(string(actualJSON)) {
		t.Fatalf("json packet fixture mismatch\nexpected:\n%s\nactual:\n%s", string(expectedJSON), string(actualJSON))
	}
}
