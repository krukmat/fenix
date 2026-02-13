package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
)

func TestDealHandler_CreateGetDelete(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	svc := crm.NewDealService(db)
	h := NewDealHandler(svc)

	accountID := createAccountForTask15(t, db, wsID, ownerID, "Deal Account")
	pipelineID, stageID := createPipelineAndStageForTask15(t, db, wsID)

	body, _ := json.Marshal(map[string]any{
		"accountId":  accountID,
		"pipelineId": pipelineID,
		"stageId":    stageID,
		"ownerId":    ownerID,
		"title":      "Deal 1",
	})
	req := httptest.NewRequest("POST", "/api/v1/deals", bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	w := httptest.NewRecorder()
	h.CreateDeal(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateDeal status=%d", w.Code)
	}

	var created map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	id := created["id"].(string)

	getReq := httptest.NewRequest("GET", "/api/v1/deals/"+id, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	getReq = getReq.WithContext(context.WithValue(contextWithWorkspaceID(getReq.Context(), wsID), chi.RouteCtxKey, rctx))
	getW := httptest.NewRecorder()
	h.GetDeal(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("GetDeal status=%d", getW.Code)
	}

	delReq := httptest.NewRequest("DELETE", "/api/v1/deals/"+id, nil)
	delReq = delReq.WithContext(context.WithValue(contextWithWorkspaceID(delReq.Context(), wsID), chi.RouteCtxKey, rctx))
	delW := httptest.NewRecorder()
	h.DeleteDeal(delW, delReq)
	if delW.Code != http.StatusNoContent {
		t.Fatalf("DeleteDeal status=%d", delW.Code)
	}
}

func TestDealHandler_CreateDeal_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewDealHandler(crm.NewDealService(db))

	body, _ := json.Marshal(map[string]any{"title": "x"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deals", bytes.NewReader(body))

	rr := httptest.NewRecorder()
	h.CreateDeal(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestDealHandler_CreateDeal_InvalidJSON_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewDealHandler(crm.NewDealService(db))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/deals", bytes.NewBufferString(`{"accountId":`))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	h.CreateDeal(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestDealHandler_CreateDeal_MissingRequired_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewDealHandler(crm.NewDealService(db))

	body, _ := json.Marshal(map[string]any{"title": "only title"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deals", bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	h.CreateDeal(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestDealHandler_ListDeals_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewDealHandler(crm.NewDealService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/deals", nil)
	rr := httptest.NewRecorder()
	h.ListDeals(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestDealHandler_GetDeal_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewDealHandler(crm.NewDealService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/deals/d1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "d1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.GetDeal(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestDealHandler_UpdateDeal_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewDealHandler(crm.NewDealService(db))

	req := httptest.NewRequest(http.MethodPut, "/api/v1/deals/d1", bytes.NewBufferString(`{"title":"u"}`))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "d1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.UpdateDeal(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestDealHandler_DeleteDeal_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewDealHandler(crm.NewDealService(db))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/deals/d1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "d1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.DeleteDeal(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func createAccountForTask15(t *testing.T, db *sql.DB, wsID, ownerID, name string) string {
	t.Helper()
	id := "acc-" + randID()
	_, err := db.Exec(`
		INSERT INTO account (id, workspace_id, name, owner_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))
	`, id, wsID, name, ownerID)
	if err != nil {
		t.Fatalf("create account for task1.5 error=%v", err)
	}
	return id
}

func createPipelineAndStageForTask15(t *testing.T, db *sql.DB, wsID string) (string, string) {
	t.Helper()
	pipelineID := "pl-" + randID()
	stageID := "st-" + randID()
	_, err := db.Exec(`
		INSERT INTO pipeline (id, workspace_id, name, entity_type, created_at, updated_at)
		VALUES (?, ?, 'Sales', 'deal', datetime('now'), datetime('now'))
	`, pipelineID, wsID)
	if err != nil {
		t.Fatalf("create pipeline error=%v", err)
	}
	_, err = db.Exec(`
		INSERT INTO pipeline_stage (id, pipeline_id, name, position, created_at, updated_at)
		VALUES (?, ?, 'Discovery', 1, datetime('now'), datetime('now'))
	`, stageID, pipelineID)
	if err != nil {
		t.Fatalf("create pipeline stage error=%v", err)
	}
	return pipelineID, stageID
}
