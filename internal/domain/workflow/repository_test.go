package workflow

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	isqlite "github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	_ "modernc.org/sqlite"
)

func TestRepository_CreateAndGetByID(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)

	created, err := repo.Create(context.Background(), CreateInput{
		ID:          "wf-1",
		WorkspaceID: "ws_test",
		Name:        "qualify_lead",
		DSLSource:   "ON lead.created",
		Version:     1,
		Status:      StatusDraft,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Status != StatusDraft {
		t.Fatalf("status = %s, want %s", created.Status, StatusDraft)
	}

	got, err := repo.GetByID(context.Background(), "ws_test", created.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Name != "qualify_lead" {
		t.Fatalf("name = %s, want qualify_lead", got.Name)
	}
}

func TestRepository_GetByID_NotFound(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)

	_, err := repo.GetByID(context.Background(), "ws_test", "missing")
	if !errors.Is(err, ErrWorkflowNotFound) {
		t.Fatalf("expected ErrWorkflowNotFound, got %v", err)
	}
}

func TestRepository_GetByNameAndVersion(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)

	_, err := repo.Create(context.Background(), CreateInput{
		ID:          "wf-version",
		WorkspaceID: "ws_test",
		Name:        "qualify_lead",
		DSLSource:   "ON lead.created",
		Version:     2,
		Status:      StatusTesting,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := repo.GetByNameAndVersion(context.Background(), "ws_test", "qualify_lead", 2)
	if err != nil {
		t.Fatalf("GetByNameAndVersion() error = %v", err)
	}
	if got.Version != 2 {
		t.Fatalf("version = %d, want 2", got.Version)
	}
}

func TestRepository_GetActiveByName(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)

	_, err := repo.Create(context.Background(), CreateInput{
		ID:          "wf-active",
		WorkspaceID: "ws_test",
		Name:        "triage_case",
		DSLSource:   "ON case.created",
		Version:     1,
		Status:      StatusActive,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := repo.GetActiveByName(context.Background(), "ws_test", "triage_case")
	if err != nil {
		t.Fatalf("GetActiveByName() error = %v", err)
	}
	if got.Status != StatusActive {
		t.Fatalf("status = %s, want %s", got.Status, StatusActive)
	}
}

func TestRepository_GetActiveByAgentDefinition(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)

	agentDefinitionID := "agent-dsl-1"
	_, err := repo.Create(context.Background(), CreateInput{
		ID:                "wf-active-def",
		WorkspaceID:       "ws_test",
		AgentDefinitionID: &agentDefinitionID,
		Name:              "triage_case",
		DSLSource:         "WORKFLOW triage_case\nON case.created\nSET case.status = \"open\"",
		Version:           1,
		Status:            StatusActive,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := repo.GetActiveByAgentDefinition(context.Background(), "ws_test", agentDefinitionID)
	if err != nil {
		t.Fatalf("GetActiveByAgentDefinition() error = %v", err)
	}
	if got.ID != "wf-active-def" {
		t.Fatalf("id = %s, want wf-active-def", got.ID)
	}
}

func TestRepository_ListByWorkspaceAndStatus(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)

	inputs := []CreateInput{
		{ID: "wf-1", WorkspaceID: "ws_test", Name: "qualify_lead", DSLSource: "ON lead.created", Version: 1, Status: StatusDraft},
		{ID: "wf-2", WorkspaceID: "ws_test", Name: "triage_case", DSLSource: "ON case.created", Version: 1, Status: StatusActive},
		{ID: "wf-3", WorkspaceID: "ws_test", Name: "triage_case", DSLSource: "ON case.updated", Version: 2, Status: StatusArchived},
	}
	for _, input := range inputs {
		if _, err := repo.Create(context.Background(), input); err != nil {
			t.Fatalf("Create(%s) error = %v", input.ID, err)
		}
	}

	all, err := repo.ListByWorkspace(context.Background(), "ws_test")
	if err != nil {
		t.Fatalf("ListByWorkspace() error = %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("len(all) = %d, want 3", len(all))
	}

	active, err := repo.ListByStatus(context.Background(), "ws_test", StatusActive)
	if err != nil {
		t.Fatalf("ListByStatus() error = %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("len(active) = %d, want 1", len(active))
	}
}

func TestRepository_ListVersionsByName(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)

	for _, input := range []CreateInput{
		{ID: "wf-v1", WorkspaceID: "ws_test", Name: "qualify_lead", DSLSource: "v1", Version: 1, Status: StatusArchived},
		{ID: "wf-v2", WorkspaceID: "ws_test", Name: "qualify_lead", DSLSource: "v2", Version: 2, Status: StatusActive},
	} {
		if _, err := repo.Create(context.Background(), input); err != nil {
			t.Fatalf("Create(%s) error = %v", input.ID, err)
		}
	}

	versions, err := repo.ListVersionsByName(context.Background(), "ws_test", "qualify_lead")
	if err != nil {
		t.Fatalf("ListVersionsByName() error = %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("len(versions) = %d, want 2", len(versions))
	}
	if versions[0].Version != 2 {
		t.Fatalf("first version = %d, want 2", versions[0].Version)
	}
}

func TestRepository_Update(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)

	_, err := repo.Create(context.Background(), CreateInput{
		ID:          "wf-update",
		WorkspaceID: "ws_test",
		Name:        "qualify_lead",
		DSLSource:   "ON lead.created",
		Version:     1,
		Status:      StatusDraft,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	desc := "updated workflow"
	spec := "BEHAVIOR define_workflow"
	updated, err := repo.Update(context.Background(), "ws_test", "wf-update", UpdateInput{
		Description: &desc,
		DSLSource:   "ON lead.updated",
		SpecSource:  &spec,
		Status:      StatusTesting,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Status != StatusTesting {
		t.Fatalf("status = %s, want %s", updated.Status, StatusTesting)
	}
	if updated.Description == nil || *updated.Description != desc {
		t.Fatalf("description = %+v, want %s", updated.Description, desc)
	}
}

func TestRepository_Delete(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)

	_, err := repo.Create(context.Background(), CreateInput{
		ID:          "wf-delete",
		WorkspaceID: "ws_test",
		Name:        "qualify_lead",
		DSLSource:   "ON lead.created",
		Version:     1,
		Status:      StatusDraft,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := repo.Delete(context.Background(), "ws_test", "wf-delete"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = repo.GetByID(context.Background(), "ws_test", "wf-delete")
	if !errors.Is(err, ErrWorkflowNotFound) {
		t.Fatalf("expected ErrWorkflowNotFound after delete, got %v", err)
	}
}

func TestRepository_GetByNameAndVersion_NotFound(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)

	_, err := repo.GetByNameAndVersion(context.Background(), "ws_test", "missing", 1)
	if !errors.Is(err, ErrWorkflowNotFound) {
		t.Fatalf("expected ErrWorkflowNotFound, got %v", err)
	}
}

func TestRepository_Update_NotFound(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)

	_, err := repo.Update(context.Background(), "ws_test", "missing-id", UpdateInput{DSLSource: "ON lead.updated", Status: StatusDraft})
	if !errors.Is(err, ErrWorkflowNotFound) {
		t.Fatalf("expected ErrWorkflowNotFound, got %v", err)
	}
}

func TestRepository_Delete_NotFound(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)

	err := repo.Delete(context.Background(), "ws_test", "missing-id")
	if !errors.Is(err, ErrWorkflowNotFound) {
		t.Fatalf("expected ErrWorkflowNotFound, got %v", err)
	}
}

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	if err = isqlite.MigrateUp(db); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	if _, err = db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws_test', 'Workflow Test', 'workflow-test', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
