// Task 2.5: Integration tests for KnowledgeSearchHandler.
// Uses real in-memory SQLite DB with all migrations applied — no mocks.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
	"github.com/matiasleandrokruk/fenix/internal/infra/llm"
)

// searchStubLLM implements llm.LLMProvider for handler tests.
// Returns deterministic 3-dim vectors — no real Ollama required.
type searchStubLLM struct{}

func (s *searchStubLLM) Embed(_ context.Context, req llm.EmbedRequest) (*llm.EmbedResponse, error) {
	vecs := make([][]float32, len(req.Texts))
	for i := range vecs {
		vecs[i] = []float32{float32(i+1) * 0.1, float32(i+1) * 0.2, float32(i+1) * 0.3}
	}
	return &llm.EmbedResponse{Embeddings: vecs}, nil
}

func (s *searchStubLLM) ChatCompletion(_ context.Context, _ llm.ChatRequest) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{Content: "stub"}, nil
}

func (s *searchStubLLM) ModelInfo() llm.ModelMeta {
	return llm.ModelMeta{ID: "stub-search", Provider: "stub"}
}

func (s *searchStubLLM) HealthCheck(_ context.Context) error { return nil }

func TestKnowledgeSearchHandler_Success_Returns200(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)

	bus := eventbus.New()
	ingestSvc := knowledge.NewIngestService(db, bus)

	// Ingest a document so there's something to search
	_, err := ingestSvc.Ingest(contextWithWorkspaceID(t.Context(), wsID), knowledge.CreateKnowledgeItemInput{
		WorkspaceID: wsID,
		SourceType:  knowledge.SourceTypeDocument,
		Title:       "Pricing Strategy",
		RawContent:  "our pricing discount policy for enterprise customers",
	})
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	stub := &searchStubLLM{}
	searchSvc := knowledge.NewSearchService(db, stub)
	handler := NewKnowledgeSearchHandler(searchSvc)

	body, _ := json.Marshal(map[string]interface{}{
		"query": "pricing",
		"limit": 10,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/knowledge/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	handler.Search(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["query"] != "pricing" {
		t.Errorf("expected query 'pricing', got %v", resp["query"])
	}
	if _, ok := resp["results"]; !ok {
		t.Error("expected 'results' field in response")
	}
}

// TestKnowledgeSearchHandler_InvalidJSON_Returns400 covers the json.Decode error branch
// in Search handler — malformed body should return 400 (Task 2.5 audit).
func TestKnowledgeSearchHandler_InvalidJSON_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)

	stub := &searchStubLLM{}
	searchSvc := knowledge.NewSearchService(db, stub)
	handler := NewKnowledgeSearchHandler(searchSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/knowledge/search",
		bytes.NewBufferString(`{not valid json`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	handler.Search(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON body, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestKnowledgeSearchHandler_MissingQuery_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)

	stub := &searchStubLLM{}
	searchSvc := knowledge.NewSearchService(db, stub)
	handler := NewKnowledgeSearchHandler(searchSvc)

	body, _ := json.Marshal(map[string]interface{}{
		"limit": 10,
		// "query" intentionally missing
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/knowledge/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	handler.Search(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing query, got %d", rr.Code)
	}
}

func TestKnowledgeSearchHandler_MissingWorkspace_Returns401(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)

	stub := &searchStubLLM{}
	searchSvc := knowledge.NewSearchService(db, stub)
	handler := NewKnowledgeSearchHandler(searchSvc)

	body, _ := json.Marshal(map[string]interface{}{"query": "test"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/knowledge/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No workspace context injected → 401

	rr := httptest.NewRecorder()
	handler.Search(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without workspace context, got %d", rr.Code)
	}
}

func TestKnowledgeSearchHandler_EmptyIndex_Returns200WithNoResults(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)

	stub := &searchStubLLM{}
	searchSvc := knowledge.NewSearchService(db, stub)
	handler := NewKnowledgeSearchHandler(searchSvc)

	body, _ := json.Marshal(map[string]interface{}{"query": "anything"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/knowledge/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	handler.Search(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on empty index, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	results, ok := resp["results"].([]interface{})
	if !ok {
		t.Fatalf("expected results to be a list, got %T", resp["results"])
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results on empty index, got %d", len(results))
	}
}
