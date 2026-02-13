package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
)

func TestCaseHandler_CreateCase_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	h := NewCaseHandler(crm.NewCaseService(db))

	body, _ := json.Marshal(map[string]any{
		"ownerId": ownerID,
		"subject": "Customer issue",
		"status":  "open",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cases", bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	h.CreateCase(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCaseHandler_CreateCase_MissingRequired_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewCaseHandler(crm.NewCaseService(db))

	body, _ := json.Marshal(map[string]any{"subject": "missing owner"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cases", bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	h.CreateCase(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCaseHandler_GetCase_NotFound_Returns404(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewCaseHandler(crm.NewCaseService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cases/missing", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "missing")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.GetCase(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestCaseHandler_ListCases_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	svc := crm.NewCaseService(db)
	h := NewCaseHandler(svc)

	for i := 0; i < 2; i++ {
		_, err := svc.Create(t.Context(), crm.CreateCaseInput{
			WorkspaceID: wsID,
			OwnerID:     ownerID,
			Subject:     fmt.Sprintf("case-%d", i+1),
		})
		if err != nil {
			t.Fatalf("seed case failed: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cases?limit=10&offset=0", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	h.ListCases(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestCaseHandler_UpdateCase_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	svc := crm.NewCaseService(db)
	h := NewCaseHandler(svc)

	created, err := svc.Create(t.Context(), crm.CreateCaseInput{WorkspaceID: wsID, OwnerID: ownerID, Subject: "Old subject"})
	if err != nil {
		t.Fatalf("seed case failed: %v", err)
	}

	body, _ := json.Marshal(map[string]any{"subject": "New subject"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/cases/"+created.ID, bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", created.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.UpdateCase(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCaseHandler_DeleteCase_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	svc := crm.NewCaseService(db)
	h := NewCaseHandler(svc)

	created, err := svc.Create(t.Context(), crm.CreateCaseInput{WorkspaceID: wsID, OwnerID: ownerID, Subject: "To delete"})
	if err != nil {
		t.Fatalf("seed case failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/cases/"+created.ID, nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", created.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.DeleteCase(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}

	_, getErr := svc.Get(t.Context(), wsID, created.ID)
	if getErr != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows after delete, got %v", getErr)
	}
}
