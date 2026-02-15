// Task 1.3.6: TDD tests for Account HTTP handlers
// Traces: FR-001, FR-070
package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
)

// TestAccountHandler_CreateAccount tests POST /api/v1/accounts
func TestAccountHandler_CreateAccount(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	handler := NewAccountHandler(crm.NewAccountService(db))

	reqBody := map[string]interface{}{
		"name":        "Test Account",
		"domain":      "test.com",
		"industry":    "Technology",
		"sizeSegment": "mid",
		"ownerId":     ownerID,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/accounts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	w := httptest.NewRecorder()
	handler.CreateAccount(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("CreateAccount status = %d; want %d", w.Code, http.StatusCreated)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}

	if _, ok := resp["id"]; !ok {
		t.Error("response missing 'id' field")
	}
	if resp["name"] != "Test Account" {
		t.Errorf("response name = %v; want 'Test Account'", resp["name"])
	}
}

func TestAccountHandler_CreateAccount_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	_, ownerID := setupWorkspaceAndOwner(t, db)
	handler := NewAccountHandler(crm.NewAccountService(db))

	body, _ := json.Marshal(map[string]any{"name": "A", "ownerId": ownerID})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.CreateAccount(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=%d", w.Code, http.StatusBadRequest)
	}
}

func TestAccountHandler_CreateAccount_InvalidJSON_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	handler := NewAccountHandler(crm.NewAccountService(db))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts", bytes.NewBufferString(`{"name":`))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	w := httptest.NewRecorder()
	handler.CreateAccount(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=%d", w.Code, http.StatusBadRequest)
	}
}

func TestAccountHandler_CreateAccount_MissingRequired_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	handler := NewAccountHandler(crm.NewAccountService(db))

	body, _ := json.Marshal(map[string]any{"name": "A"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	w := httptest.NewRecorder()
	handler.CreateAccount(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=%d", w.Code, http.StatusBadRequest)
	}
}

// TestAccountHandler_GetAccount tests GET /api/v1/accounts/:id
func TestAccountHandler_GetAccount(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	svc := crm.NewAccountService(db)
	handler := NewAccountHandler(svc)

	// Create an account first
	created, _ := svc.Create(context.Background(), crm.CreateAccountInput{
		WorkspaceID: wsID,
		Name:        "Test Account",
		OwnerID:     ownerID,
	})

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/accounts/%s", created.ID), nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	// Set URL parameters using chi.URLParam pattern
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", created.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.GetAccount(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetAccount status = %d; want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}

	if resp["id"] != created.ID {
		t.Errorf("response id = %v; want %v", resp["id"], created.ID)
	}
	if resp["name"] != "Test Account" {
		t.Errorf("response name = %v; want 'Test Account'", resp["name"])
	}
}

// TestAccountHandler_GetAccountNotFound tests GET for non-existent account
func TestAccountHandler_GetAccountNotFound(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	handler := NewAccountHandler(crm.NewAccountService(db))

	req := httptest.NewRequest("GET", "/api/v1/accounts/nonexistent-id", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent-id")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.GetAccount(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("GetAccount status = %d; want %d (not found)", w.Code, http.StatusNotFound)
	}
}

// TestAccountHandler_ListAccounts tests GET /api/v1/accounts with pagination
func TestAccountHandler_ListAccounts(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	svc := crm.NewAccountService(db)
	handler := NewAccountHandler(svc)

	// Create 3 accounts
	for i := 1; i <= 3; i++ {
		_, err := svc.Create(context.Background(), crm.CreateAccountInput{
			WorkspaceID: wsID,
			Name:        fmt.Sprintf("Account %d", i),
			OwnerID:     ownerID,
		})
		if err != nil {
			t.Fatalf("seed create account %d error = %v", i, err)
		}
	}

	req := httptest.NewRequest("GET", "/api/v1/accounts?limit=2&offset=0", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	w := httptest.NewRecorder()
	handler.ListAccounts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListAccounts status = %d; want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}

	if data, ok := resp["data"]; ok {
		if items, ok := data.([]interface{}); ok && len(items) != 2 {
			t.Errorf("ListAccounts returned %d items; want 2", len(items))
		}
	} else {
		t.Error("response missing 'data' field")
	}

	if meta, ok := resp["meta"]; ok {
		metaObj := meta.(map[string]interface{})
		if metaObj["total"] != float64(3) {
			t.Errorf("response total = %v; want 3", metaObj["total"])
		}
	} else {
		t.Error("response missing 'meta' field")
	}
}

// TestAccountHandler_UpdateAccount tests PUT /api/v1/accounts/:id
func TestAccountHandler_UpdateAccount(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	svc := crm.NewAccountService(db)
	handler := NewAccountHandler(svc)

	// Create an account first
	created, _ := svc.Create(context.Background(), crm.CreateAccountInput{
		WorkspaceID: wsID,
		Name:        "Original Name",
		OwnerID:     ownerID,
	})

	reqBody := map[string]interface{}{
		"name":     "Updated Name",
		"industry": "Finance",
		"ownerId":  ownerID,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/accounts/%s", created.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", created.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.UpdateAccount(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("UpdateAccount status = %d; want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}

	if resp["name"] != "Updated Name" {
		t.Errorf("response name = %v; want 'Updated Name'", resp["name"])
	}
}

// TestAccountHandler_DeleteAccount tests DELETE /api/v1/accounts/:id (soft delete)
func TestAccountHandler_DeleteAccount(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	svc := crm.NewAccountService(db)
	handler := NewAccountHandler(svc)

	// Create an account first
	created, _ := svc.Create(context.Background(), crm.CreateAccountInput{
		WorkspaceID: wsID,
		Name:        "To Delete",
		OwnerID:     ownerID,
	})

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/accounts/%s", created.ID), nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", created.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.DeleteAccount(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("DeleteAccount status = %d; want %d (no content)", w.Code, http.StatusNoContent)
	}

	// Verify account is soft deleted (Get should fail)
	_, err := svc.Get(context.Background(), wsID, created.ID)
	if err != sql.ErrNoRows {
		t.Errorf("After delete, Get() error = %v; want sql.ErrNoRows", err)
	}
}

// TestAccountHandler_DeleteAlreadyDeleted tests TD-3: DELETE on soft-deleted account returns 404
func TestAccountHandler_DeleteAlreadyDeleted(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	svc := crm.NewAccountService(db)
	handler := NewAccountHandler(svc)

	// Create and immediately delete
	created, _ := svc.Create(context.Background(), crm.CreateAccountInput{
		WorkspaceID: wsID,
		Name:        "Already Deleted",
		OwnerID:     ownerID,
	})
	if err := svc.Delete(context.Background(), wsID, created.ID); err != nil {
		t.Fatalf("seed delete account error = %v", err)
	}

	// Try to delete again
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/accounts/%s", created.ID), nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", created.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.DeleteAccount(w, req)

	// Should return 404 not 204 or 500 (TD-3 fix)
	if w.Code != http.StatusNotFound {
		t.Errorf("Delete(already deleted) status = %d; want %d (not found)", w.Code, http.StatusNotFound)
	}
}

// TestAccountHandler_ListAccounts_LimitCapped tests TD-2: limit > 100 is capped to 100
func TestAccountHandler_ListAccounts_LimitCapped(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	svc := crm.NewAccountService(db)
	handler := NewAccountHandler(svc)

	// Create 3 accounts
	for i := 1; i <= 3; i++ {
		_, err := svc.Create(context.Background(), crm.CreateAccountInput{
			WorkspaceID: wsID,
			Name:        fmt.Sprintf("Cap Test Account %d", i),
			OwnerID:     ownerID,
		})
		if err != nil {
			t.Fatalf("seed create cap test account %d error = %v", i, err)
		}
	}

	req := httptest.NewRequest("GET", "/api/v1/accounts?limit=999999&offset=0", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	w := httptest.NewRecorder()
	handler.ListAccounts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListAccounts status = %d; want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}

	// Even with limit=999999, meta.limit should be capped at 100 (TD-2 fix)
	meta := resp["meta"].(map[string]interface{})
	if meta["limit"] != float64(100) {
		t.Errorf("meta.limit = %v; want 100 (cap enforced)", meta["limit"])
	}
}

func TestAccountHandler_GetAccount_MissingID_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	handler := NewAccountHandler(crm.NewAccountService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.GetAccount(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=%d", w.Code, http.StatusBadRequest)
	}
}

func TestAccountHandler_UpdateAccount_InvalidJSON_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	svc := crm.NewAccountService(db)
	handler := NewAccountHandler(svc)

	created, _ := svc.Create(context.Background(), crm.CreateAccountInput{WorkspaceID: wsID, Name: "A", OwnerID: ownerID})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/accounts/"+created.ID, bytes.NewBufferString(`{"name":`))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", created.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.UpdateAccount(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=%d", w.Code, http.StatusBadRequest)
	}
}

func TestAccountHandler_UpdateAccount_MissingID_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	handler := NewAccountHandler(crm.NewAccountService(db))

	req := httptest.NewRequest(http.MethodPut, "/api/v1/accounts/", bytes.NewBufferString(`{"name":"x"}`))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.UpdateAccount(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=%d", w.Code, http.StatusBadRequest)
	}
}

func TestAccountHandler_DeleteAccount_MissingID_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	handler := NewAccountHandler(crm.NewAccountService(db))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/accounts/", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.DeleteAccount(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=%d", w.Code, http.StatusBadRequest)
	}
}

func TestFormatDeletedAt(t *testing.T) {
	t.Parallel()

	if got := formatDeletedAt(nil); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}

	ts := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	got := formatDeletedAt(&ts)
	if got == nil || *got == "" {
		t.Fatalf("expected formatted timestamp, got %v", got)
	}
}

// --- helpers ---

// contextWithWorkspaceID adds workspace_id to the request context.
// Uses ctxkeys.WorkspaceID to match the exact key the middleware injects (TD-1 fix).
func contextWithWorkspaceID(ctx context.Context, wsID string) context.Context {
	return context.WithValue(ctx, ctxkeys.WorkspaceID, wsID)
}

// mustOpenDBWithMigrations opens an in-memory DB with migrations applied.
func mustOpenDBWithMigrations(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sqlite.NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB error = %v", err)
	}
	// IMPORTANT: :memory: databases are per-connection in SQLite.
	// Force a single connection so migrations and subsequent queries
	// always run against the same in-memory DB.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() { db.Close() })

	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp error = %v", err)
	}

	return db
}

// createWorkspace creates a test workspace.
func createWorkspace(t *testing.T, db *sql.DB) string {
	t.Helper()
	id := "ws-" + randID()
	_, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES (?, ?, ?, datetime('now'), datetime('now'))
	`, id, "Test Workspace", "test-"+randID())
	if err != nil {
		t.Fatalf("createWorkspace error = %v", err)
	}
	return id
}

// createUser creates a test user in a workspace.
func createUser(t *testing.T, db *sql.DB, workspaceID string) string {
	t.Helper()
	id := "user-" + randID()
	_, err := db.Exec(`
		INSERT INTO user_account (id, workspace_id, email, display_name, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'active', datetime('now'), datetime('now'))
	`, id, workspaceID, "user-"+randID()+"@example.com", "Test User")
	if err != nil {
		t.Fatalf("createUser error = %v", err)
	}
	return id
}

// setupWorkspaceAndOwner creates both a workspace and an owner user.
func setupWorkspaceAndOwner(t *testing.T, db *sql.DB) (workspaceID, ownerID string) {
	t.Helper()
	wsID := createWorkspace(t, db)
	userID := createUser(t, db, wsID)
	return wsID, userID
}

// randID generates a unique random string for test IDs using time + counter.
var randIDCounter int64 = 0

func randID() string {
	n := atomic.AddInt64(&randIDCounter, 1)
	return time.Now().Format("20060102150405") + "-" + fmt.Sprintf("%d", n)
}
