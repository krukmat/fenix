// Traces: FR-202
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
)

type toolAuthzStub struct {
	allow bool
	err   error
}

func (s *toolAuthzStub) CheckActionPermission(
	_ context.Context,
	_, _, _ string,
	_ map[string]string,
) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	return s.allow, nil
}

func TestToolHandler_CreateAndListTools(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	userID := createUser(t, db, wsID)

	h := NewToolHandler(tool.NewToolRegistry(db))

	body := map[string]any{
		"name":        "update_case",
		"inputSchema": map[string]any{"type": "object", "required": []string{"case_id"}, "properties": map[string]any{"case_id": map[string]any{"type": "string"}}, "additionalProperties": false},
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/tools", bytes.NewReader(raw))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	req = req.WithContext(context.WithValue(req.Context(), ctxkeys.UserID, userID))

	rr := httptest.NewRecorder()
	h.CreateTool(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusCreated, rr.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/tools", nil)
	listReq = listReq.WithContext(contextWithWorkspaceID(listReq.Context(), wsID))

	listRR := httptest.NewRecorder()
	h.ListTools(listRR, listReq)

	if listRR.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", listRR.Code, http.StatusOK, listRR.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(listRR.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data, ok := resp["data"].([]any)
	if !ok || len(data) != 1 {
		t.Fatalf("expected 1 tool in list, got %#v", resp["data"])
	}
}

func TestToolHandler_CreateTool_ForbiddenByAuthorizer(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	userID := createUser(t, db, wsID)

	h := NewToolHandlerWithAuthorizer(tool.NewToolRegistry(db), &toolAuthzStub{allow: false})

	body := map[string]any{
		"name":        "update_case",
		"inputSchema": map[string]any{"type": "object", "required": []string{"case_id"}, "properties": map[string]any{"case_id": map[string]any{"type": "string"}}, "additionalProperties": false},
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/tools", bytes.NewReader(raw))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	req = req.WithContext(context.WithValue(req.Context(), ctxkeys.UserID, userID))

	rr := httptest.NewRecorder()
	h.CreateTool(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusForbidden, rr.Body.String())
	}
}

func TestToolHandler_ListTools_AuthorizerError(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	userID := createUser(t, db, wsID)

	h := NewToolHandlerWithAuthorizer(tool.NewToolRegistry(db), &toolAuthzStub{err: errors.New("boom")})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/tools", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	req = req.WithContext(context.WithValue(req.Context(), ctxkeys.UserID, userID))

	rr := httptest.NewRecorder()
	h.ListTools(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusInternalServerError, rr.Body.String())
	}
}

func TestToolHandler_ToolLifecycle(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	userID := createUser(t, db, wsID)

	h := NewToolHandler(tool.NewToolRegistry(db))

	createBody := map[string]any{
		"name":                "update_case",
		"requiredPermissions": []string{"tools:update_case"},
		"inputSchema":         map[string]any{"type": "object", "required": []string{"case_id"}, "properties": map[string]any{"case_id": map[string]any{"type": "string"}}, "additionalProperties": false},
	}
	createReq := toolRequestWithBody(t, http.MethodPost, "/api/v1/admin/tools", wsID, userID, createBody)
	createRR := httptest.NewRecorder()
	h.CreateTool(createRR, createReq)

	if createRR.Code != http.StatusCreated {
		t.Fatalf("status=%d want=%d body=%s", createRR.Code, http.StatusCreated, createRR.Body.String())
	}

	var created toolResponse
	if err := json.NewDecoder(createRR.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	updateBody := map[string]any{
		"name":                "update_case_v2",
		"requiredPermissions": []string{"tools:update_case_v2"},
		"inputSchema":         map[string]any{"type": "object", "required": []string{"case_id", "status"}, "properties": map[string]any{"case_id": map[string]any{"type": "string"}, "status": map[string]any{"type": "string"}}, "additionalProperties": false},
	}
	updateReq := toolRequestWithBody(t, http.MethodPut, "/api/v1/admin/tools/"+created.ID, wsID, userID, updateBody)
	updateReq = withRouteParam(updateReq, "id", created.ID)
	updateRR := httptest.NewRecorder()
	h.UpdateTool(updateRR, updateReq)

	if updateRR.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", updateRR.Code, http.StatusOK, updateRR.Body.String())
	}

	deactivateReq := toolRequestWithBody(t, http.MethodPut, "/api/v1/admin/tools/"+created.ID+"/deactivate", wsID, userID, nil)
	deactivateReq = withRouteParam(deactivateReq, "id", created.ID)
	deactivateRR := httptest.NewRecorder()
	h.DeactivateTool(deactivateRR, deactivateReq)
	if deactivateRR.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", deactivateRR.Code, http.StatusOK, deactivateRR.Body.String())
	}

	activateReq := toolRequestWithBody(t, http.MethodPut, "/api/v1/admin/tools/"+created.ID+"/activate", wsID, userID, nil)
	activateReq = withRouteParam(activateReq, "id", created.ID)
	activateRR := httptest.NewRecorder()
	h.ActivateTool(activateRR, activateReq)
	if activateRR.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", activateRR.Code, http.StatusOK, activateRR.Body.String())
	}

	deleteReq := toolRequestWithBody(t, http.MethodDelete, "/api/v1/admin/tools/"+created.ID, wsID, userID, nil)
	deleteReq = withRouteParam(deleteReq, "id", created.ID)
	deleteRR := httptest.NewRecorder()
	h.DeleteTool(deleteRR, deleteReq)
	if deleteRR.Code != http.StatusNoContent {
		t.Fatalf("status=%d want=%d body=%s", deleteRR.Code, http.StatusNoContent, deleteRR.Body.String())
	}
}

func TestToolHandler_UpdateTool_InvalidSchema(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	userID := createUser(t, db, wsID)
	h := NewToolHandler(tool.NewToolRegistry(db))

	createBody := map[string]any{
		"name":        "update_case",
		"inputSchema": map[string]any{"type": "object", "required": []string{"case_id"}, "properties": map[string]any{"case_id": map[string]any{"type": "string"}}, "additionalProperties": false},
	}
	createReq := toolRequestWithBody(t, http.MethodPost, "/api/v1/admin/tools", wsID, userID, createBody)
	createRR := httptest.NewRecorder()
	h.CreateTool(createRR, createReq)

	var created toolResponse
	if err := json.NewDecoder(createRR.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	updateBody := map[string]any{
		"name":        "weak_tool",
		"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}, "additionalProperties": false},
	}
	updateReq := toolRequestWithBody(t, http.MethodPut, "/api/v1/admin/tools/"+created.ID, wsID, userID, updateBody)
	updateReq = withRouteParam(updateReq, "id", created.ID)
	updateRR := httptest.NewRecorder()
	h.UpdateTool(updateRR, updateReq)

	if updateRR.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=%d body=%s", updateRR.Code, http.StatusBadRequest, updateRR.Body.String())
	}
}

func toolRequestWithBody(t *testing.T, method, target, wsID, userID string, body any) *http.Request {
	t.Helper()

	var raw []byte
	if body != nil {
		var err error
		raw, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
	}

	req := httptest.NewRequest(method, target, bytes.NewReader(raw))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	if userID != "" {
		req = req.WithContext(context.WithValue(req.Context(), ctxkeys.UserID, userID))
	}
	return req
}

func withRouteParam(req *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}
