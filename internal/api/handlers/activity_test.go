package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
)

func TestActivityHandler_CreateActivity_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewActivityHandler(crm.NewActivityService(db))

	body, _ := json.Marshal(map[string]any{"subject": "call"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/activities", bytes.NewReader(body))

	rr := httptest.NewRecorder()
	h.CreateActivity(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestActivityHandler_CreateActivity_InvalidJSON_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewActivityHandler(crm.NewActivityService(db))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/activities", bytes.NewBufferString(`{"activityType":`))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	h.CreateActivity(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestActivityHandler_CreateActivity_MissingRequired_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewActivityHandler(crm.NewActivityService(db))

	body, _ := json.Marshal(map[string]any{"activityType": "task"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/activities", bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	h.CreateActivity(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestActivityHandler_GetActivity_NotFound_Returns404(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewActivityHandler(crm.NewActivityService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/activities/missing", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "missing")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.GetActivity(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestActivityHandler_ListActivities_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewActivityHandler(crm.NewActivityService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/activities?limit=10&offset=0", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	h.ListActivities(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestActivityHandler_UpdateActivity_NotFound_Returns404(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewActivityHandler(crm.NewActivityService(db))

	body, _ := json.Marshal(map[string]any{"subject": "updated"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/activities/missing", bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "missing")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.UpdateActivity(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}
