package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/audit"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
)

func TestKnowledgeReindexHandler_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)

	bus := eventbus.New()
	ingest := knowledge.NewIngestService(db, bus)
	reindex := knowledge.NewReindexService(db, bus, ingest, audit.NewAuditService(db))
	h := NewKnowledgeReindexHandler(reindex)

	body, _ := json.Marshal(map[string]any{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/knowledge/reindex", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	h.Reindex(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestKnowledgeReindexHandler_InvalidJSON_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)

	bus := eventbus.New()
	ingest := knowledge.NewIngestService(db, bus)
	reindex := knowledge.NewReindexService(db, bus, ingest, audit.NewAuditService(db))
	h := NewKnowledgeReindexHandler(reindex)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/knowledge/reindex", bytes.NewBufferString(`{"entityType":`))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	h.Reindex(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestKnowledgeReindexHandler_MissingWorkspace_Returns401(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)

	bus := eventbus.New()
	ingest := knowledge.NewIngestService(db, bus)
	reindex := knowledge.NewReindexService(db, bus, ingest, audit.NewAuditService(db))
	h := NewKnowledgeReindexHandler(reindex)

	body, _ := json.Marshal(map[string]any{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/knowledge/reindex", bytes.NewReader(body))
	req = req.WithContext(context.Background())

	rr := httptest.NewRecorder()
	h.Reindex(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}
