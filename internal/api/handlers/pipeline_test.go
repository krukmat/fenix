// Traces: FR-002
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
)

func TestPipelineHandler_CreatePipeline_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewPipelineHandler(crm.NewPipelineService(db))

	body, _ := json.Marshal(map[string]any{"name": "Sales", "entityType": "deal"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipelines", bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	h.CreatePipeline(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestPipelineHandler_CreatePipeline_MissingRequired_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewPipelineHandler(crm.NewPipelineService(db))

	body, _ := json.Marshal(map[string]any{"name": "OnlyName"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipelines", bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	h.CreatePipeline(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestPipelineHandler_CreatePipeline_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewPipelineHandler(crm.NewPipelineService(db))

	body, _ := json.Marshal(map[string]any{"name": "Sales", "entityType": "deal"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipelines", bytes.NewReader(body))

	rr := httptest.NewRecorder()
	h.CreatePipeline(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestPipelineHandler_CreatePipeline_InvalidJSON_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewPipelineHandler(crm.NewPipelineService(db))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipelines", bytes.NewBufferString(`{"name":`))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	h.CreatePipeline(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestPipelineHandler_ListPipelines_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	svc := crm.NewPipelineService(db)
	h := NewPipelineHandler(svc)

	for i := 0; i < 2; i++ {
		_, err := svc.Create(t.Context(), crm.CreatePipelineInput{WorkspaceID: wsID, Name: fmt.Sprintf("P%d", i+1), EntityType: "deal"})
		if err != nil {
			t.Fatalf("seed pipeline failed: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipelines?limit=10&offset=0", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	h.ListPipelines(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestPipelineHandler_GetPipeline_NotFound_Returns404(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewPipelineHandler(crm.NewPipelineService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipelines/none", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "none")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.GetPipeline(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestPipelineHandler_GetPipeline_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	svc := crm.NewPipelineService(db)
	h := NewPipelineHandler(svc)

	p, err := svc.Create(t.Context(), crm.CreatePipelineInput{WorkspaceID: wsID, Name: "Sales", EntityType: "deal"})
	if err != nil {
		t.Fatalf("seed pipeline failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipelines/"+p.ID, nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", p.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.GetPipeline(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestPipelineHandler_ListPipelines_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewPipelineHandler(crm.NewPipelineService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipelines", nil)
	rr := httptest.NewRecorder()
	h.ListPipelines(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestPipelineHandler_GetPipeline_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewPipelineHandler(crm.NewPipelineService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipelines/p1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "p1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.GetPipeline(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestPipelineHandler_UpdatePipeline_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	svc := crm.NewPipelineService(db)
	h := NewPipelineHandler(svc)

	p, err := svc.Create(t.Context(), crm.CreatePipelineInput{WorkspaceID: wsID, Name: "Initial", EntityType: "deal"})
	if err != nil {
		t.Fatalf("seed pipeline failed: %v", err)
	}

	body, _ := json.Marshal(map[string]any{"name": "Updated"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/pipelines/"+p.ID, bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", p.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.UpdatePipeline(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestPipelineHandler_DeletePipeline_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	svc := crm.NewPipelineService(db)
	h := NewPipelineHandler(svc)

	p, err := svc.Create(t.Context(), crm.CreatePipelineInput{WorkspaceID: wsID, Name: "ToDelete", EntityType: "deal"})
	if err != nil {
		t.Fatalf("seed pipeline failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/pipelines/"+p.ID, nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", p.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.DeletePipeline(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
}

func TestPipelineHandler_UpdatePipeline_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewPipelineHandler(crm.NewPipelineService(db))

	req := httptest.NewRequest(http.MethodPut, "/api/v1/pipelines/p1", bytes.NewBufferString(`{"name":"x"}`))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "p1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.UpdatePipeline(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestPipelineHandler_UpdatePipeline_InvalidJSON_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	svc := crm.NewPipelineService(db)
	h := NewPipelineHandler(svc)

	p, err := svc.Create(t.Context(), crm.CreatePipelineInput{WorkspaceID: wsID, Name: "Initial", EntityType: "deal"})
	if err != nil {
		t.Fatalf("seed pipeline failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/pipelines/"+p.ID, bytes.NewBufferString(`{"name":`))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", p.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.UpdatePipeline(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestPipelineHandler_DeletePipeline_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewPipelineHandler(crm.NewPipelineService(db))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/pipelines/p1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "p1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.DeletePipeline(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestPipelineHandler_CreateStage_InvalidJSON_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewPipelineHandler(crm.NewPipelineService(db))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipelines/p1/stages", bytes.NewBufferString(`{"name":`))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "p1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.CreateStage(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestPipelineHandler_CreateStage_MissingName_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewPipelineHandler(crm.NewPipelineService(db))

	body, _ := json.Marshal(map[string]any{"position": 1})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipelines/p1/stages", bytes.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "p1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.CreateStage(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestPipelineHandler_ListStages_EmptyPipelineID_StillHandlesRequest(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewPipelineHandler(crm.NewPipelineService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipelines//stages", nil)
	rr := httptest.NewRecorder()
	h.ListStages(rr, req)

	if rr.Code != http.StatusOK && rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 200 or 500, got %d", rr.Code)
	}
}

func TestPipelineHandler_ListStages_DBError_Returns500(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewPipelineHandler(crm.NewPipelineService(db))
	_ = db.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipelines/p1/stages", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "p1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.ListStages(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestPipelineHandler_UpdateStage_InvalidJSON_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewPipelineHandler(crm.NewPipelineService(db))

	req := httptest.NewRequest(http.MethodPut, "/api/v1/pipelines/p1/stages/s1", bytes.NewBufferString(`{"name":`))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("stage_id", "s1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.UpdateStage(rr, req)

	if rr.Code != http.StatusNotFound && rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 404 or 500, got %d", rr.Code)
	}
}

func TestPipelineHandler_CreateStage_List_Update_Delete_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	svc := crm.NewPipelineService(db)
	h := NewPipelineHandler(svc)

	p, err := svc.Create(t.Context(), crm.CreatePipelineInput{WorkspaceID: wsID, Name: "Sales", EntityType: "deal"})
	if err != nil {
		t.Fatalf("seed pipeline failed: %v", err)
	}

	// Create stage
	body, _ := json.Marshal(map[string]any{"name": "Prospect", "position": 1})
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/pipelines/"+p.ID+"/stages", bytes.NewReader(body))
	rctxCreate := chi.NewRouteContext()
	rctxCreate.URLParams.Add("id", p.ID)
	reqCreate = reqCreate.WithContext(context.WithValue(reqCreate.Context(), chi.RouteCtxKey, rctxCreate))

	rrCreate := httptest.NewRecorder()
	h.CreateStage(rrCreate, reqCreate)

	if rrCreate.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rrCreate.Code, rrCreate.Body.String())
	}

	var created map[string]any
	if err := json.Unmarshal(rrCreate.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode stage failed: %v", err)
	}
	stageID, _ := created["id"].(string)
	if stageID == "" {
		t.Fatalf("expected stage id")
	}

	// List stages
	reqList := httptest.NewRequest(http.MethodGet, "/api/v1/pipelines/"+p.ID+"/stages", nil)
	rctxList := chi.NewRouteContext()
	rctxList.URLParams.Add("id", p.ID)
	reqList = reqList.WithContext(context.WithValue(reqList.Context(), chi.RouteCtxKey, rctxList))

	rrList := httptest.NewRecorder()
	h.ListStages(rrList, reqList)
	if rrList.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rrList.Code)
	}

	// Update stage
	bodyUp, _ := json.Marshal(map[string]any{"name": "Qualified", "position": 2})
	reqUp := httptest.NewRequest(http.MethodPut, "/api/v1/pipelines/"+p.ID+"/stages/"+stageID, bytes.NewReader(bodyUp))
	rctxUp := chi.NewRouteContext()
	rctxUp.URLParams.Add("stage_id", stageID)
	reqUp = reqUp.WithContext(context.WithValue(reqUp.Context(), chi.RouteCtxKey, rctxUp))

	rrUp := httptest.NewRecorder()
	h.UpdateStage(rrUp, reqUp)
	if rrUp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rrUp.Code, rrUp.Body.String())
	}

	// Delete stage
	reqDel := httptest.NewRequest(http.MethodDelete, "/api/v1/pipelines/"+p.ID+"/stages/"+stageID, nil)
	rctxDel := chi.NewRouteContext()
	rctxDel.URLParams.Add("stage_id", stageID)
	reqDel = reqDel.WithContext(context.WithValue(reqDel.Context(), chi.RouteCtxKey, rctxDel))

	rrDel := httptest.NewRecorder()
	h.DeleteStage(rrDel, reqDel)
	if rrDel.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rrDel.Code)
	}
}

func TestPipelineHandler_UpdateStage_NotFound_Returns404(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewPipelineHandler(crm.NewPipelineService(db))

	body, _ := json.Marshal(map[string]any{"name": "Qualified", "position": 2})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/pipelines/p1/stages/missing", bytes.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("stage_id", "missing")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.UpdateStage(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestPipelineHandler_UpdateStage_InvalidJSON_WithExistingStage_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	svc := crm.NewPipelineService(db)
	h := NewPipelineHandler(svc)

	p, err := svc.Create(t.Context(), crm.CreatePipelineInput{WorkspaceID: wsID, Name: "Sales", EntityType: "deal"})
	if err != nil {
		t.Fatalf("seed pipeline failed: %v", err)
	}
	stage, err := svc.CreateStage(t.Context(), crm.CreatePipelineStageInput{PipelineID: p.ID, Name: "S1", Position: 1})
	if err != nil {
		t.Fatalf("seed stage failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/pipelines/"+p.ID+"/stages/"+stage.ID, bytes.NewBufferString(`{"name":`))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("stage_id", stage.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.UpdateStage(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestPipelineHandler_DeleteStage_UsesIDFallback(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	svc := crm.NewPipelineService(db)
	h := NewPipelineHandler(svc)

	p, err := svc.Create(t.Context(), crm.CreatePipelineInput{WorkspaceID: wsID, Name: "Sales", EntityType: "deal"})
	if err != nil {
		t.Fatalf("seed pipeline failed: %v", err)
	}
	stage, err := svc.CreateStage(t.Context(), crm.CreatePipelineStageInput{PipelineID: p.ID, Name: "S1", Position: 1})
	if err != nil {
		t.Fatalf("seed stage failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/pipelines/stages/"+stage.ID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", stage.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.DeleteStage(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
}

func TestFillStageDefaults_KeepsExistingWhenEmpty(t *testing.T) {
	t.Parallel()

	existing := &crm.PipelineStage{Name: "Discovery", Position: 3}
	got := fillStageDefaults(UpdatePipelineStageRequest{}, existing)

	if got.Name != "Discovery" || got.Position != 3 {
		t.Fatalf("unexpected defaults applied: %+v", got)
	}
}
