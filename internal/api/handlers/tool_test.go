package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
)

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
