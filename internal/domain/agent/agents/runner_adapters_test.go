package agents

import (
	"encoding/json"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
)

func TestSupportRunnerImplementsAgentRunner(t *testing.T) {
	var runner agent.Runner = &SupportRunner{}
	if runner == nil {
		t.Fatal("expected non-nil runner adapter")
	}
}

func TestProspectingRunnerImplementsAgentRunner(t *testing.T) {
	var runner agent.Runner = &ProspectingRunner{}
	if runner == nil {
		t.Fatal("expected non-nil runner adapter")
	}
}

func TestKBRunnerImplementsAgentRunner(t *testing.T) {
	var runner agent.Runner = &KBRunner{}
	if runner == nil {
		t.Fatal("expected non-nil runner adapter")
	}
}

func TestInsightsRunnerImplementsAgentRunner(t *testing.T) {
	var runner agent.Runner = &InsightsRunner{}
	if runner == nil {
		t.Fatal("expected non-nil runner adapter")
	}
}

func TestDecodeSupportAgentInputUsesWorkspaceFallback(t *testing.T) {
	cfg, err := decodeSupportAgentInput(agent.TriggerAgentInput{
		WorkspaceID: "ws-1",
		Inputs:      mustMarshalJSON(t, map[string]any{"case_id": "case-1"}),
	})
	if err != nil {
		t.Fatalf("decodeSupportAgentInput() error = %v", err)
	}
	if cfg.WorkspaceID != "ws-1" {
		t.Fatalf("WorkspaceID = %q, want %q", cfg.WorkspaceID, "ws-1")
	}
	if cfg.CaseID != "case-1" {
		t.Fatalf("CaseID = %q, want %q", cfg.CaseID, "case-1")
	}
}

func TestDecodeProspectingAgentInputUsesTriggeredByFallback(t *testing.T) {
	userID := "user-1"
	cfg, err := decodeProspectingAgentInput(agent.TriggerAgentInput{
		WorkspaceID: "ws-1",
		TriggeredBy: &userID,
		Inputs:      mustMarshalJSON(t, map[string]any{"lead_id": "lead-1"}),
	})
	if err != nil {
		t.Fatalf("decodeProspectingAgentInput() error = %v", err)
	}
	if cfg.TriggeredByUserID == nil || *cfg.TriggeredByUserID != userID {
		t.Fatalf("TriggeredByUserID = %v, want %q", cfg.TriggeredByUserID, userID)
	}
}

func TestDecodeKBAgentInputUsesWorkspaceAndTriggeredByFallbacks(t *testing.T) {
	userID := "user-1"
	cfg, err := decodeKBAgentInput(agent.TriggerAgentInput{
		WorkspaceID: "ws-1",
		TriggeredBy: &userID,
		Inputs:      mustMarshalJSON(t, map[string]any{"case_id": "case-1"}),
	})
	if err != nil {
		t.Fatalf("decodeKBAgentInput() error = %v", err)
	}
	if cfg.WorkspaceID != "ws-1" {
		t.Fatalf("WorkspaceID = %q, want %q", cfg.WorkspaceID, "ws-1")
	}
	if cfg.TriggeredByUserID == nil || *cfg.TriggeredByUserID != userID {
		t.Fatalf("TriggeredByUserID = %v, want %q", cfg.TriggeredByUserID, userID)
	}
}

func TestDecodeInsightsAgentInputUsesFallbacks(t *testing.T) {
	userID := "user-1"
	cfg, err := decodeInsightsAgentInput(agent.TriggerAgentInput{
		WorkspaceID: "ws-1",
		TriggeredBy: &userID,
		Inputs:      mustMarshalJSON(t, map[string]any{"query": "show backlog"}),
	})
	if err != nil {
		t.Fatalf("decodeInsightsAgentInput() error = %v", err)
	}
	if cfg.WorkspaceID != "ws-1" {
		t.Fatalf("WorkspaceID = %q, want %q", cfg.WorkspaceID, "ws-1")
	}
	if cfg.Query != "show backlog" {
		t.Fatalf("Query = %q, want %q", cfg.Query, "show backlog")
	}
	if cfg.TriggeredByUserID == nil || *cfg.TriggeredByUserID != userID {
		t.Fatalf("TriggeredByUserID = %v, want %q", cfg.TriggeredByUserID, userID)
	}
}

func mustMarshalJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return data
}
