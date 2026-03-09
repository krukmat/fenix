// Traces: FR-001
package crm_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
)

func TestAccountAndContactServices_EmitAuditMutations(t *testing.T) {
	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)

	accountSvc := crm.NewAccountService(db)
	contactSvc := crm.NewContactService(db)

	account, err := accountSvc.Create(context.Background(), crm.CreateAccountInput{
		WorkspaceID: wsID,
		Name:        "Acme",
		OwnerID:     ownerID,
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	if _, err = accountSvc.Update(context.Background(), wsID, account.ID, crm.UpdateAccountInput{
		Name:    "Acme Updated",
		OwnerID: ownerID,
	}); err != nil {
		t.Fatalf("update account: %v", err)
	}
	if err = accountSvc.Delete(context.Background(), wsID, account.ID); err != nil {
		t.Fatalf("delete account: %v", err)
	}

	account2 := createAccount(t, db, wsID, ownerID)
	contact, err := contactSvc.Create(context.Background(), crm.CreateContactInput{
		WorkspaceID: wsID,
		AccountID:   account2,
		FirstName:   "Ada",
		LastName:    "Lovelace",
		OwnerID:     ownerID,
	})
	if err != nil {
		t.Fatalf("create contact: %v", err)
	}
	if _, err = contactSvc.Update(context.Background(), wsID, contact.ID, crm.UpdateContactInput{
		AccountID: account2,
		FirstName: "Ada",
		LastName:  "Byron",
		OwnerID:   ownerID,
		Status:    "active",
	}); err != nil {
		t.Fatalf("update contact: %v", err)
	}
	if err = contactSvc.Delete(context.Background(), wsID, contact.ID); err != nil {
		t.Fatalf("delete contact: %v", err)
	}

	assertAuditCount(t, db, wsID, "account.created", 1)
	assertAuditCount(t, db, wsID, "account.updated", 1)
	assertAuditCount(t, db, wsID, "account.deleted", 1)
	assertAuditCount(t, db, wsID, "contact.created", 1)
	assertAuditCount(t, db, wsID, "contact.updated", 1)
	assertAuditCount(t, db, wsID, "contact.deleted", 1)
}

func TestLeadDealCaseNoteServices_EmitAuditAndTimeline(t *testing.T) {
	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	now := time.Now().UTC().Format(time.RFC3339)

	leadSvc := crm.NewLeadService(db)
	dealSvc := crm.NewDealService(db)
	caseSvc := crm.NewCaseService(db)
	noteSvc := crm.NewNoteService(db)

	lead, err := leadSvc.Create(context.Background(), crm.CreateLeadInput{
		WorkspaceID: wsID,
		OwnerID:     ownerID,
		Status:      "new",
	})
	if err != nil {
		t.Fatalf("create lead: %v", err)
	}
	if _, err = leadSvc.Update(context.Background(), wsID, lead.ID, crm.UpdateLeadInput{
		OwnerID: ownerID,
		Status:  "qualified",
	}); err != nil {
		t.Fatalf("update lead: %v", err)
	}
	if err = leadSvc.Delete(context.Background(), wsID, lead.ID); err != nil {
		t.Fatalf("delete lead: %v", err)
	}

	accountID := createAccount(t, db, wsID, ownerID)
	pipelineID := "pl-" + randID()
	stageID := "st-" + randID()
	if _, err := db.Exec(`INSERT INTO pipeline (id, workspace_id, name, entity_type, created_at, updated_at) VALUES (?, ?, 'Sales', 'deal', ?, ?)`, pipelineID, wsID, now, now); err != nil {
		t.Fatalf("seed pipeline: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO pipeline_stage (id, pipeline_id, name, position, created_at, updated_at) VALUES (?, ?, 'Discovery', 1, ?, ?)`, stageID, pipelineID, now, now); err != nil {
		t.Fatalf("seed stage: %v", err)
	}

	deal, err := dealSvc.Create(context.Background(), crm.CreateDealInput{
		WorkspaceID: wsID,
		AccountID:   accountID,
		PipelineID:  pipelineID,
		StageID:     stageID,
		OwnerID:     ownerID,
		Title:       "Deal",
	})
	if err != nil {
		t.Fatalf("create deal: %v", err)
	}
	if _, err = dealSvc.Update(context.Background(), wsID, deal.ID, crm.UpdateDealInput{
		AccountID:  accountID,
		PipelineID: pipelineID,
		StageID:    stageID,
		OwnerID:    ownerID,
		Title:      "Deal Updated",
		Status:     "open",
	}); err != nil {
		t.Fatalf("update deal: %v", err)
	}
	if err = dealSvc.Delete(context.Background(), wsID, deal.ID); err != nil {
		t.Fatalf("delete deal: %v", err)
	}

	caseTicket, err := caseSvc.Create(context.Background(), crm.CreateCaseInput{
		WorkspaceID: wsID,
		OwnerID:     ownerID,
		Subject:     "Case",
		Priority:    "medium",
	})
	if err != nil {
		t.Fatalf("create case: %v", err)
	}
	if _, err = caseSvc.Update(context.Background(), wsID, caseTicket.ID, crm.UpdateCaseInput{
		OwnerID:  ownerID,
		Subject:  "Case Updated",
		Priority: "high",
		Status:   "in_progress",
	}); err != nil {
		t.Fatalf("update case: %v", err)
	}
	if err = caseSvc.Delete(context.Background(), wsID, caseTicket.ID); err != nil {
		t.Fatalf("delete case: %v", err)
	}

	note, err := noteSvc.Create(context.Background(), crm.CreateNoteInput{
		WorkspaceID: wsID,
		EntityType:  "account",
		EntityID:    accountID,
		AuthorID:    ownerID,
		Content:     "hello",
	})
	if err != nil {
		t.Fatalf("create note: %v", err)
	}
	if _, err = noteSvc.Update(context.Background(), wsID, note.ID, crm.UpdateNoteInput{
		Content:    "updated",
		IsInternal: true,
	}); err != nil {
		t.Fatalf("update note: %v", err)
	}
	if err = noteSvc.Delete(context.Background(), wsID, note.ID); err != nil {
		t.Fatalf("delete note: %v", err)
	}

	assertAuditCount(t, db, wsID, "lead.created", 1)
	assertAuditCount(t, db, wsID, "lead.updated", 1)
	assertAuditCount(t, db, wsID, "lead.deleted", 1)
	assertAuditCount(t, db, wsID, "deal.created", 1)
	assertAuditCount(t, db, wsID, "deal.updated", 1)
	assertAuditCount(t, db, wsID, "deal.deleted", 1)
	assertAuditCount(t, db, wsID, "case.created", 1)
	assertAuditCount(t, db, wsID, "case.updated", 1)
	assertAuditCount(t, db, wsID, "case.deleted", 1)
	assertAuditCount(t, db, wsID, "note.created", 1)
	assertAuditCount(t, db, wsID, "note.updated", 1)
	assertAuditCount(t, db, wsID, "note.deleted", 1)

	assertTimelineCount(t, db, wsID, "lead", lead.ID, "created", 1)
	assertTimelineCount(t, db, wsID, "lead", lead.ID, "updated", 1)
	assertTimelineCount(t, db, wsID, "lead", lead.ID, "deleted", 1)
	assertTimelineCount(t, db, wsID, "deal", deal.ID, "created", 1)
	assertTimelineCount(t, db, wsID, "deal", deal.ID, "updated", 1)
	assertTimelineCount(t, db, wsID, "deal", deal.ID, "deleted", 1)
	assertTimelineCount(t, db, wsID, "case_ticket", caseTicket.ID, "created", 1)
	assertTimelineCount(t, db, wsID, "case_ticket", caseTicket.ID, "updated", 1)
	assertTimelineCount(t, db, wsID, "case_ticket", caseTicket.ID, "deleted", 1)
	assertTimelineCount(t, db, wsID, "account", accountID, "note_added", 1)
	assertTimelineCount(t, db, wsID, "account", accountID, "updated", 1)
	assertTimelineCount(t, db, wsID, "account", accountID, "deleted", 1)
}

func TestDealService_Update_RejectsInvalidBusinessState(t *testing.T) {
	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	now := time.Now().UTC().Format(time.RFC3339)

	accountID := createAccount(t, db, wsID, ownerID)
	pipelineA := "pl-a-" + randID()
	stageA := "st-a-" + randID()
	pipelineB := "pl-b-" + randID()
	stageB := "st-b-" + randID()
	for _, pipelineID := range []string{pipelineA, pipelineB} {
		if _, err := db.Exec(`INSERT INTO pipeline (id, workspace_id, name, entity_type, created_at, updated_at) VALUES (?, ?, ?, 'deal', ?, ?)`, pipelineID, wsID, pipelineID, now, now); err != nil {
			t.Fatalf("seed pipeline %s: %v", pipelineID, err)
		}
	}
	if _, err := db.Exec(`INSERT INTO pipeline_stage (id, pipeline_id, name, position, created_at, updated_at) VALUES (?, ?, 'A', 1, ?, ?)`, stageA, pipelineA, now, now); err != nil {
		t.Fatalf("seed stage a: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO pipeline_stage (id, pipeline_id, name, position, created_at, updated_at) VALUES (?, ?, 'B', 1, ?, ?)`, stageB, pipelineB, now, now); err != nil {
		t.Fatalf("seed stage b: %v", err)
	}

	svc := crm.NewDealService(db)
	deal, err := svc.Create(context.Background(), crm.CreateDealInput{
		WorkspaceID: wsID,
		AccountID:   accountID,
		PipelineID:  pipelineA,
		StageID:     stageA,
		OwnerID:     ownerID,
		Title:       "Deal",
	})
	if err != nil {
		t.Fatalf("create deal: %v", err)
	}

	_, err = svc.Update(context.Background(), wsID, deal.ID, crm.UpdateDealInput{
		AccountID:  accountID,
		PipelineID: pipelineA,
		StageID:    stageB,
		OwnerID:    ownerID,
		Title:      "Deal",
		Status:     "open",
	})
	if !errors.Is(err, crm.ErrInvalidDealInput) {
		t.Fatalf("expected ErrInvalidDealInput for wrong stage/pipeline, got %v", err)
	}

	negative := -5.0
	_, err = svc.Update(context.Background(), wsID, deal.ID, crm.UpdateDealInput{
		AccountID:  accountID,
		PipelineID: pipelineA,
		StageID:    stageA,
		OwnerID:    ownerID,
		Title:      "Deal",
		Status:     "open",
		Amount:     &negative,
	})
	if !errors.Is(err, crm.ErrInvalidDealInput) {
		t.Fatalf("expected ErrInvalidDealInput for negative amount, got %v", err)
	}
}

func TestCaseService_Update_RejectsInvalidBusinessState(t *testing.T) {
	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	wsOther, otherOwnerID := setupWorkspaceAndOwner(t, db)

	accountOther := createAccount(t, db, wsOther, otherOwnerID)
	svc := crm.NewCaseService(db)

	caseTicket, err := svc.Create(context.Background(), crm.CreateCaseInput{
		WorkspaceID: wsID,
		OwnerID:     ownerID,
		Subject:     "Case",
		Priority:    "medium",
		Status:      "open",
	})
	if err != nil {
		t.Fatalf("create case: %v", err)
	}

	_, err = svc.Update(context.Background(), wsID, caseTicket.ID, crm.UpdateCaseInput{
		OwnerID:  ownerID,
		Subject:  "Case",
		Priority: "impossible",
		Status:   "open",
	})
	if !errors.Is(err, crm.ErrInvalidCaseInput) {
		t.Fatalf("expected ErrInvalidCaseInput for invalid priority, got %v", err)
	}

	_, err = svc.Update(context.Background(), wsID, caseTicket.ID, crm.UpdateCaseInput{
		OwnerID:  ownerID,
		Subject:  "Case",
		Priority: "high",
		Status:   "teleported",
	})
	if !errors.Is(err, crm.ErrInvalidCaseInput) {
		t.Fatalf("expected ErrInvalidCaseInput for invalid status, got %v", err)
	}

	_, err = svc.Update(context.Background(), wsID, caseTicket.ID, crm.UpdateCaseInput{
		AccountID: accountOther,
		OwnerID:   ownerID,
		Subject:   "Case",
		Priority:  "high",
		Status:    "open",
	})
	if !errors.Is(err, crm.ErrInvalidCaseInput) {
		t.Fatalf("expected ErrInvalidCaseInput for cross-workspace account, got %v", err)
	}
}

func assertAuditCount(t *testing.T, db *sql.DB, workspaceID, action string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(`SELECT COUNT(*) FROM audit_event WHERE workspace_id = ? AND action = ?`, workspaceID, action).Scan(&got); err != nil {
		t.Fatalf("count audit %s: %v", action, err)
	}
	if got != want {
		t.Fatalf("audit count for %s = %d; want %d", action, got, want)
	}
}

func assertTimelineCount(t *testing.T, db *sql.DB, workspaceID, entityType, entityID, eventType string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(`SELECT COUNT(*) FROM timeline_event WHERE workspace_id = ? AND entity_type = ? AND entity_id = ? AND event_type = ?`, workspaceID, entityType, entityID, eventType).Scan(&got); err != nil {
		t.Fatalf("count timeline %s/%s/%s: %v", entityType, entityID, eventType, err)
	}
	if got != want {
		t.Fatalf("timeline count for %s/%s/%s = %d; want %d", entityType, entityID, eventType, got, want)
	}
}
