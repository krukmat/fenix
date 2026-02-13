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

func TestNoteHandler_GetNote_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	handler := NewNoteHandler(crm.NewNoteService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notes/n1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "n1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.GetNote(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestNoteHandler_ListNotes_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	handler := NewNoteHandler(crm.NewNoteService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notes", nil)
	rr := httptest.NewRecorder()
	handler.ListNotes(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestNoteHandler_UpdateNote_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	handler := NewNoteHandler(crm.NewNoteService(db))

	body := bytes.NewBufferString(`{"content":"updated"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/notes/n1", body)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "n1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.UpdateNote(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestNoteHandler_DeleteNote_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	handler := NewNoteHandler(crm.NewNoteService(db))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/notes/n1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "n1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.DeleteNote(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestNoteHandler_CreateNote_InvalidJSON_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	handler := NewNoteHandler(crm.NewNoteService(db))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", bytes.NewBufferString(`{"entityType":`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	handler.CreateNote(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestNoteHandler_CreateNote_MissingRequiredFields_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	handler := NewNoteHandler(crm.NewNoteService(db))

	body, _ := json.Marshal(map[string]any{"content": "sin campos clave"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	handler.CreateNote(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestNoteHandler_GetNote_NotFound_Returns404(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	handler := NewNoteHandler(crm.NewNoteService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notes/nonexistent", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.GetNote(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestNoteHandler_CreateNote_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	handler := NewNoteHandler(crm.NewNoteService(db))

	body, _ := json.Marshal(map[string]any{"content": "test"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", bytes.NewReader(body))

	rr := httptest.NewRecorder()
	handler.CreateNote(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}
