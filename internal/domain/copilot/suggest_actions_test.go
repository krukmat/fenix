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

func TestSuggestActions_AllowsAccountCreateTask(t *testing.T) {
	t.Parallel()

	snippet := "account renewal is blocked by procurement review"
	svc := NewActionService(
		&evidenceStub{pack: &knowledge.EvidencePack{
			Sources:    []knowledge.Evidence{{ID: "ev_1", Snippet: &snippet}},
			Confidence: knowledge.ConfidenceMedium,
		}},
		&llmStub{resp: `{"actions":[
			{"title":"Crear seguimiento","description":"Coordinar proximo paso comercial","tool":"create_task","params":{"entity_type":"account","entity_id":"acc_1"}}
		]}`},
		&policyStub{filter: policy.Filter{Where: "workspace_id = ?"}},
		&auditStub{},
	)

	actions, err := svc.SuggestActions(context.Background(), SuggestActionsInput{
		WorkspaceID: "ws_1",
		UserID:      "u_1",
		EntityType:  "account",
		EntityID:    "acc_1",
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

func TestSuggestActions_AllowsDealUpdateAndTask(t *testing.T) {
	t.Parallel()

	snippet := "deal stalled after pricing objection from procurement"
	svc := NewActionService(
		&evidenceStub{pack: &knowledge.EvidencePack{
			Sources:    []knowledge.Evidence{{ID: "ev_1", Snippet: &snippet}},
			Confidence: knowledge.ConfidenceHigh,
		}},
		&llmStub{resp: `{"actions":[
			{"title":"Mover deal","description":"Actualizar etapa","tool":"update_deal","params":{"deal_id":"d1","stage_id":"stage_2"}},
			{"title":"Crear seguimiento","description":"Programar llamada","tool":"create_task","params":{"entity_type":"deal","entity_id":"d1"}}
		]}`},
		&policyStub{filter: policy.Filter{Where: "workspace_id = ?"}},
		&auditStub{},
	)

	actions, err := svc.SuggestActions(context.Background(), SuggestActionsInput{
		WorkspaceID: "ws_1",
		UserID:      "u_1",
		EntityType:  "deal",
		EntityID:    "d1",
	})
	if err != nil {
		t.Fatalf("SuggestActions error: %v", err)
	}
	if len(actions) != 2 {
		t.Fatalf("expected 2 eligible actions, got %d", len(actions))
	}
	if actions[0].Tool != "update_deal" {
		t.Fatalf("unexpected first tool %q", actions[0].Tool)
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
			{"title":"Enviar respuesta","description":"Sin body","tool":"send_reply","params":{}}
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

func TestSalesBrief_CompletesForDealContext(t *testing.T) {
	t.Parallel()

	snippet := "deal has procurement objection and next meeting is pending"
	recorder := &usageRecorderStub{}
	svc := NewActionServiceWithUsage(
		&evidenceStub{pack: &knowledge.EvidencePack{
			Sources:    []knowledge.Evidence{{ID: "ev_1", Snippet: &snippet}},
			Confidence: knowledge.ConfidenceHigh,
		}},
		&llmStub{responses: []string{
			"```json\n{\"summary\":\"Deal summary\",\"risks\":[\"Procurement objection\"]}\n```",
			"```json\n{\"actions\":[{\"title\":\"Actualizar deal\",\"description\":\"Mover a negociacion\",\"tool\":\"update_deal\",\"params\":{}},{\"title\":\"Crear seguimiento\",\"description\":\"Programar llamada\",\"tool\":\"create_task\",\"params\":{}}]}\n```",
		}},
		&policyStub{filter: policy.Filter{Where: "workspace_id = ?"}},
		&auditStub{},
		recorder,
	)

	result, err := svc.SalesBrief(context.Background(), SalesBriefInput{
		WorkspaceID: "ws_1",
		UserID:      "u_1",
		EntityType:  "deal",
		EntityID:    "d1",
	})
	if err != nil {
		t.Fatalf("SalesBrief error: %v", err)
	}
	if result.Outcome != "completed" {
		t.Fatalf("expected completed, got %q", result.Outcome)
	}
	if result.Summary == "" || len(result.Risks) != 1 || len(result.NextBestActions) != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if got := result.NextBestActions[0].Params["deal_id"]; got != "d1" {
		t.Fatalf("expected normalized deal_id, got %#v", got)
	}
	if got := result.NextBestActions[1].Params["entity_type"]; got != "deal" {
		t.Fatalf("expected normalized entity_type, got %#v", got)
	}
	if got := result.NextBestActions[1].Params["entity_id"]; got != "d1" {
		t.Fatalf("expected normalized entity_id, got %#v", got)
	}
	if len(recorder.inputs) != 1 {
		t.Fatalf("expected 1 usage event, got %d", len(recorder.inputs))
	}
}

func TestSalesBrief_AbstainsWhenEvidenceIsInsufficient(t *testing.T) {
	t.Parallel()

	snippet := "generic crm note"
	llmSvc := &llmStub{resp: "should not be used"}
	svc := NewActionService(
		&evidenceStub{pack: &knowledge.EvidencePack{
			Sources:    []knowledge.Evidence{{Snippet: &snippet}},
			Confidence: knowledge.ConfidenceLow,
		}},
		llmSvc,
		&policyStub{filter: policy.Filter{Where: "workspace_id = ?"}},
		&auditStub{},
	)

	result, err := svc.SalesBrief(context.Background(), SalesBriefInput{
		WorkspaceID: "ws_1",
		UserID:      "u_1",
		EntityType:  "account",
		EntityID:    "acc_1",
	})
	if err != nil {
		t.Fatalf("SalesBrief error: %v", err)
	}
	if result.Outcome != "abstained" {
		t.Fatalf("expected abstained, got %q", result.Outcome)
	}
	if llmSvc.call != 0 {
		t.Fatalf("expected llm not to be called, got %d", llmSvc.call)
	}
}

func TestSalesBrief_FallsBackWhenBriefResponseIsEmpty(t *testing.T) {
	t.Parallel()

	snippet := "Latest updates timeline: budget approved. Risks: - Procurement review is blocked. - Pricing language needs revision. Next steps: - Send addendum today."
	svc := NewActionService(
		&evidenceStub{pack: &knowledge.EvidencePack{
			Sources:    []knowledge.Evidence{{ID: "ev_1", Snippet: &snippet}},
			Confidence: knowledge.ConfidenceHigh,
		}},
		&llmStub{responses: []string{
			"",
			`{"actions":[{"title":"Actualizar deal","description":"Registrar avance","tool":"update_deal","params":{}}]}`,
		}},
		&policyStub{filter: policy.Filter{Where: "workspace_id = ?"}},
		&auditStub{},
	)

	result, err := svc.SalesBrief(context.Background(), SalesBriefInput{
		WorkspaceID: "ws_1",
		UserID:      "u_1",
		EntityType:  "deal",
		EntityID:    "d1",
	})
	if err != nil {
		t.Fatalf("SalesBrief error: %v", err)
	}
	if result.Outcome != "completed" {
		t.Fatalf("expected completed, got %q", result.Outcome)
	}
	if result.Summary == "" {
		t.Fatal("expected fallback summary to be populated")
	}
	if len(result.Risks) != 2 {
		t.Fatalf("expected fallback risks from evidence, got %+v", result.Risks)
	}
	if len(result.NextBestActions) != 1 {
		t.Fatalf("expected normalized next best action, got %+v", result.NextBestActions)
	}
	if got := result.NextBestActions[0].Params["deal_id"]; got != "d1" {
		t.Fatalf("expected normalized deal_id on fallback action, got %#v", got)
	}
}
