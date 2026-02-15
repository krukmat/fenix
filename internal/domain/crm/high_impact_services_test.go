// Traces: FR-001
package crm_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
)

func TestLeadService_CRUD(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	svc := crm.NewLeadService(db)

	created, err := svc.Create(context.Background(), crm.CreateLeadInput{
		WorkspaceID: wsID,
		OwnerID:     ownerID,
		Status:      "new",
		Source:      "web",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := svc.Get(context.Background(), wsID, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.OwnerID != ownerID {
		t.Fatalf("owner mismatch: got %q want %q", got.OwnerID, ownerID)
	}

	items, total, err := svc.List(context.Background(), wsID, crm.ListLeadsInput{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total < 1 || len(items) < 1 {
		t.Fatalf("expected at least one lead, total=%d len=%d", total, len(items))
	}

	byOwner, err := svc.ListByOwner(context.Background(), wsID, ownerID)
	if err != nil {
		t.Fatalf("ListByOwner() error = %v", err)
	}
	if len(byOwner) < 1 {
		t.Fatalf("expected at least one lead by owner")
	}

	updated, err := svc.Update(context.Background(), wsID, created.ID, crm.UpdateLeadInput{
		Status:  "qualified",
		OwnerID: ownerID,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Status != "qualified" {
		t.Fatalf("expected qualified, got %q", updated.Status)
	}

	if err := svc.Delete(context.Background(), wsID, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}

func TestCaseService_CRUD(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	svc := crm.NewCaseService(db)

	created, err := svc.Create(context.Background(), crm.CreateCaseInput{
		WorkspaceID: wsID,
		OwnerID:     ownerID,
		Subject:     "Case Subject",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := svc.Get(context.Background(), wsID, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Subject != "Case Subject" {
		t.Fatalf("subject mismatch: got %q", got.Subject)
	}

	items, total, err := svc.List(context.Background(), wsID, crm.ListCasesInput{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total < 1 || len(items) < 1 {
		t.Fatalf("expected at least one case, total=%d len=%d", total, len(items))
	}

	updated, err := svc.Update(context.Background(), wsID, created.ID, crm.UpdateCaseInput{
		OwnerID:  ownerID,
		Subject:  "Case Updated",
		Priority: "high",
		Status:   "in_progress",
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Subject != "Case Updated" {
		t.Fatalf("expected updated subject, got %q", updated.Subject)
	}

	if err := svc.Delete(context.Background(), wsID, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}

func TestDealService_CRUD(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	now := time.Now().UTC().Format(time.RFC3339)

	accountID := "acc-" + randID()
	if _, err := db.Exec(`INSERT INTO account (id, workspace_id, name, owner_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, accountID, wsID, "Acme", ownerID, now, now); err != nil {
		t.Fatalf("seed account error = %v", err)
	}

	pipelineID := "pl-" + randID()
	if _, err := db.Exec(`INSERT INTO pipeline (id, workspace_id, name, entity_type, created_at, updated_at) VALUES (?, ?, ?, 'deal', ?, ?)`, pipelineID, wsID, "Sales", now, now); err != nil {
		t.Fatalf("seed pipeline error = %v", err)
	}

	stageID := "st-" + randID()
	if _, err := db.Exec(`INSERT INTO pipeline_stage (id, pipeline_id, name, position, created_at, updated_at) VALUES (?, ?, ?, 1, ?, ?)`, stageID, pipelineID, "Discovery", now, now); err != nil {
		t.Fatalf("seed stage error = %v", err)
	}

	svc := crm.NewDealService(db)

	created, err := svc.Create(context.Background(), crm.CreateDealInput{
		WorkspaceID: wsID,
		AccountID:   accountID,
		PipelineID:  pipelineID,
		StageID:     stageID,
		OwnerID:     ownerID,
		Title:       "Deal 1",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := svc.Get(context.Background(), wsID, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Title != "Deal 1" {
		t.Fatalf("title mismatch: got %q", got.Title)
	}

	items, total, err := svc.List(context.Background(), wsID, crm.ListDealsInput{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total < 1 || len(items) < 1 {
		t.Fatalf("expected at least one deal, total=%d len=%d", total, len(items))
	}

	updated, err := svc.Update(context.Background(), wsID, created.ID, crm.UpdateDealInput{
		AccountID:  accountID,
		PipelineID: pipelineID,
		StageID:    stageID,
		OwnerID:    ownerID,
		Title:      "Deal Updated",
		Status:     "open",
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Title != "Deal Updated" {
		t.Fatalf("expected updated title, got %q", updated.Title)
	}

	if err := svc.Delete(context.Background(), wsID, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}

func TestNoteService_TimelineConstraintAndReadPaths(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, authorID := setupWorkspaceAndOwner(t, db)
	svc := crm.NewNoteService(db)

	_, err := svc.Create(context.Background(), crm.CreateNoteInput{
		WorkspaceID: wsID,
		EntityType:  "account",
		EntityID:    "acc-1",
		AuthorID:    authorID,
		Content:     "hello",
	})
	if err == nil || !strings.Contains(err.Error(), "create note timeline") {
		t.Fatalf("expected create note timeline error, got %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	noteID := "note-" + randID()
	if _, err := db.Exec(`INSERT INTO note (id, workspace_id, entity_type, entity_id, author_id, content, is_internal, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, 0, ?, ?)`, noteID, wsID, "account", "acc-1", authorID, "seeded", now, now); err != nil {
		t.Fatalf("seed note error = %v", err)
	}

	if _, err := svc.Get(context.Background(), wsID, noteID); err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	items, total, err := svc.List(context.Background(), wsID, crm.ListNotesInput{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total < 1 || len(items) < 1 {
		t.Fatalf("expected notes, total=%d len=%d", total, len(items))
	}

	_, err = svc.Update(context.Background(), wsID, noteID, crm.UpdateNoteInput{Content: "updated", IsInternal: true})
	if err == nil || !strings.Contains(err.Error(), "update note timeline") {
		t.Fatalf("expected update note timeline error, got %v", err)
	}

	err = svc.Delete(context.Background(), wsID, noteID)
	if err == nil || !strings.Contains(err.Error(), "delete note timeline") {
		t.Fatalf("expected delete note timeline error, got %v", err)
	}
}

func TestTimelineService_CreateGetListByEntity(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, actorID := setupWorkspaceAndOwner(t, db)
	svc := crm.NewTimelineService(db)

	created, err := svc.Create(context.Background(), crm.CreateTimelineEventInput{
		WorkspaceID: wsID,
		EntityType:  "account",
		EntityID:    "acc-1",
		ActorID:     actorID,
		EventType:   "created",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if _, err := svc.Get(context.Background(), wsID, created.ID); err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if _, total, err := svc.List(context.Background(), wsID, crm.ListTimelineInput{Limit: 10, Offset: 0}); err != nil || total < 1 {
		t.Fatalf("List() err=%v total=%d", err, total)
	}

	items, err := svc.ListByEntity(context.Background(), wsID, "account", "acc-1", crm.ListTimelineInput{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("ListByEntity() error = %v", err)
	}
	if len(items) < 1 {
		t.Fatalf("expected at least one timeline item")
	}
}
