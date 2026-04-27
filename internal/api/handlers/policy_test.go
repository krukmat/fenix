// GO-POLICY-READ-01: Read-only HTTP handlers for policy_set and policy_version.
package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

// seedPolicySet inserts a policy_set row and returns its id.
func seedPolicySet(t *testing.T, db *sql.DB, wsID, name string, isActive int) string {
	t.Helper()
	id := uuid.NewV7().String()
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO policy_set (id, workspace_id, name, description, is_active, created_by, created_at, updated_at)
		VALUES (?, ?, ?, '', ?, 'test', ?, ?)`,
		id, wsID, name, isActive,
		time.Now().UTC().Format(time.RFC3339),
		time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("seedPolicySet: %v", err)
	}
	return id
}

// seedPolicyVersion inserts a policy_version row and returns its id.
func seedPolicyVersion(t *testing.T, db *sql.DB, wsID, setID string, versionNum int, status string) string {
	t.Helper()
	id := uuid.NewV7().String()
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO policy_version (id, policy_set_id, workspace_id, version_number, policy_json, status, created_by, created_at)
		VALUES (?, ?, ?, ?, '{"rules":[]}', ?, 'test', ?)`,
		id, setID, wsID, versionNum, status,
		time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("seedPolicyVersion: %v", err)
	}
	return id
}

// --- ListPolicySets ---

func TestPolicyHandler_ListPolicySets_200_EmptyList(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewPolicyHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/policy/sets", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()

	h.ListPolicySets(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := resp["data"]; !ok {
		t.Fatal("expected data field")
	}
}

func TestPolicyHandler_ListPolicySets_200_ReturnsSets(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewPolicyHandler(db)

	seedPolicySet(t, db, wsID, "policy-alpha", 1)
	seedPolicySet(t, db, wsID, "policy-beta", 0)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/policy/sets", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()

	h.ListPolicySets(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "policy-alpha") || !strings.Contains(body, "policy-beta") {
		t.Fatalf("expected both sets in response, got %s", body)
	}
}

func TestPolicyHandler_ListPolicySets_200_FilterByIsActive(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewPolicyHandler(db)

	seedPolicySet(t, db, wsID, "active-set", 1)
	seedPolicySet(t, db, wsID, "inactive-set", 0)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/policy/sets?is_active=true", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()

	h.ListPolicySets(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "active-set") {
		t.Fatalf("expected active-set in response, got %s", body)
	}
	if strings.Contains(body, "inactive-set") {
		t.Fatalf("inactive-set must not appear when is_active=true, got %s", body)
	}
}

func TestPolicyHandler_ListPolicySets_400_MissingWorkspaceID(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	h := NewPolicyHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/policy/sets", nil)
	rr := httptest.NewRecorder()

	h.ListPolicySets(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestPolicyHandler_ListPolicySets_200_CrossTenantIsolation(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID1, _ := setupWorkspaceAndOwner(t, db)
	wsID2, _ := setupWorkspaceAndOwner(t, db)
	h := NewPolicyHandler(db)

	seedPolicySet(t, db, wsID1, "ws1-policy", 1)
	seedPolicySet(t, db, wsID2, "ws2-policy", 1)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/policy/sets", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID1))
	rr := httptest.NewRecorder()

	h.ListPolicySets(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if strings.Contains(body, "ws2-policy") {
		t.Fatalf("cross-tenant leak: ws2-policy must not appear for ws1, got %s", body)
	}
}

// --- GetPolicyVersions ---

func TestPolicyHandler_GetPolicyVersions_200_ReturnsVersions(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewPolicyHandler(db)

	setID := seedPolicySet(t, db, wsID, "versioned-set", 1)
	seedPolicyVersion(t, db, wsID, setID, 1, "active")
	seedPolicyVersion(t, db, wsID, setID, 2, "draft")

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/policy/sets/%s/versions", setID), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", setID)
	req = req.WithContext(context.WithValue(contextWithWorkspaceID(req.Context(), wsID), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetPolicyVersions(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	items, ok := resp["data"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("expected 2 versions, got %v", resp["data"])
	}
}

func TestPolicyHandler_GetPolicyVersions_400_MissingWorkspaceID(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	h := NewPolicyHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/policy/sets/some-id/versions", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "some-id")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetPolicyVersions(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestPolicyHandler_GetPolicyVersions_400_MissingSetID(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewPolicyHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/policy/sets//versions", nil)
	// No chi route context injected — chi.URLParam returns ""
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()

	h.GetPolicyVersions(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestPolicyHandler_GetPolicyVersions_200_EmptyVersions(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewPolicyHandler(db)

	setID := seedPolicySet(t, db, wsID, "empty-set", 1)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/policy/sets/%s/versions", setID), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", setID)
	req = req.WithContext(context.WithValue(contextWithWorkspaceID(req.Context(), wsID), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetPolicyVersions(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestPolicyHandler_GetPolicyVersions_200_CrossTenantIsolation(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID1, _ := setupWorkspaceAndOwner(t, db)
	wsID2, _ := setupWorkspaceAndOwner(t, db)
	h := NewPolicyHandler(db)

	setID := seedPolicySet(t, db, wsID1, "ws1-set", 1)
	otherSetID := seedPolicySet(t, db, wsID2, "ws2-set", 1)
	seedPolicyVersion(t, db, wsID2, otherSetID, 1, "active")

	// Query ws1's set — must return 0 versions, not ws2's
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/policy/sets/%s/versions", setID), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", setID)
	req = req.WithContext(context.WithValue(contextWithWorkspaceID(req.Context(), wsID1), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetPolicyVersions(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	data, _ := resp["data"].([]any)
	if len(data) != 0 {
		t.Fatalf("cross-tenant leak: expected 0 versions for ws1 set, got %d", len(data))
	}
}
