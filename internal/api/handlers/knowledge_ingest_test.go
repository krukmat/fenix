// Task 2.2: Integration tests for KnowledgeIngestHandler.
// Uses real in-memory SQLite DB with all migrations applied — no mocks.
// Traces: FR-090
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
)

func TestKnowledgeIngestHandler_Success_Returns201(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)

	bus := eventbus.New()
	svc := knowledge.NewIngestService(db, bus)
	handler := NewKnowledgeIngestHandler(svc)

	body, _ := json.Marshal(map[string]interface{}{
		"sourceType": "document",
		"title":      "Test Document",
		"rawContent": "This is the raw content of the document.",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/knowledge/ingest", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	handler.Ingest(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["id"] == "" || resp["id"] == nil {
		t.Error("expected response to include 'id'")
	}
	if resp["workspaceId"] != wsID {
		t.Errorf("expected workspaceId %q, got %v", wsID, resp["workspaceId"])
	}
}

func TestKnowledgeIngestHandler_MissingTitle_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)

	bus := eventbus.New()
	svc := knowledge.NewIngestService(db, bus)
	handler := NewKnowledgeIngestHandler(svc)

	body, _ := json.Marshal(map[string]interface{}{
		"sourceType": "document",
		"rawContent": "content without a title",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/knowledge/ingest", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	handler.Ingest(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing title, got %d", rr.Code)
	}
}

func TestKnowledgeIngestHandler_MissingSourceType_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)

	bus := eventbus.New()
	svc := knowledge.NewIngestService(db, bus)
	handler := NewKnowledgeIngestHandler(svc)

	body, _ := json.Marshal(map[string]interface{}{
		"title":      "Some Title",
		"rawContent": "content without sourceType",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/knowledge/ingest", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	handler.Ingest(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing sourceType, got %d", rr.Code)
	}
}

func TestKnowledgeIngestHandler_InvalidSourceType_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)

	bus := eventbus.New()
	svc := knowledge.NewIngestService(db, bus)
	handler := NewKnowledgeIngestHandler(svc)

	body, _ := json.Marshal(map[string]interface{}{
		"sourceType": "unknown_type",
		"title":      "Some Title",
		"rawContent": "content",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/knowledge/ingest", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	handler.Ingest(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid sourceType, got %d", rr.Code)
	}
}

func TestKnowledgeIngestHandler_NoWorkspaceContext_Returns401(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)

	bus := eventbus.New()
	svc := knowledge.NewIngestService(db, bus)
	handler := NewKnowledgeIngestHandler(svc)

	body, _ := json.Marshal(map[string]interface{}{
		"sourceType": "document",
		"title":      "Title",
		"rawContent": "content",
	})

	// No workspace ID in context
	req := httptest.NewRequest(http.MethodPost, "/api/v1/knowledge/ingest", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Ingest(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without workspace context, got %d", rr.Code)
	}
}
