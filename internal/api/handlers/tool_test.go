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
