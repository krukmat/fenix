package agent

import (
	"context"
	"database/sql"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/audit"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	_ "modernc.org/sqlite"
)

// TestMain sets up the test database for PromptService tests
func TestMain(m *testing.M) {
	// Not needed in this case, but can be added if DB state management is needed
	m.Run()
}

func newTestPromptService(t *testing.T, db *sql.DB) *PromptService {
	auditSvc := audit.NewAuditService(db)
	return NewPromptService(db, auditSvc)
}

func TestCreatePromptVersion_Succeeds(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := newTestPromptService(t, db)
	ctx := context.Background()

	input := CreatePromptVersionInput{
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "You are a support agent.",
		Config:            `{"temperature": 0.3, "max_tokens": 2048}`,
	}

	pv, err := svc.CreatePromptVersion(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pv.VersionNumber != 1 {
		t.Errorf("expected version_number=1, got %d", pv.VersionNumber)
	}
	if pv.Status != PromptStatusDraft {
		t.Errorf("expected status=draft, got %s", pv.Status)
	}
	if pv.SystemPrompt != input.SystemPrompt {
		t.Errorf("expected system_prompt=%s, got %s", input.SystemPrompt, pv.SystemPrompt)
	}
}

func TestCreatePromptVersion_AutoIncrementsVersion(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := newTestPromptService(t, db)
	ctx := context.Background()

	baseInput := CreatePromptVersionInput{
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "Version 1",
		Config:            `{"temperature": 0.3}`,
	}

	pv1, err := svc.CreatePromptVersion(ctx, baseInput)
	if err != nil {
		t.Fatalf("create v1: %v", err)
	}

	baseInput.SystemPrompt = "Version 2"
	pv2, err := svc.CreatePromptVersion(ctx, baseInput)
	if err != nil {
		t.Fatalf("create v2: %v", err)
	}

	if pv2.VersionNumber != 2 {
		t.Errorf("expected version_number=2, got %d", pv2.VersionNumber)
	}
	if pv1.VersionNumber != 1 {
		t.Errorf("v1: expected version_number=1, got %d", pv1.VersionNumber)
	}
}

func TestGetActivePrompt_NoActiveVersion_ReturnsError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := newTestPromptService(t, db)
	ctx := context.Background()

	_, err := svc.GetActivePrompt(ctx, "ws_test", "agent_nonexistent")
	if err == nil {
		t.Error("expected error for no active prompt, got nil")
	}
}

func TestGetActivePrompt_ReturnsActive(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := newTestPromptService(t, db)
	ctx := context.Background()

	// Create and promote a prompt version
	input := CreatePromptVersionInput{
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "Active prompt",
		Config:            `{"temperature": 0.3}`,
	}
	pv, err := svc.CreatePromptVersion(ctx, input)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Promote to active
	err = svc.PromotePrompt(ctx, "ws_test", pv.ID)
	if err != nil {
		t.Fatalf("promote: %v", err)
	}

	// Now GetActivePrompt should return it
	active, err := svc.GetActivePrompt(ctx, "ws_test", "agent_support")
	if err != nil {
		t.Fatalf("get active: %v", err)
	}
	if active.ID != pv.ID {
		t.Errorf("expected active.ID=%s, got %s", pv.ID, active.ID)
	}
	if active.Status != PromptStatusActive {
		t.Errorf("expected status=active, got %s", active.Status)
	}
}

func TestPromotePrompt_ActivatesVersion(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := newTestPromptService(t, db)
	ctx := context.Background()

	input := CreatePromptVersionInput{
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "Draft prompt",
		Config:            `{"temperature": 0.3}`,
	}
	pv, err := svc.CreatePromptVersion(ctx, input)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	err = svc.PromotePrompt(ctx, "ws_test", pv.ID)
	if err != nil {
		t.Fatalf("promote: %v", err)
	}

	// Verify it's now active
	updated, err := svc.GetPromptVersionByID(ctx, "ws_test", pv.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if updated.Status != PromptStatusActive {
		t.Errorf("expected status=active, got %s", updated.Status)
	}
}

func TestPromotePrompt_ArchivesPreviousActive(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := newTestPromptService(t, db)
	ctx := context.Background()

	// Create and promote v1
	input := CreatePromptVersionInput{
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "Version 1",
		Config:            `{"temperature": 0.3}`,
	}
	pv1, err := svc.CreatePromptVersion(ctx, input)
	if err != nil {
		t.Fatalf("create v1: %v", err)
	}
	err = svc.PromotePrompt(ctx, "ws_test", pv1.ID)
	if err != nil {
		t.Fatalf("promote v1: %v", err)
	}

	// Create and promote v2
	input.SystemPrompt = "Version 2"
	pv2, err := svc.CreatePromptVersion(ctx, input)
	if err != nil {
		t.Fatalf("create v2: %v", err)
	}
	err = svc.PromotePrompt(ctx, "ws_test", pv2.ID)
	if err != nil {
		t.Fatalf("promote v2: %v", err)
	}

	// v1 should now be archived
	v1, err := svc.GetPromptVersionByID(ctx, "ws_test", pv1.ID)
	if err != nil {
		t.Fatalf("get v1: %v", err)
	}
	if v1.Status != PromptStatusArchived {
		t.Errorf("expected v1 status=archived, got %s", v1.Status)
	}

	// v2 should be active
	v2, err := svc.GetPromptVersionByID(ctx, "ws_test", pv2.ID)
	if err != nil {
		t.Fatalf("get v2: %v", err)
	}
	if v2.Status != PromptStatusActive {
		t.Errorf("expected v2 status=active, got %s", v2.Status)
	}
}

func TestPromotePrompt_WrongStatus_ReturnsError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := newTestPromptService(t, db)
	ctx := context.Background()

	// Create, promote, then try to promote an archived version
	input := CreatePromptVersionInput{
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "Test",
		Config:            `{}`,
	}
	pv, err := svc.CreatePromptVersion(ctx, input)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	err = svc.PromotePrompt(ctx, "ws_test", pv.ID)
	if err != nil {
		t.Fatalf("promote: %v", err)
	}

	// Create a second one and promote it to archive the first
	input.SystemPrompt = "Test 2"
	pv2, err := svc.CreatePromptVersion(ctx, input)
	if err != nil {
		t.Fatalf("create v2: %v", err)
	}
	err = svc.PromotePrompt(ctx, "ws_test", pv2.ID)
	if err != nil {
		t.Fatalf("promote v2: %v", err)
	}

	// Now try to promote the archived one — should error
	err = svc.PromotePrompt(ctx, "ws_test", pv.ID)
	if err == nil {
		t.Error("expected error promoting archived prompt, got nil")
	}
}

func TestRollbackPrompt_ReactivatesPrevious(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := newTestPromptService(t, db)
	ctx := context.Background()

	// Create and promote v1
	input := CreatePromptVersionInput{
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "V1",
		Config:            `{}`,
	}
	pv1, _ := svc.CreatePromptVersion(ctx, input)
	svc.PromotePrompt(ctx, "ws_test", pv1.ID)

	// Create and promote v2 (archives v1)
	input.SystemPrompt = "V2"
	pv2, _ := svc.CreatePromptVersion(ctx, input)
	svc.PromotePrompt(ctx, "ws_test", pv2.ID)

	// Rollback — v1 should be active again, v2 archived
	err := svc.RollbackPrompt(ctx, "ws_test", "agent_support")
	if err != nil {
		t.Fatalf("rollback: %v", err)
	}

	v1, _ := svc.GetPromptVersionByID(ctx, "ws_test", pv1.ID)
	if v1.Status != PromptStatusActive {
		t.Errorf("expected v1 active after rollback, got %s", v1.Status)
	}

	v2, _ := svc.GetPromptVersionByID(ctx, "ws_test", pv2.ID)
	if v2.Status != PromptStatusArchived {
		t.Errorf("expected v2 archived after rollback, got %s", v2.Status)
	}
}

func TestRollbackPrompt_NoArchived_ReturnsError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := newTestPromptService(t, db)
	ctx := context.Background()

	// Try to rollback when there's no archived version
	err := svc.RollbackPrompt(ctx, "ws_test", "agent_nonexistent")
	if err == nil {
		t.Error("expected error rolling back with no archived prompt, got nil")
	}
}

// Helper: setupTestDB creates an in-memory SQLite DB for testing
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}

	// Apply all migrations
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db
}
