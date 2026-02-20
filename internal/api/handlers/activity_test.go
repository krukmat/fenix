// Traces: FR-001
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

func TestActivityHandler_GetActivity_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewActivityHandler(crm.NewActivityService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/activities/a1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "a1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.GetActivity(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestActivityHandler_ListActivities_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewActivityHandler(crm.NewActivityService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/activities", nil)
	rr := httptest.NewRecorder()
	h.ListActivities(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestActivityHandler_UpdateActivity_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewActivityHandler(crm.NewActivityService(db))

	req := httptest.NewRequest(http.MethodPut, "/api/v1/activities/a1", bytes.NewBufferString(`{"subject":"x"}`))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "a1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.UpdateActivity(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestActivityHandler_DeleteActivity_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewActivityHandler(crm.NewActivityService(db))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/activities/a1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "a1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.DeleteActivity(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestActivityHandler_GetActivity_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	h := NewActivityHandler(crm.NewActivityService(db))
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := db.Exec(`INSERT INTO activity (id, workspace_id, activity_type, entity_type, entity_id, owner_id, subject, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"act-get-1", wsID, "task", "account", "acc-1", ownerID, "Seed activity", "pending", now, now)
	if err != nil {
		t.Fatalf("seed activity error=%v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/activities/act-get-1", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "act-get-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.GetActivity(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestActivityHandler_UpdateActivity_InvalidJSON_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	h := NewActivityHandler(crm.NewActivityService(db))
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := db.Exec(`INSERT INTO activity (id, workspace_id, activity_type, entity_type, entity_id, owner_id, subject, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"act-upd-json", wsID, "task", "account", "acc-1", ownerID, "Seed activity", "pending", now, now)
	if err != nil {
		t.Fatalf("seed activity error=%v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/activities/act-upd-json", bytes.NewBufferString(`{"subject":`))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "act-upd-json")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.UpdateActivity(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestActivityHandler_DeleteActivity_Success_Returns204(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	h := NewActivityHandler(crm.NewActivityService(db))
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := db.Exec(`INSERT INTO activity (id, workspace_id, activity_type, entity_type, entity_id, owner_id, subject, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"act-del-500", wsID, "task", "account", "acc-1", ownerID, "Seed activity", "pending", now, now)
	if err != nil {
		t.Fatalf("seed activity error=%v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/activities/act-del-500", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "act-del-500")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.DeleteActivity(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
}

func TestBuildUpdateActivityInput_UsesExistingValues(t *testing.T) {
	t.Parallel()

	existing := &crm.Activity{ActivityType: "task", EntityType: "account", EntityID: "e1", OwnerID: "u1", Subject: "old"}
	got := buildUpdateActivityInput(UpdateActivityRequest{Subject: "new"}, existing)

	if got.ActivityType != "task" || got.EntityType != "account" || got.EntityID != "e1" || got.OwnerID != "u1" {
		t.Fatalf("expected fallback fields from existing, got %+v", got)
	}
	if got.Subject != "new" {
		t.Fatalf("expected updated subject, got %q", got.Subject)
	}
}
