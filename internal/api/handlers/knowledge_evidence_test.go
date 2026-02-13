// Task 2.6: Integration tests for KnowledgeEvidenceHandler.
// Uses real in-memory SQLite DB with migrations applied.
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

func TestKnowledgeEvidenceHandler_Success_Returns200(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)

	stub := &searchStubLLM{}
	bus := eventbus.New()
	ingestSvc := knowledge.NewIngestService(db, bus)
	embedder := knowledge.NewEmbedderService(db, stub)
	searchSvc := knowledge.NewSearchService(db, stub)
	evidenceSvc := knowledge.NewEvidencePackService(db, searchSvc, knowledge.DefaultEvidenceConfig())
	handler := NewKnowledgeEvidenceHandler(evidenceSvc)

	item, err := ingestSvc.Ingest(contextWithWorkspaceID(t.Context(), wsID), knowledge.CreateKnowledgeItemInput{
		WorkspaceID: wsID,
		SourceType:  knowledge.SourceTypeDocument,
		Title:       "Pricing Evidence",
		RawContent:  "pricing strategy and enterprise discounts",
	})
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	if err := embedder.EmbedChunks(t.Context(), item.ID, wsID); err != nil {
		t.Fatalf("embed failed: %v", err)
	}

	body, _ := json.Marshal(map[string]interface{}{
		"query": "pricing",
		"limit": 5,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/knowledge/evidence", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	handler.Build(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if _, ok := resp["data"]; !ok {
		t.Fatal("expected 'data' field")
	}
	data := resp["data"].(map[string]interface{})
	if data["confidence"] == "" {
		t.Error("expected confidence in response")
	}
}

func TestKnowledgeEvidenceHandler_MissingWorkspace_Returns401(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	stub := &searchStubLLM{}
	searchSvc := knowledge.NewSearchService(db, stub)
	evidenceSvc := knowledge.NewEvidencePackService(db, searchSvc, knowledge.DefaultEvidenceConfig())
	handler := NewKnowledgeEvidenceHandler(evidenceSvc)

	body, _ := json.Marshal(map[string]interface{}{"query": "pricing"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/knowledge/evidence", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Build(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestKnowledgeEvidenceHandler_InvalidJSON_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)

	stub := &searchStubLLM{}
	searchSvc := knowledge.NewSearchService(db, stub)
	evidenceSvc := knowledge.NewEvidencePackService(db, searchSvc, knowledge.DefaultEvidenceConfig())
	handler := NewKnowledgeEvidenceHandler(evidenceSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/knowledge/evidence", bytes.NewBufferString(`{"query":`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	handler.Build(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestKnowledgeEvidenceHandler_MissingQuery_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)

	stub := &searchStubLLM{}
	searchSvc := knowledge.NewSearchService(db, stub)
	evidenceSvc := knowledge.NewEvidencePackService(db, searchSvc, knowledge.DefaultEvidenceConfig())
	handler := NewKnowledgeEvidenceHandler(evidenceSvc)

	body, _ := json.Marshal(map[string]interface{}{"limit": 10})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/knowledge/evidence", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	handler.Build(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}
