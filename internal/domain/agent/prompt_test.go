package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/audit"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	_ "modernc.org/sqlite"
)

func TestMain(m *testing.M) {
	m.Run()
}

func newTestPromptService(t *testing.T, db *sql.DB) *PromptService {
	t.Helper()
	return NewPromptService(db, audit.NewAuditService(db))
}

func TestCreatePromptVersion_Succeeds(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := newTestPromptService(t, db)
	pv, err := svc.CreatePromptVersion(context.Background(), CreatePromptVersionInput{
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "You are a support agent.",
		Config:            `{"temperature": 0.3, "max_tokens": 2048}`,
	})
	if err != nil {
		t.Fatalf("CreatePromptVersion: %v", err)
	}
	if pv.VersionNumber != 1 {
		t.Fatalf("expected version 1, got %d", pv.VersionNumber)
	}
	if pv.Status != PromptStatusDraft {
		t.Fatalf("expected draft, got %s", pv.Status)
	}
	assertAuditActionCount(t, db, "ws_test", "prompt.created", 1)
}

func TestCreatePromptVersion_AutoIncrementsVersion(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := newTestPromptService(t, db)
	ctx := context.Background()
	_, err := svc.CreatePromptVersion(ctx, CreatePromptVersionInput{
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "Version 1",
		Config:            `{}`,
	})
	if err != nil {
		t.Fatalf("create v1: %v", err)
	}
	pv2, err := svc.CreatePromptVersion(ctx, CreatePromptVersionInput{
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "Version 2",
		Config:            `{}`,
	})
	if err != nil {
		t.Fatalf("create v2: %v", err)
	}
	if pv2.VersionNumber != 2 {
		t.Fatalf("expected version 2, got %d", pv2.VersionNumber)
	}
}

func TestCreatePromptVersion_UsesProvidedCreatedByWhenContextMissing(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := newTestPromptService(t, db)
	createdBy := "user_override"
	pv, err := svc.CreatePromptVersion(context.Background(), CreatePromptVersionInput{
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "Created by fallback",
		Config:            `{}`,
		CreatedBy:         &createdBy,
	})
	if err != nil {
		t.Fatalf("CreatePromptVersion: %v", err)
	}
	if pv.CreatedBy == nil || *pv.CreatedBy != createdBy {
		t.Fatalf("expected created_by %s, got %+v", createdBy, pv.CreatedBy)
	}
}

func TestPromotePrompt_RequiresPassingEval(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := newTestPromptService(t, db)
	ctx := context.Background()
	pv, err := svc.CreatePromptVersion(ctx, CreatePromptVersionInput{
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "Needs eval",
		Config:            `{}`,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	err = svc.PromotePrompt(ctx, "ws_test", pv.ID)
	if !errors.Is(err, ErrPromptPromotionEvalMissing) {
		t.Fatalf("expected ErrPromptPromotionEvalMissing, got %v", err)
	}
	assertAuditActionCount(t, db, "ws_test", "prompt.promote_blocked", 1)
	assertAuditMetadataContains(t, db, "ws_test", "prompt.promote_blocked", "agent_id", "agent_support")
}

func TestPromotePrompt_BlocksFailedEval(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := newTestPromptService(t, db)
	ctx := context.Background()
	pv, err := svc.CreatePromptVersion(ctx, CreatePromptVersionInput{
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "Failed eval",
		Config:            `{}`,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	insertPromptEvalRun(t, db, "ws_test", pv.ID, "failed")

	err = svc.PromotePrompt(ctx, "ws_test", pv.ID)
	if !errors.Is(err, ErrPromptPromotionEvalFailed) {
		t.Fatalf("expected ErrPromptPromotionEvalFailed, got %v", err)
	}
}

func TestPromotePrompt_ActivatesVersion(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	insertAgentDefinition(t, db, "ws_test", "agent_support")

	svc := newTestPromptService(t, db)
	ctx := context.Background()
	pv, err := svc.CreatePromptVersion(ctx, CreatePromptVersionInput{
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "Draft prompt",
		Config:            `{}`,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	insertPromptEvalRun(t, db, "ws_test", pv.ID, evalStatusPassed)

	err = svc.PromotePrompt(ctx, "ws_test", pv.ID)
	if err != nil {
		t.Fatalf("PromotePrompt: %v", err)
	}

	got, err := svc.GetPromptVersionByID(ctx, "ws_test", pv.ID)
	if err != nil {
		t.Fatalf("GetPromptVersionByID: %v", err)
	}
	if got.Status != PromptStatusActive {
		t.Fatalf("expected active, got %s", got.Status)
	}
	if activeID := getAgentActivePromptVersionID(t, db, "ws_test", "agent_support"); activeID != pv.ID {
		t.Fatalf("expected active_prompt_version_id %s, got %s", pv.ID, activeID)
	}
	assertAuditActionCount(t, db, "ws_test", "prompt.activated", 1)
	assertAuditMetadataContains(t, db, "ws_test", "prompt.activated", "agent_id", "agent_support")
}

func TestPromotePrompt_ArchivesPreviousActive(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	insertAgentDefinition(t, db, "ws_test", "agent_support")

	svc := newTestPromptService(t, db)
	ctx := context.Background()
	pv1, _ := svc.CreatePromptVersion(ctx, CreatePromptVersionInput{
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "Version 1",
		Config:            `{}`,
	})
	insertPromptEvalRun(t, db, "ws_test", pv1.ID, evalStatusPassed)
	if err := svc.PromotePrompt(ctx, "ws_test", pv1.ID); err != nil {
		t.Fatalf("promote v1: %v", err)
	}

	pv2, _ := svc.CreatePromptVersion(ctx, CreatePromptVersionInput{
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "Version 2",
		Config:            `{}`,
	})
	insertPromptEvalRun(t, db, "ws_test", pv2.ID, evalStatusPassed)
	if err := svc.PromotePrompt(ctx, "ws_test", pv2.ID); err != nil {
		t.Fatalf("promote v2: %v", err)
	}

	gotV1, _ := svc.GetPromptVersionByID(ctx, "ws_test", pv1.ID)
	gotV2, _ := svc.GetPromptVersionByID(ctx, "ws_test", pv2.ID)
	if gotV1.Status != PromptStatusArchived {
		t.Fatalf("expected v1 archived, got %s", gotV1.Status)
	}
	if gotV2.Status != PromptStatusActive {
		t.Fatalf("expected v2 active, got %s", gotV2.Status)
	}
}

func TestPromotePrompt_RejectsArchivedVersion(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := newTestPromptService(t, db)
	ctx := context.Background()
	pv, err := svc.CreatePromptVersion(ctx, CreatePromptVersionInput{
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "Archived prompt",
		Config:            `{}`,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	_, err = db.Exec(`UPDATE prompt_version SET status = ? WHERE id = ?`, PromptStatusArchived, pv.ID)
	if err != nil {
		t.Fatalf("archive prompt: %v", err)
	}
	insertPromptEvalRun(t, db, "ws_test", pv.ID, evalStatusPassed)

	err = svc.PromotePrompt(ctx, "ws_test", pv.ID)
	if !errors.Is(err, ErrPromptVersionArchived) {
		t.Fatalf("expected ErrPromptVersionArchived, got %v", err)
	}
}

func TestRollbackPrompt_ReactivatesArchivedVersionByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	insertAgentDefinition(t, db, "ws_test", "agent_support")

	svc := newTestPromptService(t, db)
	ctx := context.Background()
	pv1, _ := svc.CreatePromptVersion(ctx, CreatePromptVersionInput{
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "V1",
		Config:            `{}`,
	})
	insertPromptEvalRun(t, db, "ws_test", pv1.ID, evalStatusPassed)
	_ = svc.PromotePrompt(ctx, "ws_test", pv1.ID)

	pv2, _ := svc.CreatePromptVersion(ctx, CreatePromptVersionInput{
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "V2",
		Config:            `{}`,
	})
	insertPromptEvalRun(t, db, "ws_test", pv2.ID, evalStatusPassed)
	_ = svc.PromotePrompt(ctx, "ws_test", pv2.ID)

	if err := svc.RollbackPrompt(ctx, "ws_test", pv1.ID); err != nil {
		t.Fatalf("RollbackPrompt: %v", err)
	}

	gotV1, _ := svc.GetPromptVersionByID(ctx, "ws_test", pv1.ID)
	gotV2, _ := svc.GetPromptVersionByID(ctx, "ws_test", pv2.ID)
	if gotV1.Status != PromptStatusActive {
		t.Fatalf("expected v1 active, got %s", gotV1.Status)
	}
	if gotV2.Status != PromptStatusArchived {
		t.Fatalf("expected v2 archived, got %s", gotV2.Status)
	}
	if activeID := getAgentActivePromptVersionID(t, db, "ws_test", "agent_support"); activeID != pv1.ID {
		t.Fatalf("expected active_prompt_version_id %s, got %s", pv1.ID, activeID)
	}
	assertAuditActionCount(t, db, "ws_test", "prompt.rollback", 1)
	assertAuditMetadataContains(t, db, "ws_test", "prompt.rollback", "agent_id", "agent_support")
}

func TestRollbackPrompt_RejectsNonArchivedVersion(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := newTestPromptService(t, db)
	ctx := context.Background()
	pv, err := svc.CreatePromptVersion(ctx, CreatePromptVersionInput{
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "Draft only",
		Config:            `{}`,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	err = svc.RollbackPrompt(ctx, "ws_test", pv.ID)
	if !errors.Is(err, ErrPromptRollbackInvalid) {
		t.Fatalf("expected ErrPromptRollbackInvalid, got %v", err)
	}
}

func TestRollbackPrompt_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := newTestPromptService(t, db)
	err := svc.RollbackPrompt(context.Background(), "ws_test", "missing-id")
	if !errors.Is(err, ErrPromptVersionNotFound) {
		t.Fatalf("expected ErrPromptVersionNotFound, got %v", err)
	}
}

func TestPromptExperiment_StartAndStop(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := newTestPromptService(t, db)
	ctx := context.Background()
	control, _ := svc.CreatePromptVersion(ctx, CreatePromptVersionInput{
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "Control",
		Config:            `{}`,
	})
	candidate, _ := svc.CreatePromptVersion(ctx, CreatePromptVersionInput{
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "Candidate",
		Config:            `{}`,
	})

	experiment, err := svc.StartPromptExperiment(ctx, StartPromptExperimentInput{
		WorkspaceID:              "ws_test",
		ControlPromptVersionID:   control.ID,
		CandidatePromptVersionID: candidate.ID,
		ControlTrafficPercent:    50,
		CandidateTrafficPercent:  50,
	})
	if err != nil {
		t.Fatalf("StartPromptExperiment: %v", err)
	}
	if experiment.Status != PromptExperimentStatusRunning {
		t.Fatalf("expected running, got %s", experiment.Status)
	}

	stopped, err := svc.StopPromptExperiment(ctx, StopPromptExperimentInput{
		WorkspaceID:           "ws_test",
		ExperimentID:          experiment.ID,
		WinnerPromptVersionID: &control.ID,
	})
	if err != nil {
		t.Fatalf("StopPromptExperiment: %v", err)
	}
	if stopped.Status != PromptExperimentStatusCompleted {
		t.Fatalf("expected completed, got %s", stopped.Status)
	}
}

func TestPromptExperiment_RejectsSecondRunningExperiment(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := newTestPromptService(t, db)
	ctx := context.Background()
	control, _ := svc.CreatePromptVersion(ctx, CreatePromptVersionInput{
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "Control",
		Config:            `{}`,
	})
	candidate1, _ := svc.CreatePromptVersion(ctx, CreatePromptVersionInput{
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "Candidate 1",
		Config:            `{}`,
	})
	candidate2, _ := svc.CreatePromptVersion(ctx, CreatePromptVersionInput{
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "Candidate 2",
		Config:            `{}`,
	})

	_, err := svc.StartPromptExperiment(ctx, StartPromptExperimentInput{
		WorkspaceID:              "ws_test",
		ControlPromptVersionID:   control.ID,
		CandidatePromptVersionID: candidate1.ID,
		ControlTrafficPercent:    50,
		CandidateTrafficPercent:  50,
	})
	if err != nil {
		t.Fatalf("first StartPromptExperiment: %v", err)
	}

	_, err = svc.StartPromptExperiment(ctx, StartPromptExperimentInput{
		WorkspaceID:              "ws_test",
		ControlPromptVersionID:   control.ID,
		CandidatePromptVersionID: candidate2.ID,
		ControlTrafficPercent:    50,
		CandidateTrafficPercent:  50,
	})
	if !errors.Is(err, ErrPromptExperimentAlreadyRunning) {
		t.Fatalf("expected ErrPromptExperimentAlreadyRunning, got %v", err)
	}
}

func TestListPromptVersions_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := newTestPromptService(t, db)
	ctx := context.Background()
	_, err := svc.CreatePromptVersion(ctx, CreatePromptVersionInput{
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "List test prompt",
		Config:            `{}`,
	})
	if err != nil {
		t.Fatalf("CreatePromptVersion: %v", err)
	}

	versions, err := svc.ListPromptVersions(ctx, "ws_test", "agent_support")
	if err != nil {
		t.Fatalf("ListPromptVersions: %v", err)
	}
	if len(versions) != 1 {
		t.Fatalf("expected 1 version, got %d", len(versions))
	}
}

func TestGetPromptVersionByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := newTestPromptService(t, db)
	_, err := svc.GetPromptVersionByID(context.Background(), "ws_test", "nonexistent-id")
	if !errors.Is(err, ErrPromptVersionNotFound) {
		t.Fatalf("expected ErrPromptVersionNotFound, got %v", err)
	}
}

func TestPromptExperiment_RejectsInvalidSplit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := newTestPromptService(t, db)
	ctx := context.Background()
	control, _ := svc.CreatePromptVersion(ctx, CreatePromptVersionInput{
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "Control",
		Config:            `{}`,
	})
	candidate, _ := svc.CreatePromptVersion(ctx, CreatePromptVersionInput{
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "Candidate",
		Config:            `{}`,
	})

	_, err := svc.StartPromptExperiment(ctx, StartPromptExperimentInput{
		WorkspaceID:              "ws_test",
		ControlPromptVersionID:   control.ID,
		CandidatePromptVersionID: candidate.ID,
		ControlTrafficPercent:    70,
		CandidateTrafficPercent:  20,
	})
	if !errors.Is(err, ErrPromptExperimentInvalidSplit) {
		t.Fatalf("expected ErrPromptExperimentInvalidSplit, got %v", err)
	}
}

func TestPromptExperiment_RejectsSameVersion(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := newTestPromptService(t, db)
	ctx := context.Background()
	control, _ := svc.CreatePromptVersion(ctx, CreatePromptVersionInput{
		WorkspaceID:       "ws_test",
		AgentDefinitionID: "agent_support",
		SystemPrompt:      "Control",
		Config:            `{}`,
	})

	_, err := svc.StartPromptExperiment(ctx, StartPromptExperimentInput{
		WorkspaceID:              "ws_test",
		ControlPromptVersionID:   control.ID,
		CandidatePromptVersionID: control.ID,
		ControlTrafficPercent:    50,
		CandidateTrafficPercent:  50,
	})
	if !errors.Is(err, ErrPromptExperimentSameVersion) {
		t.Fatalf("expected ErrPromptExperimentSameVersion, got %v", err)
	}
}

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	if err = sqlite.MigrateUp(db); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func insertPromptEvalRun(t *testing.T, db *sql.DB, workspaceID, promptVersionID, status string) {
	t.Helper()

	suiteID := "suite_" + promptVersionID + "_" + status
	_, err := db.Exec(`
		INSERT INTO eval_suite (id, workspace_id, name, domain, test_cases, thresholds, created_at, updated_at)
		VALUES (?, ?, ?, 'support', '[]', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, suiteID, workspaceID, suiteID)
	if err != nil {
		t.Fatalf("insert eval suite: %v", err)
	}

	now := time.Now()
	_, err = db.Exec(`
		INSERT INTO eval_run (
			id, workspace_id, eval_suite_id, prompt_version_id, status, scores, details,
			started_at, completed_at, created_at
		) VALUES (?, ?, ?, ?, ?, '{}', '[]', ?, ?, ?)
	`, "run_"+promptVersionID+"_"+status, workspaceID, suiteID, promptVersionID, status, now, now, now)
	if err != nil {
		t.Fatalf("insert eval run: %v", err)
	}
}

func insertAgentDefinition(t *testing.T, db *sql.DB, workspaceID, agentID string) {
	t.Helper()

	_, err := db.Exec(`
		INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		VALUES (?, ?, ?, ?, 'active')
	`, agentID, workspaceID, agentID, "support")
	if err != nil {
		t.Fatalf("insert agent_definition: %v", err)
	}
}

func getAgentActivePromptVersionID(t *testing.T, db *sql.DB, workspaceID, agentID string) string {
	t.Helper()

	var activeID sql.NullString
	err := db.QueryRow(`
		SELECT active_prompt_version_id
		FROM agent_definition
		WHERE workspace_id = ? AND id = ?
	`, workspaceID, agentID).Scan(&activeID)
	if err != nil {
		t.Fatalf("query active prompt version: %v", err)
	}
	if !activeID.Valid {
		return ""
	}
	return activeID.String
}

func assertAuditActionCount(t *testing.T, db *sql.DB, workspaceID, action string, expected int) {
	t.Helper()

	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM audit_event WHERE workspace_id = ? AND action = ?`, workspaceID, action).Scan(&count)
	if err != nil {
		t.Fatalf("count audit events: %v", err)
	}
	if count != expected {
		t.Fatalf("expected %d audit events for %s, got %d", expected, action, count)
	}
}

func assertAuditMetadataContains(t *testing.T, db *sql.DB, workspaceID, action, key, expected string) {
	t.Helper()

	var details string
	err := db.QueryRow(`
		SELECT details
		FROM audit_event
		WHERE workspace_id = ? AND action = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, workspaceID, action).Scan(&details)
	if err != nil {
		t.Fatalf("select audit details: %v", err)
	}

	var payload map[string]any
	if err = json.Unmarshal([]byte(details), &payload); err != nil {
		t.Fatalf("unmarshal audit details: %v", err)
	}
	metadata, ok := payload["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected metadata object in audit details: %#v", payload)
	}
	if metadata[key] != expected {
		t.Fatalf("expected metadata[%s]=%s, got %#v", key, expected, metadata[key])
	}
}
