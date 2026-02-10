// Task 1.5: Lead handler tests - TDD approach
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

func TestLeadHandler_CreateLead(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	handler := NewLeadHandler(crm.NewLeadService(db))

	reqBody := map[string]interface{}{
		"source":   "website",
		"status":   "new",
		"ownerId":  ownerID,
		"score":    75.5,
		"metadata": "{}",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/leads", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	w := httptest.NewRecorder()
	handler.CreateLead(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("CreateLead status = %d; want %d", w.Code, http.StatusCreated)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}

	if _, ok := resp["id"]; !ok {
		t.Error("response missing 'id' field")
	}
	if resp["status"] != "new" {
		t.Errorf("response status = %v; want 'new'", resp["status"])
	}
}

func TestLeadHandler_GetLead(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	svc := crm.NewLeadService(db)
	handler := NewLeadHandler(svc)

	created, _ := svc.Create(context.Background(), crm.CreateLeadInput{
		WorkspaceID: wsID,
		Source:      "website",
		Status:      "new",
		OwnerID:     ownerID,
	})

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/leads/%s", created.ID), nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", created.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.GetLead(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetLead status = %d; want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}

	if resp["id"] != created.ID {
		t.Errorf("response id = %v; want %v", resp["id"], created.ID)
	}
}

func TestLeadHandler_GetLeadNotFound(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	handler := NewLeadHandler(crm.NewLeadService(db))

	req := httptest.NewRequest("GET", "/api/v1/leads/nonexistent-id", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent-id")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.GetLead(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("GetLead status = %d; want %d (not found)", w.Code, http.StatusNotFound)
	}
}

func TestLeadHandler_ListLeads(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	svc := crm.NewLeadService(db)
	handler := NewLeadHandler(svc)

	for i := 1; i <= 3; i++ {
		_, err := svc.Create(context.Background(), crm.CreateLeadInput{
			WorkspaceID: wsID,
			Source:      "website",
			Status:      "new",
			OwnerID:     ownerID,
		})
		if err != nil {
			t.Fatalf("seed create lead %d error = %v", i, err)
		}
	}

	req := httptest.NewRequest("GET", "/api/v1/leads?limit=2&offset=0", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	w := httptest.NewRecorder()
	handler.ListLeads(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListLeads status = %d; want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}

	if data, ok := resp["data"]; ok {
		if items, ok := data.([]interface{}); ok && len(items) != 2 {
			t.Errorf("ListLeads returned %d items; want 2", len(items))
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

func TestLeadHandler_ListLeadsByOwner(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	otherOwnerID := createUser(t, db, wsID)
	svc := crm.NewLeadService(db)
	handler := NewLeadHandler(svc)

	// Create leads for different owners
	if _, err := svc.Create(context.Background(), crm.CreateLeadInput{
		WorkspaceID: wsID,
		Source:      "website",
		Status:      "new",
		OwnerID:     ownerID,
	}); err != nil {
		t.Fatalf("seed create owner lead 1 error = %v", err)
	}
	if _, err := svc.Create(context.Background(), crm.CreateLeadInput{
		WorkspaceID: wsID,
		Source:      "referral",
		Status:      "qualified",
		OwnerID:     ownerID,
	}); err != nil {
		t.Fatalf("seed create owner lead 2 error = %v", err)
	}
	if _, err := svc.Create(context.Background(), crm.CreateLeadInput{
		WorkspaceID: wsID,
		Source:      "event",
		Status:      "new",
		OwnerID:     otherOwnerID,
	}); err != nil {
		t.Fatalf("seed create other owner lead error = %v", err)
	}

	// List leads for specific owner
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/leads?owner_id=%s&limit=10", ownerID), nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	w := httptest.NewRecorder()
	handler.ListLeads(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListLeads status = %d; want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}

	if data, ok := resp["data"]; ok {
		if items, ok := data.([]interface{}); ok {
			if len(items) != 2 {
				t.Errorf("ListLeads by owner returned %d items; want 2", len(items))
			}
		}
	}
}

func TestLeadHandler_UpdateLead(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	svc := crm.NewLeadService(db)
	handler := NewLeadHandler(svc)

	created, _ := svc.Create(context.Background(), crm.CreateLeadInput{
		WorkspaceID: wsID,
		Source:      "website",
		Status:      "new",
		OwnerID:     ownerID,
	})

	reqBody := map[string]interface{}{
		"status":  "qualified",
		"ownerId": ownerID,
		"score":   90.0,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/leads/%s", created.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", created.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.UpdateLead(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("UpdateLead status = %d; want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}

	if resp["status"] != "qualified" {
		t.Errorf("response status = %v; want 'qualified'", resp["status"])
	}
}

func TestLeadHandler_DeleteLead(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	svc := crm.NewLeadService(db)
	handler := NewLeadHandler(svc)

	created, _ := svc.Create(context.Background(), crm.CreateLeadInput{
		WorkspaceID: wsID,
		Source:      "website",
		Status:      "new",
		OwnerID:     ownerID,
	})

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/leads/%s", created.ID), nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", created.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.DeleteLead(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("DeleteLead status = %d; want %d (no content)", w.Code, http.StatusNoContent)
	}

	// Verify lead is soft deleted
	_, err := svc.Get(context.Background(), wsID, created.ID)
	if err != sql.ErrNoRows {
		t.Errorf("After delete, Get() error = %v; want sql.ErrNoRows", err)
	}
}
