package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	schedulerdomain "github.com/matiasleandrokruk/fenix/internal/domain/scheduler"
)

func TestService_Create_SetsDraftV1(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	svc := NewService(db)

	got, err := svc.Create(context.Background(), CreateWorkflowInput{
		WorkspaceID: "ws_test",
		Name:        "qualify_lead",
		DSLSource:   "ON lead.created",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if got.Version != 1 {
		t.Fatalf("version = %d, want 1", got.Version)
	}
	if got.Status != StatusDraft {
		t.Fatalf("status = %s, want %s", got.Status, StatusDraft)
	}
}

func TestService_Create_RejectsMissingFields(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	svc := NewService(db)

	_, err := svc.Create(context.Background(), CreateWorkflowInput{
		WorkspaceID: "ws_test",
		Name:        "",
		DSLSource:   "",
	})
	if !errors.Is(err, ErrInvalidWorkflowInput) {
		t.Fatalf("expected ErrInvalidWorkflowInput, got %v", err)
	}
}

func TestService_Create_RejectsWhitespaceDSL(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	svc := NewService(db)

	_, err := svc.Create(context.Background(), CreateWorkflowInput{
		WorkspaceID: "ws_test",
		Name:        "qualify_lead",
		DSLSource:   "   ",
	})
	if !errors.Is(err, ErrInvalidWorkflowInput) {
		t.Fatalf("expected ErrInvalidWorkflowInput, got %v", err)
	}
}

func TestService_Create_RejectsOversizedSources(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	svc := NewService(db)
	oversized := strings.Repeat("a", maxSourceSizeBytes+1)

	_, err := svc.Create(context.Background(), CreateWorkflowInput{
		WorkspaceID: "ws_test",
		Name:        "qualify_lead",
		DSLSource:   oversized,
	})
	if !errors.Is(err, ErrInvalidWorkflowInput) {
		t.Fatalf("expected ErrInvalidWorkflowInput for dsl_source, got %v", err)
	}

	_, err = svc.Create(context.Background(), CreateWorkflowInput{
		WorkspaceID: "ws_test",
		Name:        "qualify_lead",
		DSLSource:   "ON lead.created",
		SpecSource:  oversized,
	})
	if !errors.Is(err, ErrInvalidWorkflowInput) {
		t.Fatalf("expected ErrInvalidWorkflowInput for spec_source, got %v", err)
	}
}

func TestService_Create_RejectsDuplicateNameVersion(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	svc := NewService(db)

	_, err := svc.Create(context.Background(), CreateWorkflowInput{
		WorkspaceID: "ws_test",
		Name:        "qualify_lead",
		DSLSource:   "ON lead.created",
	})
	if err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	_, err = svc.Create(context.Background(), CreateWorkflowInput{
		WorkspaceID: "ws_test",
		Name:        "qualify_lead",
		DSLSource:   "ON lead.created",
	})
	if !errors.Is(err, ErrWorkflowNameConflict) {
		t.Fatalf("expected ErrWorkflowNameConflict, got %v", err)
	}
}

func TestService_GetAndList_ScopedByWorkspace(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	svc := NewService(db)

	created, err := svc.Create(context.Background(), CreateWorkflowInput{
		WorkspaceID: "ws_test",
		Name:        "qualify_lead",
		DSLSource:   "ON lead.created",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := svc.Get(context.Background(), "ws_test", created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("id = %s, want %s", got.ID, created.ID)
	}

	list, err := svc.List(context.Background(), "ws_test", ListWorkflowsInput{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}
}

func TestService_GetActiveByAgentDefinition(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	svc := NewService(db)
	agentDefinitionID := "agent-dsl-1"

	created, err := svc.Create(context.Background(), CreateWorkflowInput{
		WorkspaceID:       "ws_test",
		AgentDefinitionID: &agentDefinitionID,
		Name:              "triage_case",
		DSLSource:         "WORKFLOW triage_case\nON case.created\nSET case.status = \"open\"",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := svc.MarkTesting(context.Background(), "ws_test", created.ID); err != nil {
		t.Fatalf("MarkTesting() error = %v", err)
	}
	if _, err := svc.MarkActive(context.Background(), "ws_test", created.ID); err != nil {
		t.Fatalf("MarkActive() error = %v", err)
	}

	got, err := svc.GetActiveByAgentDefinition(context.Background(), "ws_test", agentDefinitionID)
	if err != nil {
		t.Fatalf("GetActiveByAgentDefinition() error = %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("id = %s, want %s", got.ID, created.ID)
	}
}

func TestService_List_FiltersByStatusAndName(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewServiceWithRepository(repo)

	for _, input := range []CreateInput{
		{ID: "wf-1", WorkspaceID: "ws_test", Name: "qualify_lead", DSLSource: "ON lead.created", Version: 1, Status: StatusDraft},
		{ID: "wf-2", WorkspaceID: "ws_test", Name: "triage_case", DSLSource: "ON case.created", Version: 1, Status: StatusActive},
		{ID: "wf-3", WorkspaceID: "ws_test", Name: "triage_case", DSLSource: "ON case.updated", Version: 2, Status: StatusArchived},
	} {
		if _, err := repo.Create(context.Background(), input); err != nil {
			t.Fatalf("repo.Create(%s) error = %v", input.ID, err)
		}
	}

	status := StatusActive
	active, err := svc.List(context.Background(), "ws_test", ListWorkflowsInput{Status: &status})
	if err != nil {
		t.Fatalf("List(status) error = %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("len(active) = %d, want 1", len(active))
	}

	named, err := svc.List(context.Background(), "ws_test", ListWorkflowsInput{Name: "triage_case"})
	if err != nil {
		t.Fatalf("List(name) error = %v", err)
	}
	if len(named) != 2 {
		t.Fatalf("len(named) = %d, want 2", len(named))
	}
}

func TestService_Update_OnlyDraft(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	svc := NewService(db)

	created, err := svc.Create(context.Background(), CreateWorkflowInput{
		WorkspaceID: "ws_test",
		Name:        "qualify_lead",
		DSLSource:   "ON lead.created",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updated, err := svc.Update(context.Background(), "ws_test", created.ID, UpdateWorkflowInput{
		Description: "updated",
		DSLSource:   "ON lead.updated",
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Description == nil || *updated.Description != "updated" {
		t.Fatalf("description = %+v, want updated", updated.Description)
	}

	_, err = NewRepository(db).Update(context.Background(), "ws_test", created.ID, UpdateInput{
		Description: updated.Description,
		DSLSource:   updated.DSLSource,
		SpecSource:  updated.SpecSource,
		Status:      StatusActive,
	})
	if err != nil {
		t.Fatalf("repo.Update() to active error = %v", err)
	}

	_, err = svc.Update(context.Background(), "ws_test", created.ID, UpdateWorkflowInput{
		Description: "blocked",
		DSLSource:   "ON lead.closed",
	})
	if !errors.Is(err, ErrWorkflowNotEditable) {
		t.Fatalf("expected ErrWorkflowNotEditable, got %v", err)
	}
}

func TestService_Update_ValidatesInput(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	svc := NewService(db)

	created, err := svc.Create(context.Background(), CreateWorkflowInput{
		WorkspaceID: "ws_test",
		Name:        "qualify_lead",
		DSLSource:   "ON lead.created",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	_, err = svc.Update(context.Background(), "ws_test", created.ID, UpdateWorkflowInput{
		DSLSource: " ",
	})
	if !errors.Is(err, ErrInvalidWorkflowInput) {
		t.Fatalf("expected ErrInvalidWorkflowInput, got %v", err)
	}
}

func TestService_SetStatus_AllowsValidTransitions(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	svc := NewService(db)

	created, err := svc.Create(context.Background(), CreateWorkflowInput{
		WorkspaceID: "ws_test",
		Name:        "qualify_lead",
		DSLSource:   "ON lead.created",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	testingWF, err := svc.MarkTesting(context.Background(), "ws_test", created.ID)
	if err != nil {
		t.Fatalf("MarkTesting() error = %v", err)
	}
	if testingWF.Status != StatusTesting {
		t.Fatalf("status = %s, want %s", testingWF.Status, StatusTesting)
	}

	activeWF, err := svc.MarkActive(context.Background(), "ws_test", created.ID)
	if err != nil {
		t.Fatalf("MarkActive() error = %v", err)
	}
	if activeWF.Status != StatusActive {
		t.Fatalf("status = %s, want %s", activeWF.Status, StatusActive)
	}

	archivedWF, err := svc.MarkArchived(context.Background(), "ws_test", created.ID)
	if err != nil {
		t.Fatalf("MarkArchived() error = %v", err)
	}
	if archivedWF.Status != StatusArchived {
		t.Fatalf("status = %s, want %s", archivedWF.Status, StatusArchived)
	}
	if archivedWF.ArchivedAt == nil {
		t.Fatal("ArchivedAt = nil, want timestamp")
	}
}

func TestService_SetStatus_AllowsRollbackShapeArchivedToActive(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewServiceWithRepository(repo)

	_, err := repo.Create(context.Background(), CreateInput{
		ID:          "wf-rollback",
		WorkspaceID: "ws_test",
		Name:        "triage_case",
		DSLSource:   "ON case.created",
		Version:     1,
		Status:      StatusArchived,
	})
	if err != nil {
		t.Fatalf("repo.Create() error = %v", err)
	}

	activeWF, err := svc.MarkActive(context.Background(), "ws_test", "wf-rollback")
	if err != nil {
		t.Fatalf("MarkActive() from archived error = %v", err)
	}
	if activeWF.Status != StatusActive {
		t.Fatalf("status = %s, want %s", activeWF.Status, StatusActive)
	}
}

func TestService_SetStatus_RejectsInvalidTransitions(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	svc := NewService(db)

	created, err := svc.Create(context.Background(), CreateWorkflowInput{
		WorkspaceID: "ws_test",
		Name:        "qualify_lead",
		DSLSource:   "ON lead.created",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	_, err = svc.MarkActive(context.Background(), "ws_test", created.ID)
	if !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("expected ErrInvalidStatusTransition for draft->active, got %v", err)
	}

	testingWF, err := svc.MarkTesting(context.Background(), "ws_test", created.ID)
	if err != nil {
		t.Fatalf("MarkTesting() error = %v", err)
	}

	_, err = svc.MarkArchived(context.Background(), "ws_test", testingWF.ID)
	if !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("expected ErrInvalidStatusTransition for testing->archived, got %v", err)
	}
}

func TestService_SetStatus_AllowsTestingBackToDraft(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	svc := NewService(db)

	created, err := svc.Create(context.Background(), CreateWorkflowInput{
		WorkspaceID: "ws_test",
		Name:        "qualify_lead",
		DSLSource:   "ON lead.created",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if _, err := svc.MarkTesting(context.Background(), "ws_test", created.ID); err != nil {
		t.Fatalf("MarkTesting() error = %v", err)
	}

	backToDraft, err := svc.SetStatus(context.Background(), "ws_test", created.ID, StatusDraft)
	if err != nil {
		t.Fatalf("SetStatus(draft) error = %v", err)
	}
	if backToDraft.Status != StatusDraft {
		t.Fatalf("status = %s, want %s", backToDraft.Status, StatusDraft)
	}
}

func TestService_NewVersion_CreatesDraftFromActive(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	svc := NewService(db)

	created, err := svc.Create(context.Background(), CreateWorkflowInput{
		WorkspaceID: "ws_test",
		Name:        "qualify_lead",
		Description: "v1",
		DSLSource:   "ON lead.created",
		SpecSource:  "BEHAVIOR define_workflow",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := svc.MarkTesting(context.Background(), "ws_test", created.ID); err != nil {
		t.Fatalf("MarkTesting() error = %v", err)
	}
	if _, err := svc.MarkActive(context.Background(), "ws_test", created.ID); err != nil {
		t.Fatalf("MarkActive() error = %v", err)
	}

	next, err := svc.NewVersion(context.Background(), "ws_test", created.ID)
	if err != nil {
		t.Fatalf("NewVersion() error = %v", err)
	}
	if next.Version != 2 {
		t.Fatalf("version = %d, want 2", next.Version)
	}
	if next.Status != StatusDraft {
		t.Fatalf("status = %s, want %s", next.Status, StatusDraft)
	}
	if next.ParentVersionID == nil || *next.ParentVersionID != created.ID {
		t.Fatalf("parent_version_id = %+v, want %s", next.ParentVersionID, created.ID)
	}
	if next.Description == nil || *next.Description != "v1" {
		t.Fatalf("description = %+v, want v1", next.Description)
	}
}

func TestService_NewVersion_RejectsNonActiveSource(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	svc := NewService(db)

	created, err := svc.Create(context.Background(), CreateWorkflowInput{
		WorkspaceID: "ws_test",
		Name:        "qualify_lead",
		DSLSource:   "ON lead.created",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	_, err = svc.NewVersion(context.Background(), "ws_test", created.ID)
	if !errors.Is(err, ErrWorkflowVersionInvalid) {
		t.Fatalf("expected ErrWorkflowVersionInvalid, got %v", err)
	}
}

func TestService_Rollback_ReactivatesArchivedVersion(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewServiceWithRepository(repo)

	_, err := repo.Create(context.Background(), CreateInput{
		ID:          "wf-archived",
		WorkspaceID: "ws_test",
		Name:        "triage_case",
		DSLSource:   "ON case.created",
		Version:     1,
		Status:      StatusArchived,
	})
	if err != nil {
		t.Fatalf("repo.Create() error = %v", err)
	}

	out, err := svc.Rollback(context.Background(), "ws_test", "wf-archived")
	if err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}
	if out.Status != StatusActive {
		t.Fatalf("status = %s, want %s", out.Status, StatusActive)
	}
}

func TestService_Rollback_RejectsNonArchived(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	svc := NewService(db)

	created, err := svc.Create(context.Background(), CreateWorkflowInput{
		WorkspaceID: "ws_test",
		Name:        "qualify_lead",
		DSLSource:   "ON lead.created",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	_, err = svc.Rollback(context.Background(), "ws_test", created.ID)
	if !errors.Is(err, ErrWorkflowVersionInvalid) {
		t.Fatalf("expected ErrWorkflowVersionInvalid, got %v", err)
	}
}

func TestService_DeleteDraft_OnlyDraft(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	svc := NewService(db)

	created, err := svc.Create(context.Background(), CreateWorkflowInput{
		WorkspaceID: "ws_test",
		Name:        "qualify_lead",
		DSLSource:   "ON lead.created",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := svc.DeleteDraft(context.Background(), "ws_test", created.ID); err != nil {
		t.Fatalf("DeleteDraft() error = %v", err)
	}

	_, err = svc.Get(context.Background(), "ws_test", created.ID)
	if !errors.Is(err, ErrWorkflowNotFound) {
		t.Fatalf("expected ErrWorkflowNotFound after delete, got %v", err)
	}
}

func TestService_DeleteDraft_RejectsNonDraft(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	svc := NewService(db)

	created, err := svc.Create(context.Background(), CreateWorkflowInput{
		WorkspaceID: "ws_test",
		Name:        "qualify_lead",
		DSLSource:   "ON lead.created",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := svc.MarkTesting(context.Background(), "ws_test", created.ID); err != nil {
		t.Fatalf("MarkTesting() error = %v", err)
	}

	err = svc.DeleteDraft(context.Background(), "ws_test", created.ID)
	if !errors.Is(err, ErrWorkflowDeleteInvalid) {
		t.Fatalf("expected ErrWorkflowDeleteInvalid, got %v", err)
	}
}

func TestService_MarkActive_RejectsWhenAnotherActiveExists(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewServiceWithRepository(repo)

	for _, input := range []CreateInput{
		{ID: "wf-active-1", WorkspaceID: "ws_test", Name: "triage_case", DSLSource: "ON case.created", Version: 1, Status: StatusActive},
		{ID: "wf-active-2", WorkspaceID: "ws_test", Name: "triage_case", DSLSource: "ON case.updated", Version: 2, Status: StatusTesting},
	} {
		if _, err := repo.Create(context.Background(), input); err != nil {
			t.Fatalf("repo.Create(%s) error = %v", input.ID, err)
		}
	}

	_, err := svc.MarkActive(context.Background(), "ws_test", "wf-active-2")
	if !errors.Is(err, ErrWorkflowActiveConflict) {
		t.Fatalf("expected ErrWorkflowActiveConflict, got %v", err)
	}
}

func TestService_Rollback_RejectsWhenAnotherActiveExists(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewServiceWithRepository(repo)

	for _, input := range []CreateInput{
		{ID: "wf-archived-1", WorkspaceID: "ws_test", Name: "triage_case", DSLSource: "ON case.created", Version: 1, Status: StatusArchived},
		{ID: "wf-active-current", WorkspaceID: "ws_test", Name: "triage_case", DSLSource: "ON case.updated", Version: 2, Status: StatusActive},
	} {
		if _, err := repo.Create(context.Background(), input); err != nil {
			t.Fatalf("repo.Create(%s) error = %v", input.ID, err)
		}
	}

	_, err := svc.Rollback(context.Background(), "ws_test", "wf-archived-1")
	if !errors.Is(err, ErrWorkflowActiveConflict) {
		t.Fatalf("expected ErrWorkflowActiveConflict, got %v", err)
	}
}

func TestService_Activate_ArchivesPreviousActiveVersion(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewServiceWithRepository(repo)

	for _, input := range []CreateInput{
		{ID: "wf-active-1", WorkspaceID: "ws_test", Name: "triage_case", DSLSource: "WORKFLOW triage_case\nON case.created\nSET case.status = \"open\"", Version: 1, Status: StatusActive},
		{ID: "wf-testing-2", WorkspaceID: "ws_test", Name: "triage_case", DSLSource: "WORKFLOW triage_case\nON case.updated\nSET case.status = \"resolved\"", Version: 2, Status: StatusTesting},
	} {
		if _, err := repo.Create(context.Background(), input); err != nil {
			t.Fatalf("repo.Create(%s) error = %v", input.ID, err)
		}
	}

	out, err := svc.Activate(context.Background(), "ws_test", "wf-testing-2")
	if err != nil {
		t.Fatalf("Activate() error = %v", err)
	}
	if out.Status != StatusActive {
		t.Fatalf("status = %s, want %s", out.Status, StatusActive)
	}

	previous, err := svc.Get(context.Background(), "ws_test", "wf-active-1")
	if err != nil {
		t.Fatalf("Get(previous) error = %v", err)
	}
	if previous.Status != StatusArchived {
		t.Fatalf("previous status = %s, want %s", previous.Status, StatusArchived)
	}
	if previous.ArchivedAt == nil {
		t.Fatal("expected ArchivedAt on previous active workflow")
	}
}

func TestWorkflowActivateBudgetSync(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewServiceWithRepository(repo)

	oldResolver := cartaBudgetLimitsResolver
	cartaBudgetLimitsResolver = func(source string) (map[string]any, error) {
		if !isCartaSource(source) {
			return nil, nil
		}
		return map[string]any{"daily_tokens": 50000}, nil
	}
	t.Cleanup(func() {
		cartaBudgetLimitsResolver = oldResolver
	})

	if _, err := db.Exec(`
		INSERT INTO agent_definition (id, workspace_id, name, agent_type, limits, status, created_at, updated_at)
		VALUES ('agent_1', 'ws_test', 'Agent 1', 'dsl', '{"existing_limit":1}', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`); err != nil {
		t.Fatalf("insert agent_definition: %v", err)
	}

	agentDefinitionID := "agent_1"
	spec := "CARTA policy\nBUDGET\n  daily_tokens: 50000\nAGENT support"
	for _, input := range []CreateInput{
		{ID: "wf-active-1", WorkspaceID: "ws_test", AgentDefinitionID: &agentDefinitionID, Name: "triage_case", DSLSource: "WORKFLOW triage_case\nON case.created\nSET case.status = \"open\"", Version: 1, Status: StatusActive},
		{ID: "wf-testing-2", WorkspaceID: "ws_test", AgentDefinitionID: &agentDefinitionID, Name: "triage_case", DSLSource: "WORKFLOW triage_case\nON case.updated\nSET case.status = \"resolved\"", SpecSource: &spec, Version: 2, Status: StatusTesting},
	} {
		if _, err := repo.Create(context.Background(), input); err != nil {
			t.Fatalf("repo.Create(%s) error = %v", input.ID, err)
		}
	}

	if _, err := svc.Activate(context.Background(), "ws_test", "wf-testing-2"); err != nil {
		t.Fatalf("Activate() error = %v", err)
	}

	got := loadAgentDefinitionLimits(t, db, "ws_test", "agent_1")
	if got["daily_tokens"] != float64(50000) {
		t.Fatalf("daily_tokens = %#v, want 50000", got["daily_tokens"])
	}
	if got["existing_limit"] != float64(1) {
		t.Fatalf("existing_limit = %#v, want 1", got["existing_limit"])
	}
}

func TestWorkflowActivateBudgetSyncNoBudgetLeavesLimitsUnchanged(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewServiceWithRepository(repo)

	oldResolver := cartaBudgetLimitsResolver
	cartaBudgetLimitsResolver = func(source string) (map[string]any, error) {
		if !isCartaSource(source) {
			return nil, nil
		}
		return nil, nil
	}
	t.Cleanup(func() {
		cartaBudgetLimitsResolver = oldResolver
	})

	if _, err := db.Exec(`
		INSERT INTO agent_definition (id, workspace_id, name, agent_type, limits, status, created_at, updated_at)
		VALUES ('agent_1', 'ws_test', 'Agent 1', 'dsl', '{"existing_limit":1}', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`); err != nil {
		t.Fatalf("insert agent_definition: %v", err)
	}

	agentDefinitionID := "agent_1"
	spec := "CARTA policy\nAGENT support"
	for _, input := range []CreateInput{
		{ID: "wf-active-1", WorkspaceID: "ws_test", AgentDefinitionID: &agentDefinitionID, Name: "triage_case", DSLSource: "WORKFLOW triage_case\nON case.created\nSET case.status = \"open\"", Version: 1, Status: StatusActive},
		{ID: "wf-testing-2", WorkspaceID: "ws_test", AgentDefinitionID: &agentDefinitionID, Name: "triage_case", DSLSource: "WORKFLOW triage_case\nON case.updated\nSET case.status = \"resolved\"", SpecSource: &spec, Version: 2, Status: StatusTesting},
	} {
		if _, err := repo.Create(context.Background(), input); err != nil {
			t.Fatalf("repo.Create(%s) error = %v", input.ID, err)
		}
	}

	if _, err := svc.Activate(context.Background(), "ws_test", "wf-testing-2"); err != nil {
		t.Fatalf("Activate() error = %v", err)
	}

	got := loadAgentDefinitionLimits(t, db, "ws_test", "agent_1")
	if len(got) != 1 || got["existing_limit"] != float64(1) {
		t.Fatalf("limits = %#v, want unchanged existing_limit only", got)
	}
}

func TestWorkflowActivateInvariantSync(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewServiceWithRepository(repo)

	oldResolver := cartaInvariantRulesResolver
	cartaInvariantRulesResolver = func(source string) ([]map[string]any, error) {
		if !isCartaSource(source) {
			return nil, nil
		}
		return []map[string]any{
			{"id": "carta_invariant_1", "resource": "tools", "action": "send_pii", "effect": "deny", "priority": 1000},
		}, nil
	}
	t.Cleanup(func() {
		cartaInvariantRulesResolver = oldResolver
	})

	if _, err := db.Exec(`
		INSERT INTO policy_set (id, workspace_id, name, description, is_active, created_at, updated_at)
		VALUES ('ps_1', 'ws_test', 'default-policy', 'test set', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`); err != nil {
		t.Fatalf("insert policy_set: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO policy_version (id, policy_set_id, workspace_id, version_number, policy_json, status, created_at)
		VALUES ('pv_1', 'ps_1', 'ws_test', 1, '{"rules":[{"id":"existing","resource":"tools","action":"update_case","effect":"allow","priority":1}]}', 'active', CURRENT_TIMESTAMP)
	`); err != nil {
		t.Fatalf("insert policy_version: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO agent_definition (id, workspace_id, name, agent_type, policy_set_id, status, created_at, updated_at)
		VALUES ('agent_1', 'ws_test', 'Agent 1', 'dsl', 'ps_1', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`); err != nil {
		t.Fatalf("insert agent_definition: %v", err)
	}

	agentDefinitionID := "agent_1"
	spec := "CARTA policy\nINVARIANT\n  never: \"send_pii\"\nAGENT support"
	for _, input := range []CreateInput{
		{ID: "wf-active-1", WorkspaceID: "ws_test", AgentDefinitionID: &agentDefinitionID, Name: "triage_case", DSLSource: "WORKFLOW triage_case\nON case.created\nSET case.status = \"open\"", Version: 1, Status: StatusActive},
		{ID: "wf-testing-2", WorkspaceID: "ws_test", AgentDefinitionID: &agentDefinitionID, Name: "triage_case", DSLSource: "WORKFLOW triage_case\nON case.updated\nSET case.status = \"resolved\"", SpecSource: &spec, Version: 2, Status: StatusTesting},
	} {
		if _, err := repo.Create(context.Background(), input); err != nil {
			t.Fatalf("repo.Create(%s) error = %v", input.ID, err)
		}
	}

	if _, err := svc.Activate(context.Background(), "ws_test", "wf-testing-2"); err != nil {
		t.Fatalf("Activate() error = %v", err)
	}

	rules := loadPolicyVersionRules(t, db, "pv_1")
	if len(rules) != 2 {
		t.Fatalf("len(rules) = %d, want 2", len(rules))
	}
	if !policyRuleExists(rules, "send_pii", "deny") {
		t.Fatalf("expected send_pii deny rule in %#v", rules)
	}
}

func TestWorkflowActivateInvariantSyncDoesNotDuplicateAction(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewServiceWithRepository(repo)

	oldResolver := cartaInvariantRulesResolver
	cartaInvariantRulesResolver = func(source string) ([]map[string]any, error) {
		if !isCartaSource(source) {
			return nil, nil
		}
		return []map[string]any{
			{"id": "carta_invariant_1", "resource": "tools", "action": "send_pii", "effect": "deny", "priority": 1000},
		}, nil
	}
	t.Cleanup(func() {
		cartaInvariantRulesResolver = oldResolver
	})

	if _, err := db.Exec(`
		INSERT INTO policy_set (id, workspace_id, name, description, is_active, created_at, updated_at)
		VALUES ('ps_1', 'ws_test', 'default-policy', 'test set', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`); err != nil {
		t.Fatalf("insert policy_set: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO policy_version (id, policy_set_id, workspace_id, version_number, policy_json, status, created_at)
		VALUES ('pv_1', 'ps_1', 'ws_test', 1, '{\"rules\":[{\"id\":\"existing\",\"resource\":\"tools\",\"action\":\"send_pii\",\"effect\":\"allow\",\"priority\":1}]}', 'active', CURRENT_TIMESTAMP)
	`); err != nil {
		t.Fatalf("insert policy_version: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO agent_definition (id, workspace_id, name, agent_type, policy_set_id, status, created_at, updated_at)
		VALUES ('agent_1', 'ws_test', 'Agent 1', 'dsl', 'ps_1', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`); err != nil {
		t.Fatalf("insert agent_definition: %v", err)
	}

	agentDefinitionID := "agent_1"
	spec := "CARTA policy\nINVARIANT\n  never: \"send_pii\"\nAGENT support"
	for _, input := range []CreateInput{
		{ID: "wf-active-1", WorkspaceID: "ws_test", AgentDefinitionID: &agentDefinitionID, Name: "triage_case", DSLSource: "WORKFLOW triage_case\nON case.created\nSET case.status = \"open\"", Version: 1, Status: StatusActive},
		{ID: "wf-testing-2", WorkspaceID: "ws_test", AgentDefinitionID: &agentDefinitionID, Name: "triage_case", DSLSource: "WORKFLOW triage_case\nON case.updated\nSET case.status = \"resolved\"", SpecSource: &spec, Version: 2, Status: StatusTesting},
	} {
		if _, err := repo.Create(context.Background(), input); err != nil {
			t.Fatalf("repo.Create(%s) error = %v", input.ID, err)
		}
	}

	if _, err := svc.Activate(context.Background(), "ws_test", "wf-testing-2"); err != nil {
		t.Fatalf("Activate() error = %v", err)
	}

	rules := loadPolicyVersionRules(t, db, "pv_1")
	if countPolicyRuleAction(rules, "send_pii") != 1 {
		t.Fatalf("expected one send_pii rule, got %#v", rules)
	}
	if !policyRuleExists(rules, "send_pii", "deny") {
		t.Fatalf("expected send_pii deny rule in %#v", rules)
	}
}

func loadPolicyVersionRules(t *testing.T, db *sql.DB, versionID string) []map[string]any {
	t.Helper()

	var raw string
	if err := db.QueryRow(`
		SELECT policy_json
		FROM policy_version
		WHERE id = ?
	`, versionID).Scan(&raw); err != nil {
		t.Fatalf("select policy_version: %v", err)
	}

	var doc struct {
		Rules []map[string]any `json:"rules"`
	}
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatalf("unmarshal policy_json: %v", err)
	}
	return doc.Rules
}

func policyRuleExists(rules []map[string]any, action, effect string) bool {
	for _, rule := range rules {
		if rule["action"] == action && rule["effect"] == effect {
			return true
		}
	}
	return false
}

func countPolicyRuleAction(rules []map[string]any, action string) int {
	count := 0
	for _, rule := range rules {
		if rule["action"] == action {
			count++
		}
	}
	return count
}

func loadAgentDefinitionLimits(t *testing.T, db *sql.DB, workspaceID, agentDefinitionID string) map[string]any {
	t.Helper()

	var raw sql.NullString
	if err := db.QueryRow(`
		SELECT limits
		FROM agent_definition
		WHERE workspace_id = ? AND id = ?
	`, workspaceID, agentDefinitionID).Scan(&raw); err != nil {
		t.Fatalf("select agent_definition limits: %v", err)
	}

	out := make(map[string]any)
	if raw.Valid && strings.TrimSpace(raw.String) != "" {
		if err := json.Unmarshal([]byte(raw.String), &out); err != nil {
			t.Fatalf("unmarshal limits: %v", err)
		}
	}
	return out
}

func TestService_Activate_RejectsNonTestingWorkflow(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	svc := NewService(db)

	created, err := svc.Create(context.Background(), CreateWorkflowInput{
		WorkspaceID: "ws_test",
		Name:        "triage_case",
		DSLSource:   "WORKFLOW triage_case\nON case.created\nSET case.status = \"open\"",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	_, err = svc.Activate(context.Background(), "ws_test", created.ID)
	if !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("expected ErrInvalidStatusTransition, got %v", err)
	}
}

func TestService_MarkArchived_CancelsPendingScheduledJobs(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)
	schedulerRepo := schedulerdomain.NewRepository(db)
	schedulerSvc := schedulerdomain.NewService(schedulerRepo)
	svc := NewServiceWithDependencies(repo, schedulerSvc)

	created, err := svc.Create(context.Background(), CreateWorkflowInput{
		WorkspaceID: "ws_test",
		Name:        "qualify_lead",
		DSLSource:   "ON lead.created",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := svc.MarkTesting(context.Background(), "ws_test", created.ID); err != nil {
		t.Fatalf("MarkTesting() error = %v", err)
	}
	if _, err := svc.MarkActive(context.Background(), "ws_test", created.ID); err != nil {
		t.Fatalf("MarkActive() error = %v", err)
	}

	for _, payload := range []schedulerdomain.WorkflowResumePayload{
		{WorkflowID: created.ID, RunID: "run-1", ResumeStepIndex: 1},
		{WorkflowID: created.ID, RunID: "run-2", ResumeStepIndex: 2},
	} {
		if _, err := schedulerSvc.Schedule(context.Background(), schedulerdomain.ScheduleJobInput{
			WorkspaceID: "ws_test",
			JobType:     schedulerdomain.JobTypeWorkflowResume,
			Payload:     payload,
			ExecuteAt:   time.Now().UTC().Add(1 * time.Hour),
			SourceID:    created.ID,
		}); err != nil {
			t.Fatalf("Schedule() error = %v", err)
		}
	}

	archived, err := svc.MarkArchived(context.Background(), "ws_test", created.ID)
	if err != nil {
		t.Fatalf("MarkArchived() error = %v", err)
	}
	if archived.Status != StatusArchived {
		t.Fatalf("status = %s, want %s", archived.Status, StatusArchived)
	}

	due, err := schedulerRepo.ListDue(context.Background(), time.Now().UTC().Add(2*time.Hour), 10)
	if err != nil {
		t.Fatalf("ListDue() error = %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("len(due) = %d, want 0 after archive cancellation", len(due))
	}
}

func TestService_MarkArchived_WithoutSchedulerStillArchives(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	svc := NewService(db)

	created, err := svc.Create(context.Background(), CreateWorkflowInput{
		WorkspaceID: "ws_test",
		Name:        "qualify_lead",
		DSLSource:   "ON lead.created",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := svc.MarkTesting(context.Background(), "ws_test", created.ID); err != nil {
		t.Fatalf("MarkTesting() error = %v", err)
	}
	if _, err := svc.MarkActive(context.Background(), "ws_test", created.ID); err != nil {
		t.Fatalf("MarkActive() error = %v", err)
	}

	archived, err := svc.MarkArchived(context.Background(), "ws_test", created.ID)
	if err != nil {
		t.Fatalf("MarkArchived() error = %v", err)
	}
	if archived.Status != StatusArchived {
		t.Fatalf("status = %s, want %s", archived.Status, StatusArchived)
	}
}
