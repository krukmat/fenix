// Task 1.4: TDD tests for Contact HTTP handlers
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

func TestContactHandler_CreateContact(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	accountID := createAccountForHandler(t, db, wsID, ownerID)
	handler := NewContactHandler(crm.NewContactService(db))

	reqBody := map[string]interface{}{
		"accountId": accountID,
		"firstName": "Ada",
		"lastName":  "Lovelace",
		"ownerId":   ownerID,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/contacts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	w := httptest.NewRecorder()
	handler.CreateContact(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("CreateContact status = %d; want %d", w.Code, http.StatusCreated)
	}
}

func TestContactHandler_GetContact(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	accountID := createAccountForHandler(t, db, wsID, ownerID)
	svc := crm.NewContactService(db)
	handler := NewContactHandler(svc)

	created, _ := svc.Create(context.Background(), crm.CreateContactInput{
		WorkspaceID: wsID,
		AccountID:   accountID,
		FirstName:   "Grace",
		LastName:    "Hopper",
		OwnerID:     ownerID,
	})

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/contacts/%s", created.ID), nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", created.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.GetContact(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetContact status = %d; want %d", w.Code, http.StatusOK)
	}
}

func TestContactHandler_ListContacts(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	accountID := createAccountForHandler(t, db, wsID, ownerID)
	svc := crm.NewContactService(db)
	handler := NewContactHandler(svc)

	for i := 0; i < 3; i++ {
		_, _ = svc.Create(context.Background(), crm.CreateContactInput{
			WorkspaceID: wsID,
			AccountID:   accountID,
			FirstName:   fmt.Sprintf("Name%d", i),
			LastName:    "User",
			OwnerID:     ownerID,
		})
	}

	req := httptest.NewRequest("GET", "/api/v1/contacts?limit=2&offset=0", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	w := httptest.NewRecorder()
	handler.ListContacts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListContacts status = %d; want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}
	if data, ok := resp["data"].([]interface{}); ok {
		if len(data) != 2 {
			t.Errorf("ListContacts data length = %d; want 2", len(data))
		}
	}
}

func TestContactHandler_ListContactsByAccount(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	accountA := createAccountForHandler(t, db, wsID, ownerID)
	accountB := createAccountForHandler(t, db, wsID, ownerID)
	svc := crm.NewContactService(db)
	handler := NewContactHandler(svc)

	_, _ = svc.Create(context.Background(), crm.CreateContactInput{
		WorkspaceID: wsID,
		AccountID:   accountA,
		FirstName:   "A1",
		LastName:    "User",
		OwnerID:     ownerID,
	})
	_, _ = svc.Create(context.Background(), crm.CreateContactInput{
		WorkspaceID: wsID,
		AccountID:   accountA,
		FirstName:   "A2",
		LastName:    "User",
		OwnerID:     ownerID,
	})
	_, _ = svc.Create(context.Background(), crm.CreateContactInput{
		WorkspaceID: wsID,
		AccountID:   accountB,
		FirstName:   "B1",
		LastName:    "User",
		OwnerID:     ownerID,
	})

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/accounts/%s/contacts", accountA), nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("account_id", accountA)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.ListContactsByAccount(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListContactsByAccount status = %d; want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}
	if data, ok := resp["data"].([]interface{}); ok {
		if len(data) != 2 {
			t.Errorf("ListContactsByAccount data length = %d; want 2", len(data))
		}
	}
}

func TestContactHandler_UpdateContact(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	accountID := createAccountForHandler(t, db, wsID, ownerID)
	svc := crm.NewContactService(db)
	handler := NewContactHandler(svc)

	created, _ := svc.Create(context.Background(), crm.CreateContactInput{
		WorkspaceID: wsID,
		AccountID:   accountID,
		FirstName:   "Old",
		LastName:    "Name",
		OwnerID:     ownerID,
	})

	reqBody := map[string]interface{}{
		"firstName": "New",
		"lastName":  "Name",
		"ownerId":   ownerID,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/contacts/%s", created.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", created.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.UpdateContact(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("UpdateContact status = %d; want %d", w.Code, http.StatusOK)
	}
}

func TestContactHandler_DeleteContact(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	accountID := createAccountForHandler(t, db, wsID, ownerID)
	svc := crm.NewContactService(db)
	handler := NewContactHandler(svc)

	created, _ := svc.Create(context.Background(), crm.CreateContactInput{
		WorkspaceID: wsID,
		AccountID:   accountID,
		FirstName:   "To",
		LastName:    "Delete",
		OwnerID:     ownerID,
	})

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/contacts/%s", created.ID), nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", created.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.DeleteContact(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("DeleteContact status = %d; want %d", w.Code, http.StatusNoContent)
	}

	_, err := svc.Get(context.Background(), wsID, created.ID)
	if err != sql.ErrNoRows {
		t.Errorf("After delete, Get() error = %v; want sql.ErrNoRows", err)
	}
}

func TestContactHandler_GetContact_MissingID_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	handler := NewContactHandler(crm.NewContactService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts/", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.GetContact(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=%d", w.Code, http.StatusBadRequest)
	}
}

func TestContactHandler_DeleteContact_NotFound_Returns404(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	handler := NewContactHandler(crm.NewContactService(db))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/contacts/missing", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "missing")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.DeleteContact(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d want=%d", w.Code, http.StatusNotFound)
	}
}

func createAccountForHandler(t *testing.T, db *sql.DB, workspaceID, ownerID string) string {
	t.Helper()
	id := "acc-h-" + randID()
	_, err := db.Exec(`
		INSERT INTO account (id, workspace_id, name, owner_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))
	`, id, workspaceID, "Account "+id, ownerID)
	if err != nil {
		t.Fatalf("createAccountForHandler error = %v", err)
	}
	return id
}
