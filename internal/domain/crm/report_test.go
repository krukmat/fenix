package crm_test

import (
	"context"
	"database/sql"
	"io"
	"strings"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
)

func TestReportService_GetSalesFunnel(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	seedDealForReports(t, db, wsID, ownerID)

	svc := crm.NewReportService(db)
	report, err := svc.GetSalesFunnel(context.Background(), wsID)
	if err != nil {
		t.Fatalf("GetSalesFunnel() error = %v", err)
	}
	if len(report.Stages) == 0 {
		t.Fatalf("expected at least one stage")
	}
}

func TestReportService_GetDealAging(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	seedDealForReports(t, db, wsID, ownerID)

	svc := crm.NewReportService(db)
	rows, err := svc.GetDealAging(context.Background(), wsID)
	if err != nil {
		t.Fatalf("GetDealAging() error = %v", err)
	}
	if len(rows) == 0 {
		t.Fatalf("expected aging rows")
	}
}

func TestReportService_GetCaseVolume(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	seedCaseForReports(t, db, wsID, ownerID)

	svc := crm.NewReportService(db)
	rows, err := svc.GetCaseVolume(context.Background(), wsID)
	if err != nil {
		t.Fatalf("GetCaseVolume() error = %v", err)
	}
	if len(rows) == 0 {
		t.Fatalf("expected case volume rows")
	}
}

func TestReportService_GetSupportBacklog(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	seedCaseForReports(t, db, wsID, ownerID)

	svc := crm.NewReportService(db)
	report, err := svc.GetSupportBacklog(context.Background(), wsID, 30)
	if err != nil {
		t.Fatalf("GetSupportBacklog() error = %v", err)
	}
	if report.OpenTotal == 0 {
		t.Fatalf("expected open backlog")
	}
	if len(report.AgingBuckets) != 3 {
		t.Fatalf("expected 3 buckets")
	}
}

func TestReportService_ExportSalesFunnelCSV(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	seedDealForReports(t, db, wsID, ownerID)

	svc := crm.NewReportService(db)
	r, err := svc.ExportSalesFunnelCSV(context.Background(), wsID)
	if err != nil {
		t.Fatalf("ExportSalesFunnelCSV() error = %v", err)
	}
	b, _ := io.ReadAll(r)
	if !strings.Contains(string(b), "stage,order,deal_count,total_value,probability") {
		t.Fatalf("expected csv header, got %s", string(b))
	}
}

func TestReportService_ExportSupportBacklogCSV(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	seedCaseForReports(t, db, wsID, ownerID)

	svc := crm.NewReportService(db)
	r, err := svc.ExportSupportBacklogCSV(context.Background(), wsID)
	if err != nil {
		t.Fatalf("ExportSupportBacklogCSV() error = %v", err)
	}
	b, _ := io.ReadAll(r)
	if !strings.Contains(string(b), "id,priority,status,created_at,aging_days") {
		t.Fatalf("expected csv header, got %s", string(b))
	}
}

func seedDealForReports(t *testing.T, db DBTX, wsID, ownerID string) {
	t.Helper()
	accountID := "acc-" + randID()
	pipelineID := "pl-" + randID()
	stageID := "st-" + randID()
	dealID := "deal-" + randID()

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

func seedCaseForReports(t *testing.T, db DBTX, wsID, ownerID string) {
	t.Helper()
	caseID := "case-" + randID()
	_, err := db.Exec(`INSERT INTO case_ticket (id, workspace_id, owner_id, subject, priority, status, created_at, updated_at) VALUES (?, ?, ?, 'R Case', 'high', 'open', datetime('now','-40 day'), datetime('now'))`, caseID, wsID, ownerID)
	if err != nil {
		t.Fatalf("seed case: %v", err)
	}
}

type DBTX interface {
	Exec(query string, args ...any) (sql.Result, error)
}

func TestReportService_GetSupportBacklog_EmptyMTTR(t *testing.T) {
	t.Parallel()
	db := mustOpenDBWithMigrations(t)
	wsID, _ := setupWorkspaceAndOwner(t, db)
	// No closed cases → AvgResolutionDays is NULL → safeFloat64Ptr(nil) path
	svc := crm.NewReportService(db)
	report, err := svc.GetSupportBacklog(context.Background(), wsID, 30)
	if err != nil {
		t.Fatalf("GetSupportBacklog: %v", err)
	}
	if report == nil {
		t.Fatal("expected non-nil report")
	}
}
