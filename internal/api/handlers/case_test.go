// Traces: FR-001
package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestCaseHandler_GetCase_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewCaseHandler(crm.NewCaseService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cases/c1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "c1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.GetCase(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCaseHandler_ListCases_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewCaseHandler(crm.NewCaseService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cases", nil)
	rr := httptest.NewRecorder()
	h.ListCases(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCaseHandler_UpdateCase_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewCaseHandler(crm.NewCaseService(db))

	req := httptest.NewRequest(http.MethodPut, "/api/v1/cases/c1", bytes.NewBufferString(`{"subject":"x"}`))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "c1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.UpdateCase(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCaseHandler_DeleteCase_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewCaseHandler(crm.NewCaseService(db))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/cases/c1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "c1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.DeleteCase(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCaseHandler_ListCases_FilterByPriority(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	svc := crm.NewCaseService(db)
	h := NewCaseHandler(svc)

	_, err := svc.Create(t.Context(), crm.CreateCaseInput{
		WorkspaceID: wsID,
		OwnerID:     ownerID,
		Subject:     "critical case",
		Priority:    "urgent",
	})
	if err != nil {
		t.Fatalf("seed critical case failed: %v", err)
	}
	_, err = svc.Create(t.Context(), crm.CreateCaseInput{
		WorkspaceID: wsID,
		OwnerID:     ownerID,
		Subject:     "low case",
		Priority:    "low",
	})
	if err != nil {
		t.Fatalf("seed low case failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cases?priority="+url.QueryEscape("urgent"), nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()

	h.ListCases(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Data []struct {
			Priority string `json:"priority"`
		} `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Meta.Total != 1 || len(resp.Data) != 1 {
		t.Fatalf("expected one filtered case, total=%d len=%d", resp.Meta.Total, len(resp.Data))
	}
	if resp.Data[0].Priority != "urgent" {
		t.Fatalf("expected urgent priority, got %s", resp.Data[0].Priority)
	}
}

func TestCaseHandler_ListCases_MultipleFilters_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewCaseHandler(crm.NewCaseService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cases?status=open&owner_id=u1", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()

	h.ListCases(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCaseHandler_UpdateCase_NotFound_Returns404(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewCaseHandler(crm.NewCaseService(db))

	req := httptest.NewRequest(http.MethodPut, "/api/v1/cases/missing", bytes.NewBufferString(`{"subject":"x"}`))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "missing")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.UpdateCase(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rr.Code, rr.Body.String())
	}
}
