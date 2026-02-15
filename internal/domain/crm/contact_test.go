// Task 1.4: TDD tests for ContactService
// Traces: FR-001
package crm_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
)

func TestContactService_Create(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	svc := crm.NewContactService(db)

	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	accountID := createAccount(t, db, wsID, ownerID)

	contact, err := svc.Create(context.Background(), crm.CreateContactInput{
		WorkspaceID: wsID,
		AccountID:   accountID,
		FirstName:   "Ada",
		LastName:    "Lovelace",
		Email:       "ada@example.com",
		Phone:       "+34123456789",
		Title:       "CTO",
		OwnerID:     ownerID,
	})

	if err != nil {
		t.Fatalf("Create() error = %v; want nil", err)
	}
	if contact.ID == "" {
		t.Error("contact.ID is empty; want non-empty UUID")
	}
	if contact.Status != "active" {
		t.Errorf("contact.Status = %q; want %q", contact.Status, "active")
	}
}

func TestContactService_GetNotFound(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	svc := crm.NewContactService(db)

	wsID := createWorkspace(t, db)

	_, err := svc.Get(context.Background(), wsID, "nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("Get(nonexistent) error = %v; want sql.ErrNoRows", err)
	}
}

func TestContactService_List(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	svc := crm.NewContactService(db)

	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	accountID := createAccount(t, db, wsID, ownerID)

	for i := 0; i < 3; i++ {
		_, _ = svc.Create(context.Background(), crm.CreateContactInput{
			WorkspaceID: wsID,
			AccountID:   accountID,
			FirstName:   "Name",
			LastName:    "Surname",
			OwnerID:     ownerID,
		})
	}

	contacts, total, err := svc.List(context.Background(), wsID, crm.ListContactsInput{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatalf("List() error = %v; want nil", err)
	}
	if len(contacts) != 2 {
		t.Errorf("List() returned %d contacts; want 2", len(contacts))
	}
	if total != 3 {
		t.Errorf("List() total = %d; want 3", total)
	}
}

func TestContactService_ListByAccount(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	svc := crm.NewContactService(db)

	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	accountA := createAccount(t, db, wsID, ownerID)
	accountB := createAccount(t, db, wsID, ownerID)

	_, _ = svc.Create(context.Background(), crm.CreateContactInput{
		WorkspaceID: wsID,
		AccountID:   accountA,
		FirstName:   "A1",
		LastName:    "Contact",
		OwnerID:     ownerID,
	})
	_, _ = svc.Create(context.Background(), crm.CreateContactInput{
		WorkspaceID: wsID,
		AccountID:   accountA,
		FirstName:   "A2",
		LastName:    "Contact",
		OwnerID:     ownerID,
	})
	_, _ = svc.Create(context.Background(), crm.CreateContactInput{
		WorkspaceID: wsID,
		AccountID:   accountB,
		FirstName:   "B1",
		LastName:    "Contact",
		OwnerID:     ownerID,
	})

	contacts, err := svc.ListByAccount(context.Background(), wsID, accountA)
	if err != nil {
		t.Fatalf("ListByAccount() error = %v; want nil", err)
	}
	if len(contacts) != 2 {
		t.Errorf("ListByAccount() returned %d contacts; want 2", len(contacts))
	}
}

func TestContactService_Update(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	svc := crm.NewContactService(db)

	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	accountID := createAccount(t, db, wsID, ownerID)

	created, _ := svc.Create(context.Background(), crm.CreateContactInput{
		WorkspaceID: wsID,
		AccountID:   accountID,
		FirstName:   "Old",
		LastName:    "Name",
		OwnerID:     ownerID,
	})

	updated, err := svc.Update(context.Background(), wsID, created.ID, crm.UpdateContactInput{
		AccountID: accountID,
		FirstName: "New",
		LastName:  "Name",
		Status:    "inactive",
		OwnerID:   ownerID,
	})
	if err != nil {
		t.Fatalf("Update() error = %v; want nil", err)
	}
	if updated.FirstName != "New" {
		t.Errorf("updated.FirstName = %q; want %q", updated.FirstName, "New")
	}
	if updated.Status != "inactive" {
		t.Errorf("updated.Status = %q; want %q", updated.Status, "inactive")
	}
}

func TestContactService_Delete(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	svc := crm.NewContactService(db)

	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	accountID := createAccount(t, db, wsID, ownerID)

	created, _ := svc.Create(context.Background(), crm.CreateContactInput{
		WorkspaceID: wsID,
		AccountID:   accountID,
		FirstName:   "To",
		LastName:    "Delete",
		OwnerID:     ownerID,
	})

	err := svc.Delete(context.Background(), wsID, created.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v; want nil", err)
	}

	_, err = svc.Get(context.Background(), wsID, created.ID)
	if err != sql.ErrNoRows {
		t.Errorf("After Delete(), Get() error = %v; want sql.ErrNoRows", err)
	}
}

func createAccount(t *testing.T, db *sql.DB, workspaceID, ownerID string) string {
	t.Helper()
	id := "acc-" + randID()
	_, err := db.Exec(`
		INSERT INTO account (id, workspace_id, name, owner_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))
	`, id, workspaceID, "Account "+id, ownerID)
	if err != nil {
		t.Fatalf("createAccount error = %v", err)
	}
	return id
}
