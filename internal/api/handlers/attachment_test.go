// Traces: FR-001
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

func TestAttachmentHandler_CreateAttachment_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewAttachmentHandler(crm.NewAttachmentService(db))

	body, _ := json.Marshal(map[string]any{"filename": "f.txt"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/attachments", bytes.NewReader(body))

	rr := httptest.NewRecorder()
	h.CreateAttachment(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestAttachmentHandler_CreateAttachment_InvalidJSON_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewAttachmentHandler(crm.NewAttachmentService(db))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/attachments", bytes.NewBufferString(`{"entityType":`))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	h.CreateAttachment(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestAttachmentHandler_CreateAttachment_MissingRequired_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewAttachmentHandler(crm.NewAttachmentService(db))

	body, _ := json.Marshal(map[string]any{"entityType": "case"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/attachments", bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	h.CreateAttachment(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestAttachmentHandler_GetAttachment_NotFound_Returns404(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewAttachmentHandler(crm.NewAttachmentService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/attachments/missing", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "missing")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.GetAttachment(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestAttachmentHandler_ListAttachments_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewAttachmentHandler(crm.NewAttachmentService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/attachments?limit=10&offset=0", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	h.ListAttachments(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestAttachmentHandler_DeleteAttachment_SuccessEvenIfMissing(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewAttachmentHandler(crm.NewAttachmentService(db))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/attachments/missing", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "missing")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.DeleteAttachment(rr, req)

	// service delete is idempotent from handler perspective in this path
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAttachmentHandler_GetAttachment_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewAttachmentHandler(crm.NewAttachmentService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/attachments/a1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "a1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.GetAttachment(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestAttachmentHandler_ListAttachments_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewAttachmentHandler(crm.NewAttachmentService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/attachments", nil)
	rr := httptest.NewRecorder()
	h.ListAttachments(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestAttachmentHandler_DeleteAttachment_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewAttachmentHandler(crm.NewAttachmentService(db))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/attachments/a1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "a1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.DeleteAttachment(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}
