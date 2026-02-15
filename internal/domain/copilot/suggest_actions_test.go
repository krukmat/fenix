package copilot

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/domain/policy"
)

func TestSuggestActions_ParsesEnvelopeAndKeepsThree(t *testing.T) {
	t.Parallel()

	snippet := "case timeline: customer blocked"
	auditLog := &auditStub{}
	svc := NewActionService(
		&evidenceStub{pack: &knowledge.EvidencePack{Sources: []knowledge.Evidence{{Snippet: &snippet}}, Confidence: knowledge.ConfidenceHigh}},
		&llmStub{resp: `{"actions":[
			{"title":"Priorizar caso","description":"Subir prioridad","tool":"update_case","params":{"case_id":"c1","priority":"high"}},
			{"title":"Enviar respuesta","description":"Informar workaround","tool":"send_reply","params":{"case_id":"c1","body":"Estamos trabajando"}},
			{"title":"Crear seguimiento","description":"Coordinar con owner","tool":"create_task","params":{"entity_type":"case","entity_id":"c1"}},
			{"title":"No válida","description":"Tool no permitido","tool":"unknown_tool","params":{}}
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
	if auditLog.called != 1 {
		t.Fatalf("expected audit to be called once, got %d", auditLog.called)
	}
}

func TestSuggestActions_ParsesJSONFence(t *testing.T) {
	t.Parallel()

	snippet := "recent updates"
	svc := NewActionService(
		&evidenceStub{pack: &knowledge.EvidencePack{Sources: []knowledge.Evidence{{Snippet: &snippet}}}},
		&llmStub{resp: "Texto previo\n```json\n{\"actions\":[{\"title\":\"Acción\",\"description\":\"Desc\",\"tool\":\"create_task\",\"params\":{}}]}\n```\ntexto final"},
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
		&llmStub{resp: "sin json válido"},
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
