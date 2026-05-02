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

	demo := loadReviewPacketDemoFixture(t, "support_case_demo.json")

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

	assertDemoPacketFixtures(t, packet, "demo_support_run.md", "demo_support_run.json")
}

func TestPolicyDenialDemoBuildsPassingReviewPacket(t *testing.T) {
	t.Parallel()

	demo := loadReviewPacketDemoFixture(t, "policy_denial_demo.json")

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
	if packet.Run.FinalOutcome != "escalated" {
		t.Fatalf("expected demo final outcome escalated, got %q", packet.Run.FinalOutcome)
	}
	if len(packet.Evaluation.DeniedActions) == 0 {
		t.Fatal("expected denied actions to be visible in packet")
	}

	assertDemoPacketFixtures(t, packet, "demo_policy_denial_run.md", "demo_policy_denial_run.json")
}

func loadReviewPacketDemoFixture(t *testing.T, fixtureName string) reviewPacketDemoFixture {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", "demo", fixtureName))
	if err != nil {
		t.Fatalf("read demo fixture: %v", err)
	}

	var fixture reviewPacketDemoFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatalf("parse demo fixture: %v", err)
	}
	return fixture
}

func assertDemoPacketFixtures(t *testing.T, packet ReviewPacket, markdownName, jsonName string) {
	t.Helper()

	expectedMarkdown, err := os.ReadFile(filepath.Join("testdata", "packets", markdownName))
	if err != nil {
		t.Fatalf("read markdown packet fixture: %v", err)
	}
	if strings.TrimSpace(string(expectedMarkdown)) != strings.TrimSpace(packet.ToMarkdown()) {
		t.Fatalf("markdown packet fixture mismatch\nexpected:\n%s\nactual:\n%s", string(expectedMarkdown), packet.ToMarkdown())
	}

	expectedJSON, err := os.ReadFile(filepath.Join("testdata", "packets", jsonName))
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
