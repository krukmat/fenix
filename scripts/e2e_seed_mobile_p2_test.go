package main

import (
	"context"
	"database/sql"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
)

func TestSeedDealCreatesSupportingPipelineAndStage(t *testing.T) {
	db := mustOpenScriptTestDB(t)
	workspaceID, userID := seedScriptTestWorkspaceAndUser(t, db)
	accountID := seedScriptTestAccount(t, db, workspaceID, userID)

	dealID, err := seedDeal(context.Background(), db, authResponse{
		UserID:      userID,
		WorkspaceID: workspaceID,
	}, accountID, "test")
	if err != nil {
		t.Fatalf("seedDeal() error = %v", err)
	}

	var pipelineID, stageID, title string
	if err := db.QueryRow(`
		SELECT pipeline_id, stage_id, title
		FROM deal
		WHERE id = ?
	`, dealID).Scan(&pipelineID, &stageID, &title); err != nil {
		t.Fatalf("query seeded deal: %v", err)
	}
	if pipelineID == "" {
		t.Fatal("expected seeded deal to have pipeline_id")
	}
	if stageID == "" {
		t.Fatal("expected seeded deal to have stage_id")
	}
	if title == "" {
		t.Fatal("expected seeded deal to have title")
	}

	var entityType string
	if err := db.QueryRow(`SELECT entity_type FROM pipeline WHERE id = ?`, pipelineID).Scan(&entityType); err != nil {
		t.Fatalf("query seeded pipeline: %v", err)
	}
	if entityType != "deal" {
		t.Fatalf("expected seeded pipeline entity_type deal, got %q", entityType)
	}

	var stagePipelineID string
	if err := db.QueryRow(`SELECT pipeline_id FROM pipeline_stage WHERE id = ?`, stageID).Scan(&stagePipelineID); err != nil {
		t.Fatalf("query seeded stage: %v", err)
	}
	if stagePipelineID != pipelineID {
		t.Fatalf("expected stage pipeline_id %q, got %q", pipelineID, stagePipelineID)
	}
}

func mustOpenScriptTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sqlite.NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}
	return db
}

func seedScriptTestWorkspaceAndUser(t *testing.T, db *sql.DB) (workspaceID, userID string) {
	t.Helper()

	workspaceID = "ws-seed-test"
	userID = "user-seed-test"

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES (?, ?, ?, datetime('now'), datetime('now'))
	`, workspaceID, "Seed Test Workspace", "seed-test-workspace"); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO user_account (id, workspace_id, email, display_name, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'active', datetime('now'), datetime('now'))
	`, userID, workspaceID, "seed@test.local", "Seed Test User"); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	return workspaceID, userID
}

func seedScriptTestAccount(t *testing.T, db *sql.DB, workspaceID, ownerID string) string {
	t.Helper()

	accountID := "account-seed-test"
	if _, err := db.Exec(`
		INSERT INTO account (id, workspace_id, name, owner_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))
	`, accountID, workspaceID, "Seed Test Account", ownerID); err != nil {
		t.Fatalf("insert account: %v", err)
	}
	return accountID
}
