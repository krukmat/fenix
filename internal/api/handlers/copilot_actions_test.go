package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/copilot"
)

type copilotActionsServiceStub struct {
	actions []copilot.SuggestedAction
	summary string
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

func TestCopilotActionsHandler_SuggestActions_OK(t *testing.T) {
	t.Parallel()

	h := NewCopilotActionsHandler(&copilotActionsServiceStub{actions: []copilot.SuggestedAction{{
		Title:       "Crear seguimiento",
		Description: "Coordinar próximo paso",
		Tool:        "create_task",
		Params:      map[string]any{"entity_id": "c1"},
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
