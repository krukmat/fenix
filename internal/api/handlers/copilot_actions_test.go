package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/copilot"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
)

type copilotActionsServiceStub struct {
	actions []copilot.SuggestedAction
	summary string
	brief   *copilot.SalesBriefResult
	err     error
}

func (s *copilotActionsServiceStub) SuggestActions(_ context.Context, _ copilot.SuggestActionsInput) ([]copilot.SuggestedAction, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.actions, nil
}

func (s *copilotActionsServiceStub) Summarize(_ context.Context, _ copilot.SummarizeInput) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.summary, nil
}

func (s *copilotActionsServiceStub) SalesBrief(_ context.Context, _ copilot.SalesBriefInput) (*copilot.SalesBriefResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.brief, nil
}

func TestActionRequestError_Error(t *testing.T) {
	t.Parallel()

	e := actionRequestError{status: 401, message: "unauthorized"}
	if got := e.Error(); got != "unauthorized" {
		t.Fatalf("expected %q, got %q", "unauthorized", got)
	}
}

func TestCopilotActionsHandler_SuggestActions_OK(t *testing.T) {
	t.Parallel()

	h := NewCopilotActionsHandler(&copilotActionsServiceStub{actions: []copilot.SuggestedAction{{
		Title:           "Crear seguimiento",
		Description:     "Coordinar proximo paso",
		Tool:            "create_task",
		Params:          map[string]any{"entity_type": "case", "entity_id": "c1"},
		ConfidenceScore: 0.8,
		ConfidenceLevel: copilot.ConfidenceLevelHigh,
	}}})

	body, _ := json.Marshal(map[string]any{"entityType": "case", "entityId": "c1"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/copilot/suggest-actions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), ctxkeys.WorkspaceID, "ws_1")
	ctx = context.WithValue(ctx, ctxkeys.UserID, "u_1")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.SuggestActions(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Data struct {
			Actions []copilot.SuggestedAction `json:"actions"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response error: %v", err)
	}
	if len(resp.Data.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(resp.Data.Actions))
	}
	if resp.Data.Actions[0].ConfidenceScore <= 0 {
		t.Fatalf("expected confidence score, got %f", resp.Data.Actions[0].ConfidenceScore)
	}
	if resp.Data.Actions[0].ConfidenceLevel == "" {
		t.Fatal("expected confidence level")
	}
}

func TestCopilotActionsHandler_Summarize_OK(t *testing.T) {
	t.Parallel()

	h := NewCopilotActionsHandler(&copilotActionsServiceStub{summary: "Resumen del caso en progreso."})

	body, _ := json.Marshal(map[string]any{"entityType": "case", "entityId": "c1"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/copilot/summarize", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), ctxkeys.WorkspaceID, "ws_1")
	ctx = context.WithValue(ctx, ctxkeys.UserID, "u_1")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.Summarize(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Data struct {
			Summary string `json:"summary"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response error: %v", err)
	}
	if resp.Data.Summary == "" {
		t.Fatal("expected non-empty summary")
	}
}

func TestCopilotActionsHandler_SalesBrief_OK(t *testing.T) {
	t.Parallel()

	h := NewCopilotActionsHandler(&copilotActionsServiceStub{brief: &copilot.SalesBriefResult{
		Outcome:         "completed",
		EntityType:      "deal",
		EntityID:        "d1",
		Summary:         "Deal summary",
		Risks:           []string{"Pricing objection"},
		NextBestActions: []copilot.SuggestedAction{{Title: "Actualizar deal", Tool: "update_deal", Params: map[string]any{"deal_id": "d1"}}},
		Confidence:      copilot.ConfidenceLevelHigh,
		EvidencePack: &knowledge.EvidencePack{
			SchemaVersion:        knowledge.EvidencePackSchemaVersion,
			Query:                "entity_type:deal entity_id:d1 latest updates timeline next steps",
			SourceCount:          1,
			DedupCount:           0,
			FilteredCount:        0,
			Confidence:           knowledge.ConfidenceHigh,
			Warnings:             []string{},
			RetrievalMethodsUsed: []knowledge.EvidenceMethod{knowledge.EvidenceMethodHybrid},
			BuiltAt:              time.Now().UTC(),
		},
	}})

	body, _ := json.Marshal(map[string]any{"entityType": "deal", "entityId": "d1"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/copilot/sales-brief", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), ctxkeys.WorkspaceID, "ws_1")
	ctx = context.WithValue(ctx, ctxkeys.UserID, "u_1")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.SalesBrief(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Data struct {
			Outcome      string         `json:"outcome"`
			Confidence   string         `json:"confidence"`
			Risks        []string       `json:"risks"`
			EvidencePack map[string]any `json:"evidencePack"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response error: %v", err)
	}
	if resp.Data.Outcome != "completed" {
		t.Fatalf("expected completed outcome, got %q", resp.Data.Outcome)
	}
	if resp.Data.Confidence != string(copilot.ConfidenceLevelHigh) {
		t.Fatalf("expected high confidence, got %q", resp.Data.Confidence)
	}
	if len(resp.Data.Risks) != 1 {
		t.Fatalf("expected 1 risk, got %d", len(resp.Data.Risks))
	}
	if resp.Data.EvidencePack["schema_version"] != knowledge.EvidencePackSchemaVersion {
		t.Fatalf("unexpected evidence pack: %#v", resp.Data.EvidencePack)
	}
}

func TestCopilotActionsHandler_ValidationErrors(t *testing.T) {
	t.Parallel()

	h := NewCopilotActionsHandler(&copilotActionsServiceStub{})

	t.Run("missing workspace", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/copilot/suggest-actions", bytes.NewBufferString(`{"entityType":"case","entityId":"c1"}`))
		req = req.WithContext(context.WithValue(req.Context(), ctxkeys.UserID, "u_1"))
		rr := httptest.NewRecorder()
		h.SuggestActions(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("missing user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/copilot/suggest-actions", bytes.NewBufferString(`{"entityType":"case","entityId":"c1"}`))
		req = req.WithContext(context.WithValue(req.Context(), ctxkeys.WorkspaceID, "ws_1"))
		rr := httptest.NewRecorder()
		h.SuggestActions(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/copilot/suggest-actions", bytes.NewBufferString(`{"entityType":`))
		ctx := context.WithValue(req.Context(), ctxkeys.WorkspaceID, "ws_1")
		ctx = context.WithValue(ctx, ctxkeys.UserID, "u_1")
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()
		h.SuggestActions(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("missing entity fields", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/copilot/summarize", bytes.NewBufferString(`{}`))
		ctx := context.WithValue(req.Context(), ctxkeys.WorkspaceID, "ws_1")
		ctx = context.WithValue(ctx, ctxkeys.UserID, "u_1")
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()
		h.Summarize(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})
}

func TestCopilotActionsHandler_ServiceFailure_Returns500(t *testing.T) {
	t.Parallel()

	h := NewCopilotActionsHandler(&copilotActionsServiceStub{err: errors.New("provider down")})
	body, _ := json.Marshal(map[string]any{"entityType": "case", "entityId": "c1"})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/copilot/suggest-actions", bytes.NewReader(body))
	ctx := context.WithValue(req.Context(), ctxkeys.WorkspaceID, "ws_1")
	ctx = context.WithValue(ctx, ctxkeys.UserID, "u_1")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	h.SuggestActions(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}
