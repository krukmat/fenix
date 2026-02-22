package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	domainaudit "github.com/matiasleandrokruk/fenix/internal/domain/audit"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

func TestAuditHandler_Query_200_NoFilters(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewAuditHandler(domainaudit.NewAuditService(db))
	seedAuditEvent(t, h.auditService, wsID, "tool.executed", domainaudit.OutcomeSuccess, time.Now())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events?limit=10&offset=0", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()

	h.Query(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := resp["data"]; !ok {
		t.Fatalf("expected data field")
	}
}

func TestAuditHandler_Query_200_WithActionFilter(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewAuditHandler(domainaudit.NewAuditService(db))
	seedAuditEvent(t, h.auditService, wsID, "tool.executed", domainaudit.OutcomeSuccess, time.Now())
	seedAuditEvent(t, h.auditService, wsID, "approval.decided", domainaudit.OutcomeSuccess, time.Now())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events?action=tool.executed", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()

	h.Query(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "tool.executed") {
		t.Fatalf("expected filtered action in response, got %s", rr.Body.String())
	}
}

func TestAuditHandler_Query_200_WithDateRange(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewAuditHandler(domainaudit.NewAuditService(db))
	now := time.Now().UTC()
	seedAuditEvent(t, h.auditService, wsID, "tool.executed", domainaudit.OutcomeSuccess, now.Add(-72*time.Hour))
	seedAuditEvent(t, h.auditService, wsID, "tool.executed", domainaudit.OutcomeSuccess, now)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events?date_from="+now.Add(-1*time.Hour).Format(time.RFC3339), nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()

	h.Query(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAuditHandler_GetByID_200(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewAuditHandler(domainaudit.NewAuditService(db))
	e := seedAuditEvent(t, h.auditService, wsID, "tool.executed", domainaudit.OutcomeSuccess, time.Now())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events/"+e.ID, nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", e.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetByID(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAuditHandler_GetByID_404(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewAuditHandler(domainaudit.NewAuditService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events/missing", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "missing")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetByID(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAuditHandler_Export_200_CSV(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewAuditHandler(domainaudit.NewAuditService(db))
	seedAuditEvent(t, h.auditService, wsID, "tool.executed", domainaudit.OutcomeSuccess, time.Now())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/audit/export?format=csv", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()

	h.Export(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("Content-Type"); got != "text/csv" {
		t.Fatalf("expected text/csv content type, got %q", got)
	}
	if !strings.Contains(rr.Body.String(), "id,workspace_id,actor_id,actor_type,action") {
		t.Fatalf("expected csv header, got %q", rr.Body.String())
	}
}

func TestAuditHandler_Export_400_BadFormat(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewAuditHandler(domainaudit.NewAuditService(db))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/audit/export?format=json", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()

	h.Export(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAuditHandler_Query_MissingWorkspaceID_400(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	h := NewAuditHandler(domainaudit.NewAuditService(db))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events", nil)
	rr := httptest.NewRecorder()

	h.Query(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func seedAuditEvent(
	t *testing.T,
	svc *domainaudit.AuditService,
	wsID string,
	action string,
	outcome domainaudit.Outcome,
	createdAt time.Time,
) *domainaudit.AuditEvent {
	t.Helper()
	e := &domainaudit.AuditEvent{
		ID:          uuid.NewV7().String(),
		WorkspaceID: wsID,
		ActorID:     uuid.NewV7().String(),
		ActorType:   domainaudit.ActorTypeUser,
		Action:      action,
		Outcome:     outcome,
		CreatedAt:   createdAt,
	}
	if err := svc.Log(context.Background(), e); err != nil {
		t.Fatalf("seed audit event: %v", err)
	}
	return e
}
