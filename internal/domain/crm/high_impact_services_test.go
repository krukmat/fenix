// Traces: FR-001
package crm_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
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

func TestCaseService_NewCaseServiceWithBus_PublishesCreatedEvent(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	bus := eventbus.New()
	svc := crm.NewCaseServiceWithBus(db, bus)

	createdCh := bus.Subscribe("record.created")

	_, err := svc.Create(context.Background(), crm.CreateCaseInput{
		WorkspaceID: wsID,
		OwnerID:     ownerID,
		Subject:     "Case with bus",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	select {
	case <-createdCh:
	case <-time.After(1 * time.Second):
		t.Fatal("expected created event")
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

func TestDealService_List_FilterByAccount(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	now := time.Now().UTC().Format(time.RFC3339)

	accountA := "acc-" + randID()
	accountB := "acc-" + randID()
	for _, accountID := range []string{accountA, accountB} {
		if _, err := db.Exec(`INSERT INTO account (id, workspace_id, name, owner_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, accountID, wsID, "Acme-"+accountID, ownerID, now, now); err != nil {
			t.Fatalf("seed account error = %v", err)
		}
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
	_, err := svc.Create(context.Background(), crm.CreateDealInput{
		WorkspaceID: wsID, AccountID: accountA, PipelineID: pipelineID, StageID: stageID, OwnerID: ownerID, Title: "Deal A",
	})
	if err != nil {
		t.Fatalf("seed deal A error = %v", err)
	}
	_, err = svc.Create(context.Background(), crm.CreateDealInput{
		WorkspaceID: wsID, AccountID: accountB, PipelineID: pipelineID, StageID: stageID, OwnerID: ownerID, Title: "Deal B",
	})
	if err != nil {
		t.Fatalf("seed deal B error = %v", err)
	}

	items, total, err := svc.List(context.Background(), wsID, crm.ListDealsInput{
		Limit:     10,
		Offset:    0,
		AccountID: accountA,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("expected one filtered deal, total=%d len=%d", total, len(items))
	}
	if items[0].AccountID != accountA {
		t.Fatalf("expected account=%s got %s", accountA, items[0].AccountID)
	}
}

func TestCaseService_List_FilterByPriority(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	svc := crm.NewCaseService(db)

	_, err := svc.Create(context.Background(), crm.CreateCaseInput{
		WorkspaceID: wsID,
		OwnerID:     ownerID,
		Subject:     "Urgent case",
		Priority:    "urgent",
	})
	if err != nil {
		t.Fatalf("seed urgent case error = %v", err)
	}
	_, err = svc.Create(context.Background(), crm.CreateCaseInput{
		WorkspaceID: wsID,
		OwnerID:     ownerID,
		Subject:     "Low case",
		Priority:    "low",
	})
	if err != nil {
		t.Fatalf("seed low case error = %v", err)
	}

	items, total, err := svc.List(context.Background(), wsID, crm.ListCasesInput{
		Limit:    10,
		Offset:   0,
		Priority: "urgent",
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("expected one filtered case, total=%d len=%d", total, len(items))
	}
	if items[0].Priority != "urgent" {
		t.Fatalf("expected urgent priority, got %s", items[0].Priority)
	}
}

func TestNoteService_CRUDAndReadPaths(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, authorID := setupWorkspaceAndOwner(t, db)
	svc := crm.NewNoteService(db)

	created, err := svc.Create(context.Background(), crm.CreateNoteInput{
		WorkspaceID: wsID,
		EntityType:  "account",
		EntityID:    "acc-1",
		AuthorID:    authorID,
		Content:     "hello",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	noteID := created.ID

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

	if _, err = svc.Update(context.Background(), wsID, noteID, crm.UpdateNoteInput{Content: "updated", IsInternal: true}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if err = svc.Delete(context.Background(), wsID, noteID); err != nil {
		t.Fatalf("Delete() error = %v", err)
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

func TestDealService_List_FilterByStage(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	now := time.Now().UTC().Format(time.RFC3339)

	accountID := "acc-fs-" + randID()
	if _, err := db.Exec(`INSERT INTO account (id, workspace_id, name, owner_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, accountID, wsID, "Acme", ownerID, now, now); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	pipelineID := "pl-fs-" + randID()
	if _, err := db.Exec(`INSERT INTO pipeline (id, workspace_id, name, entity_type, created_at, updated_at) VALUES (?, ?, ?, 'deal', ?, ?)`, pipelineID, wsID, "Sales", now, now); err != nil {
		t.Fatalf("seed pipeline: %v", err)
	}
	stageID := "st-fs-" + randID()
	if _, err := db.Exec(`INSERT INTO pipeline_stage (id, pipeline_id, name, position, created_at, updated_at) VALUES (?, ?, ?, 1, ?, ?)`, stageID, pipelineID, "Discovery", now, now); err != nil {
		t.Fatalf("seed stage: %v", err)
	}

	svc := crm.NewDealService(db)
	if _, err := svc.Create(context.Background(), crm.CreateDealInput{WorkspaceID: wsID, AccountID: accountID, PipelineID: pipelineID, StageID: stageID, OwnerID: ownerID, Title: "Staged Deal"}); err != nil {
		t.Fatalf("seed deal: %v", err)
	}

	items, total, err := svc.List(context.Background(), wsID, crm.ListDealsInput{Limit: 10, StageID: stageID})
	if err != nil {
		t.Fatalf("List by stage error = %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("expected 1 deal, got total=%d len=%d", total, len(items))
	}
}

func TestDealService_List_FilterByOwner(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	now := time.Now().UTC().Format(time.RFC3339)

	accountID := "acc-fo-" + randID()
	if _, err := db.Exec(`INSERT INTO account (id, workspace_id, name, owner_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, accountID, wsID, "Acme", ownerID, now, now); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	pipelineID := "pl-fo-" + randID()
	if _, err := db.Exec(`INSERT INTO pipeline (id, workspace_id, name, entity_type, created_at, updated_at) VALUES (?, ?, ?, 'deal', ?, ?)`, pipelineID, wsID, "Sales", now, now); err != nil {
		t.Fatalf("seed pipeline: %v", err)
	}
	stageID := "st-fo-" + randID()
	if _, err := db.Exec(`INSERT INTO pipeline_stage (id, pipeline_id, name, position, created_at, updated_at) VALUES (?, ?, ?, 1, ?, ?)`, stageID, pipelineID, "Discovery", now, now); err != nil {
		t.Fatalf("seed stage: %v", err)
	}

	svc := crm.NewDealService(db)
	if _, err := svc.Create(context.Background(), crm.CreateDealInput{WorkspaceID: wsID, AccountID: accountID, PipelineID: pipelineID, StageID: stageID, OwnerID: ownerID, Title: "Owner Deal"}); err != nil {
		t.Fatalf("seed deal: %v", err)
	}

	items, _, err := svc.List(context.Background(), wsID, crm.ListDealsInput{Limit: 10, OwnerID: ownerID})
	if err != nil {
		t.Fatalf("List by owner error = %v", err)
	}
	if len(items) < 1 {
		t.Fatalf("expected at least 1 deal by owner")
	}
}

func TestDealService_List_FilterByStatus(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	now := time.Now().UTC().Format(time.RFC3339)

	accountID := "acc-fst-" + randID()
	if _, err := db.Exec(`INSERT INTO account (id, workspace_id, name, owner_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, accountID, wsID, "Acme", ownerID, now, now); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	pipelineID := "pl-fst-" + randID()
	if _, err := db.Exec(`INSERT INTO pipeline (id, workspace_id, name, entity_type, created_at, updated_at) VALUES (?, ?, ?, 'deal', ?, ?)`, pipelineID, wsID, "Sales", now, now); err != nil {
		t.Fatalf("seed pipeline: %v", err)
	}
	stageID := "st-fst-" + randID()
	if _, err := db.Exec(`INSERT INTO pipeline_stage (id, pipeline_id, name, position, created_at, updated_at) VALUES (?, ?, ?, 1, ?, ?)`, stageID, pipelineID, "Discovery", now, now); err != nil {
		t.Fatalf("seed stage: %v", err)
	}

	svc := crm.NewDealService(db)
	if _, err := svc.Create(context.Background(), crm.CreateDealInput{WorkspaceID: wsID, AccountID: accountID, PipelineID: pipelineID, StageID: stageID, OwnerID: ownerID, Title: "Open Deal"}); err != nil {
		t.Fatalf("seed deal: %v", err)
	}

	items, _, err := svc.List(context.Background(), wsID, crm.ListDealsInput{Limit: 10, Status: "open"})
	if err != nil {
		t.Fatalf("List by status error = %v", err)
	}
	if len(items) < 1 {
		t.Fatalf("expected at least 1 open deal")
	}
}

func TestDealService_List_SortAscending(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	now := time.Now().UTC().Format(time.RFC3339)

	accountID := "acc-sort-" + randID()
	if _, err := db.Exec(`INSERT INTO account (id, workspace_id, name, owner_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, accountID, wsID, "Acme", ownerID, now, now); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	pipelineID := "pl-sort-" + randID()
	if _, err := db.Exec(`INSERT INTO pipeline (id, workspace_id, name, entity_type, created_at, updated_at) VALUES (?, ?, ?, 'deal', ?, ?)`, pipelineID, wsID, "Sales", now, now); err != nil {
		t.Fatalf("seed pipeline: %v", err)
	}
	stageID := "st-sort-" + randID()
	if _, err := db.Exec(`INSERT INTO pipeline_stage (id, pipeline_id, name, position, created_at, updated_at) VALUES (?, ?, ?, 1, ?, ?)`, stageID, pipelineID, "Discovery", now, now); err != nil {
		t.Fatalf("seed stage: %v", err)
	}

	svc := crm.NewDealService(db)
	for _, title := range []string{"Alpha", "Beta"} {
		if _, err := svc.Create(context.Background(), crm.CreateDealInput{WorkspaceID: wsID, AccountID: accountID, PipelineID: pipelineID, StageID: stageID, OwnerID: ownerID, Title: title}); err != nil {
			t.Fatalf("seed deal %s: %v", title, err)
		}
	}

	items, _, err := svc.List(context.Background(), wsID, crm.ListDealsInput{Limit: 10, Sort: "created_at"})
	if err != nil {
		t.Fatalf("List ascending error = %v", err)
	}
	if len(items) < 2 {
		t.Fatalf("expected 2 deals, got %d", len(items))
	}
	if !items[0].CreatedAt.Before(items[1].CreatedAt) && items[0].CreatedAt != items[1].CreatedAt {
		t.Fatalf("expected ascending order")
	}
}

func TestCaseService_List_SortAscending(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	svc := crm.NewCaseService(db)

	for i := range 3 {
		if _, err := svc.Create(context.Background(), crm.CreateCaseInput{WorkspaceID: wsID, OwnerID: ownerID, Subject: fmt.Sprintf("Case %d", i), Priority: "medium"}); err != nil {
			t.Fatalf("seed case %d: %v", i, err)
		}
	}

	items, _, err := svc.List(context.Background(), wsID, crm.ListCasesInput{Limit: 10, Sort: "created_at"})
	if err != nil {
		t.Fatalf("List ascending error = %v", err)
	}
	if len(items) < 2 {
		t.Fatalf("expected at least 2 cases, got %d", len(items))
	}
}

func TestCaseService_List_FilterByOwner(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	svc := crm.NewCaseService(db)

	if _, err := svc.Create(context.Background(), crm.CreateCaseInput{WorkspaceID: wsID, OwnerID: ownerID, Subject: "Owned Case", Priority: "high"}); err != nil {
		t.Fatalf("seed case: %v", err)
	}

	items, _, err := svc.List(context.Background(), wsID, crm.ListCasesInput{Limit: 10, OwnerID: ownerID})
	if err != nil {
		t.Fatalf("List by owner error = %v", err)
	}
	if len(items) < 1 {
		t.Fatalf("expected at least 1 case")
	}
}
