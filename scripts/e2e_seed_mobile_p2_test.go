package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
)

// TestSeedOutputExposesAuthBlock guards the contract consumed by
// mobile/maestro/seed-and-run.sh. The screenshot runner builds an
// e2e-bootstrap deep link from seed.auth.{token,userId,workspaceId}, so the
// JSON shape must not drift. See
// docs/plans/maestro-screenshot-auth-bypass-plan.md.
func TestSeedOutputExposesAuthBlock(t *testing.T) {
	var out seedOutput
	out.Credentials.Email = "seed@test.local"
	out.Credentials.Password = "seed-password"
	out.Auth.Token = "tok-abc.def.ghi"
	out.Auth.UserID = "user-xyz"
	out.Auth.WorkspaceID = "ws-xyz"
	out.Pipeline.ID = "pipe-xyz"
	out.Stage.ID = "stage-xyz"
	out.Workflow.ID = "workflow-xyz"

	encoded, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("json.Marshal(seedOutput) error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal error = %v", err)
	}

	auth, ok := decoded["auth"].(map[string]any)
	if !ok {
		t.Fatalf("expected seedOutput JSON to contain an 'auth' object, got: %s", string(encoded))
	}
	if auth["token"] != "tok-abc.def.ghi" {
		t.Errorf("auth.token = %v, want tok-abc.def.ghi", auth["token"])
	}
	if auth["userId"] != "user-xyz" {
		t.Errorf("auth.userId = %v, want user-xyz", auth["userId"])
	}
	if auth["workspaceId"] != "ws-xyz" {
		t.Errorf("auth.workspaceId = %v, want ws-xyz", auth["workspaceId"])
	}

	// credentials must still be emitted for non-screenshot consumers of the seeder.
	creds, ok := decoded["credentials"].(map[string]any)
	if !ok {
		t.Fatalf("expected seedOutput JSON to contain a 'credentials' object, got: %s", string(encoded))
	}
	if creds["email"] != "seed@test.local" {
		t.Errorf("credentials.email = %v, want seed@test.local", creds["email"])
	}
	if creds["password"] != "seed-password" {
		t.Errorf("credentials.password = %v, want seed-password", creds["password"])
	}

	pipeline, ok := decoded["pipeline"].(map[string]any)
	if !ok {
		t.Fatalf("expected seedOutput JSON to contain a 'pipeline' object, got: %s", string(encoded))
	}
	if pipeline["id"] != "pipe-xyz" {
		t.Errorf("pipeline.id = %v, want pipe-xyz", pipeline["id"])
	}

	stage, ok := decoded["stage"].(map[string]any)
	if !ok {
		t.Fatalf("expected seedOutput JSON to contain a 'stage' object, got: %s", string(encoded))
	}
	if stage["id"] != "stage-xyz" {
		t.Errorf("stage.id = %v, want stage-xyz", stage["id"])
	}

	workflow, ok := decoded["workflow"].(map[string]any)
	if !ok {
		t.Fatalf("expected seedOutput JSON to contain a 'workflow' object, got: %s", string(encoded))
	}
	if workflow["id"] != "workflow-xyz" {
		t.Errorf("workflow.id = %v, want workflow-xyz", workflow["id"])
	}
}

func TestSeedWorkflowGraphFixtureCreatesRenderableWorkflow(t *testing.T) {
	db := mustOpenScriptTestDB(t)
	workspaceID, userID := seedScriptTestWorkspaceAndUser(t, db)

	workflowID, err := seedWorkflowGraphFixture(context.Background(), db, authResponse{
		UserID:      userID,
		WorkspaceID: workspaceID,
	}, "test")
	if err != nil {
		t.Fatalf("seedWorkflowGraphFixture() error = %v", err)
	}

	var name, dslSource, specSource, status string
	if err := db.QueryRow(`
		SELECT name, dsl_source, spec_source, status
		FROM workflow
		WHERE id = ? AND workspace_id = ?
	`, workflowID, workspaceID).Scan(&name, &dslSource, &specSource, &status); err != nil {
		t.Fatalf("query seeded workflow: %v", err)
	}
	if name == "" {
		t.Fatal("expected seeded workflow to have name")
	}
	if status != "draft" {
		t.Fatalf("status = %q, want draft", status)
	}
	if dslSource == "" || specSource == "" {
		t.Fatalf("expected DSL and spec sources, got dsl=%q spec=%q", dslSource, specSource)
	}
	if dslSource != "WORKFLOW e2e_draft_followup_test\nON case.created\nSET case.status = \"open\"\nNOTIFY owner WITH \"Review new workflow draft\"\n" {
		t.Fatalf("dslSource = %q, want create-draft scaffold", dslSource)
	}
}

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

func TestLoginOrRegisterLogsInExistingUserToRecoverToken(t *testing.T) {
	db := mustOpenScriptTestDB(t)
	workspaceID := "ws-seed-test"
	userID := "user-seed-test"
	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES (?, ?, ?, datetime('now'), datetime('now'))
	`, workspaceID, "Seed Test Workspace", "seed-test-workspace"); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO user_account (id, workspace_id, email, display_name, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'active', datetime('now'), datetime('now'))
	`, userID, workspaceID, testEmail, "Seed Test User"); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/login" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method %q", r.Method)
		}

		var req map[string]string
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req["email"] != testEmail {
			t.Fatalf("email = %q, want %q", req["email"], testEmail)
		}
		if req["password"] != testPassword {
			t.Fatalf("password = %q, want %q", req["password"], testPassword)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(authResponse{
			Token:       "tok-existing-user",
			UserID:      userID,
			WorkspaceID: workspaceID,
		})
	}))
	t.Cleanup(server.Close)

	auth, err := loginOrRegister(context.Background(), server.URL, db)
	if err != nil {
		t.Fatalf("loginOrRegister() error = %v", err)
	}
	if auth.Token != "tok-existing-user" {
		t.Fatalf("Token = %q, want tok-existing-user", auth.Token)
	}
	if auth.UserID != userID {
		t.Fatalf("UserID = %q, want %q", auth.UserID, userID)
	}
	if auth.WorkspaceID != workspaceID {
		t.Fatalf("WorkspaceID = %q, want %q", auth.WorkspaceID, workspaceID)
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
