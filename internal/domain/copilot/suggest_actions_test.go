package copilot

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/domain/policy"
)

func TestSuggestActions_ParsesEnvelopeAssignsConfidenceAndKeepsEligible(t *testing.T) {
	t.Parallel()

	snippet := "case timeline: customer blocked"
	auditLog := &auditStub{}
	svc := NewActionService(
		&evidenceStub{pack: &knowledge.EvidencePack{
			Sources:    []knowledge.Evidence{{ID: "ev_1", Snippet: &snippet}},
			Confidence: knowledge.ConfidenceHigh,
		}},
		&llmStub{resp: `{"actions":[
			{"title":"Priorizar caso","description":"Subir prioridad","tool":"update_case","params":{"case_id":"c1","priority":"high"}},
			{"title":"Enviar respuesta","description":"Informar workaround","tool":"send_reply","params":{"case_id":"c1","body":"Estamos trabajando"}},
			{"title":"Crear seguimiento","description":"Coordinar con owner","tool":"create_task","params":{"entity_type":"case","entity_id":"c1"}}
		]}`},
		&policyStub{filter: policy.Filter{Where: "workspace_id = ?"}},
		auditLog,
	)

	actions, err := svc.SuggestActions(context.Background(), SuggestActionsInput{
		WorkspaceID: "ws_1",
		UserID:      "u_1",
		EntityType:  "case",
		EntityID:    "c1",
	})
	if err != nil {
		t.Fatalf("SuggestActions error: %v", err)
	}
	if len(actions) != 3 {
		t.Fatalf("expected 3 actions, got %d", len(actions))
	}
	for _, action := range actions {
		if action.ConfidenceScore <= 0 {
			t.Fatalf("expected positive confidence score for %q", action.Title)
		}
		if action.ConfidenceLevel == "" {
			t.Fatalf("expected confidence level for %q", action.Title)
		}
	}
	if auditLog.called != 1 {
		t.Fatalf("expected audit to be called once, got %d", auditLog.called)
	}
}

func TestSuggestActions_FiltersToolEntityMismatch(t *testing.T) {
	t.Parallel()

	snippet := "lead needs follow-up"
	svc := NewActionService(
		&evidenceStub{pack: &knowledge.EvidencePack{
			Sources:    []knowledge.Evidence{{ID: "ev_1", Snippet: &snippet}},
			Confidence: knowledge.ConfidenceMedium,
		}},
		&llmStub{resp: `{"actions":[
			{"title":"Actualizar caso","description":"No aplica","tool":"update_case","params":{"case_id":"l1"}},
			{"title":"Crear tarea","description":"Seguimiento comercial","tool":"create_task","params":{"entity_type":"lead","entity_id":"l1"}}
		]}`},
		&policyStub{filter: policy.Filter{Where: "workspace_id = ?"}},
		&auditStub{},
	)

	actions, err := svc.SuggestActions(context.Background(), SuggestActionsInput{
		WorkspaceID: "ws_1",
		UserID:      "u_1",
		EntityType:  "lead",
		EntityID:    "l1",
	})
	if err != nil {
		t.Fatalf("SuggestActions error: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 eligible action, got %d", len(actions))
	}
	if actions[0].Tool != "create_task" {
		t.Fatalf("unexpected tool %q", actions[0].Tool)
	}
}

func TestSuggestActions_FiltersMissingReplyBody(t *testing.T) {
	t.Parallel()

	snippet := "case waiting for outbound response"
	svc := NewActionService(
		&evidenceStub{pack: &knowledge.EvidencePack{
			Sources:    []knowledge.Evidence{{ID: "ev_1", Snippet: &snippet}},
			Confidence: knowledge.ConfidenceHigh,
		}},
		&llmStub{resp: `{"actions":[
			{"title":"Enviar respuesta","description":"Sin body","tool":"send_reply","params":{"case_id":"c1"}}
		]}`},
		&policyStub{filter: policy.Filter{Where: "workspace_id = ?"}},
		&auditStub{},
	)

	actions, err := svc.SuggestActions(context.Background(), SuggestActionsInput{
		WorkspaceID: "ws_1",
		UserID:      "u_1",
		EntityType:  "case",
		EntityID:    "c1",
	})
	if err != nil {
		t.Fatalf("SuggestActions error: %v", err)
	}
	if len(actions) != 0 {
		t.Fatalf("expected 0 eligible actions, got %d", len(actions))
	}
}

func TestSuggestActions_ParsesJSONFence(t *testing.T) {
	t.Parallel()

	snippet := "recent updates"
	svc := NewActionService(
		&evidenceStub{pack: &knowledge.EvidencePack{
			Sources:    []knowledge.Evidence{{ID: "ev_1", Snippet: &snippet}},
			Confidence: knowledge.ConfidenceMedium,
		}},
		&llmStub{resp: "Texto previo\n```json\n{\"actions\":[{\"title\":\"Accion\",\"description\":\"Desc\",\"tool\":\"create_task\",\"params\":{\"entity_type\":\"case\",\"entity_id\":\"c1\"}}]}\n```\ntexto final"},
		&policyStub{filter: policy.Filter{Where: "workspace_id = ?"}},
		&auditStub{},
	)

	actions, err := svc.SuggestActions(context.Background(), SuggestActionsInput{
		WorkspaceID: "ws_1",
		UserID:      "u_1",
		EntityType:  "case",
		EntityID:    "c1",
	})
	if err != nil {
		t.Fatalf("SuggestActions error: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	if actions[0].Tool != "create_task" {
		t.Fatalf("unexpected tool %q", actions[0].Tool)
	}
}

func TestSuggestActions_InvalidOutput_ReturnsError(t *testing.T) {
	t.Parallel()

	svc := NewActionService(
		&evidenceStub{pack: &knowledge.EvidencePack{}},
		&llmStub{resp: "sin json valido"},
		&policyStub{filter: policy.Filter{Where: "workspace_id = ?"}},
		&auditStub{},
	)

	_, err := svc.SuggestActions(context.Background(), SuggestActionsInput{
		WorkspaceID: "ws_1",
		UserID:      "u_1",
		EntityType:  "case",
		EntityID:    "c1",
	})
	if !errors.Is(err, errSuggestedActionsParseFail) {
		t.Fatalf("expected parse error, got %v", err)
	}
}

func TestSummarize_RedactsOutputPII(t *testing.T) {
	t.Parallel()

	snippet := "customer history"
	svc := NewActionService(
		&evidenceStub{pack: &knowledge.EvidencePack{Sources: []knowledge.Evidence{{Snippet: &snippet}}}},
		&llmStub{resp: "Contactar a john@acme.com para validar cierre."},
		&policyStub{filter: policy.Filter{Where: "workspace_id = ?"}},
		&auditStub{},
	)

	summary, err := svc.Summarize(context.Background(), SummarizeInput{
		WorkspaceID: "ws_1",
		UserID:      "u_1",
		EntityType:  "case",
		EntityID:    "c1",
	})
	if err != nil {
		t.Fatalf("Summarize error: %v", err)
	}
	if strings.Contains(summary, "john@acme.com") {
		t.Fatalf("expected email to be redacted, got %q", summary)
	}
	if !strings.Contains(summary, "[REDACTED]") {
		t.Fatalf("expected [REDACTED] marker, got %q", summary)
	}
}

func TestSummarize_ValidatesEntityInput(t *testing.T) {
	t.Parallel()

	svc := NewActionService(
		&evidenceStub{pack: &knowledge.EvidencePack{}},
		&llmStub{resp: "ok"},
		&policyStub{filter: policy.Filter{Where: "workspace_id = ?"}},
		&auditStub{},
	)

	_, err := svc.Summarize(context.Background(), SummarizeInput{WorkspaceID: "ws_1", UserID: "u_1", EntityType: "", EntityID: ""})
	if !errors.Is(err, errInvalidEntityInput) {
		t.Fatalf("expected invalid entity input error, got %v", err)
	}
}
