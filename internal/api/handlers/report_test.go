package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
)

func TestReportHandler_GetSalesFunnel_200(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	seedReportDealData(t, db, wsID, ownerID)

	h := NewReportHandler(crm.NewReportService(db))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/sales/funnel", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()

	h.GetSalesFunnel(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := resp["stages"]; !ok {
		t.Fatalf("expected stages field, got: %v", resp)
	}
}

func TestReportHandler_GetSalesFunnel_MissingWorkspace_400(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	h := NewReportHandler(crm.NewReportService(db))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/sales/funnel", nil)
	rr := httptest.NewRecorder()

	h.GetSalesFunnel(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestReportHandler_GetDealAging_200(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	seedReportDealData(t, db, wsID, ownerID)

	h := NewReportHandler(crm.NewReportService(db))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/sales/aging", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()

	h.GetDealAging(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestReportHandler_GetSupportBacklog_InvalidAgingDays_400(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	seedReportCaseData(t, db, wsID, ownerID)

	h := NewReportHandler(crm.NewReportService(db))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/support/backlog?aging_days=bad", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()

	h.GetSupportBacklog(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestReportHandler_GetSupportVolume_200(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	seedReportCaseData(t, db, wsID, ownerID)

	h := NewReportHandler(crm.NewReportService(db))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/support/volume", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()

	h.GetSupportVolume(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestReportHandler_ExportSalesFunnelCSV_200(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	seedReportDealData(t, db, wsID, ownerID)

	h := NewReportHandler(crm.NewReportService(db))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/sales/funnel/export?format=csv", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()

	h.ExportSalesFunnelCSV(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("Content-Type"); got != "text/csv" {
		t.Fatalf("expected text/csv content type, got %q", got)
	}
	if !strings.Contains(rr.Body.String(), "stage,order,deal_count,total_value,probability") {
		t.Fatalf("expected sales funnel csv header, got %q", rr.Body.String())
	}
}

func TestReportHandler_ExportSupportBacklogCSV_200(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	seedReportCaseData(t, db, wsID, ownerID)

	h := NewReportHandler(crm.NewReportService(db))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/support/backlog/export?format=csv", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()

	h.ExportSupportBacklogCSV(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("Content-Type"); got != "text/csv" {
		t.Fatalf("expected text/csv content type, got %q", got)
	}
	if !strings.Contains(rr.Body.String(), "id,priority,status,created_at,aging_days") {
		t.Fatalf("expected support backlog csv header, got %q", rr.Body.String())
	}
}

func TestReportHandler_GetDealAging_MissingWorkspace_400(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	h := NewReportHandler(crm.NewReportService(db))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/sales/aging", nil)
	rr := httptest.NewRecorder()

	h.GetDealAging(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestReportHandler_GetSupportBacklog_200(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	seedReportCaseData(t, db, wsID, ownerID)

	h := NewReportHandler(crm.NewReportService(db))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/support/backlog?aging_days=30", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()

	h.GetSupportBacklog(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestReportHandler_GetSupportBacklog_MissingWorkspace_400(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	h := NewReportHandler(crm.NewReportService(db))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/support/backlog", nil)
	rr := httptest.NewRecorder()

	h.GetSupportBacklog(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestReportHandler_GetSupportVolume_MissingWorkspace_400(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	h := NewReportHandler(crm.NewReportService(db))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/support/volume", nil)
	rr := httptest.NewRecorder()

	h.GetSupportVolume(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestReportHandler_ExportSalesFunnelCSV_MissingFormat_400(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewReportHandler(crm.NewReportService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/sales/funnel/export", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()

	h.ExportSalesFunnelCSV(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestReportHandler_ExportSalesFunnelCSV_MissingWorkspace_400(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	h := NewReportHandler(crm.NewReportService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/sales/funnel/export?format=csv", nil)
	rr := httptest.NewRecorder()

	h.ExportSalesFunnelCSV(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestReportHandler_ExportSupportBacklogCSV_MissingFormat_400(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	h := NewReportHandler(crm.NewReportService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/support/backlog/export", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()

	h.ExportSupportBacklogCSV(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestReportHandler_ExportSupportBacklogCSV_MissingWorkspace_400(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	h := NewReportHandler(crm.NewReportService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/support/backlog/export?format=csv", nil)
	rr := httptest.NewRecorder()

	h.ExportSupportBacklogCSV(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func seedReportDealData(t *testing.T, db *sql.DB, wsID, ownerID string) {
	t.Helper()
	accountID := "acc-r-" + randID()
	pipelineID := "pl-r-" + randID()
	stageID := "st-r-" + randID()
	dealID := "deal-r-" + randID()

	_, err := db.Exec(`INSERT INTO account (id, workspace_id, name, owner_id, created_at, updated_at) VALUES (?, ?, 'R Account', ?, datetime('now'), datetime('now'))`, accountID, wsID, ownerID)
	if err != nil {
		t.Fatalf("seed account: %v", err)
	}
	_, err = db.Exec(`INSERT INTO pipeline (id, workspace_id, name, entity_type, created_at, updated_at) VALUES (?, ?, 'Sales', 'deal', datetime('now'), datetime('now'))`, pipelineID, wsID)
	if err != nil {
		t.Fatalf("seed pipeline: %v", err)
	}
	_, err = db.Exec(`INSERT INTO pipeline_stage (id, pipeline_id, name, position, probability, created_at, updated_at) VALUES (?, ?, 'Discovery', 1, 0.5, datetime('now'), datetime('now'))`, stageID, pipelineID)
	if err != nil {
		t.Fatalf("seed stage: %v", err)
	}
	_, err = db.Exec(`INSERT INTO deal (id, workspace_id, account_id, pipeline_id, stage_id, owner_id, title, amount, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, 'R Deal', 1200, 'open', datetime('now','-2 day'), datetime('now','-1 day'))`, dealID, wsID, accountID, pipelineID, stageID, ownerID)
	if err != nil {
		t.Fatalf("seed deal: %v", err)
	}
}

func seedReportCaseData(t *testing.T, db *sql.DB, wsID, ownerID string) {
	t.Helper()
	caseID := "case-r-" + randID()
	_, err := db.Exec(`INSERT INTO case_ticket (id, workspace_id, owner_id, subject, priority, status, created_at, updated_at) VALUES (?, ?, ?, 'R Case', 'high', 'open', datetime('now','-35 day'), datetime('now'))`, caseID, wsID, ownerID)
	if err != nil {
		t.Fatalf("seed case: %v", err)
	}

	closedID := "case-r-closed-" + randID()
	_, err = db.Exec(`INSERT INTO case_ticket (id, workspace_id, owner_id, subject, priority, status, created_at, updated_at) VALUES (?, ?, ?, 'R Case Closed', 'high', 'closed', datetime('now','-10 day'), datetime('now'))`, closedID, wsID, ownerID)
	if err != nil {
		t.Fatalf("seed closed case: %v", err)
	}
}
