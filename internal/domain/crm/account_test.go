// Task 1.3.4: TDD tests for AccountService (written before implementation)
package crm_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
)

// TestAccountService_Create verifies creating an account inserts it into the DB.
func TestAccountService_Create(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	svc := crm.NewAccountService(db)

	// Prepare: create workspace and owner
	wsID, ownerID := setupWorkspaceAndOwner(t, db)

	// Act
	account, err := svc.Create(context.Background(), crm.CreateAccountInput{
		WorkspaceID: wsID,
		Name:        "Acme Corp",
		Domain:      "acme.com",
		Industry:    "Technology",
		SizeSegment: "mid",
		OwnerID:     ownerID,
		Address:     `{"city":"San Francisco","country":"USA"}`,
		Metadata:    `{"founded":2010}`,
	})

	// Assert
	if err != nil {
		t.Fatalf("Create() error = %v; want nil", err)
	}
	if account.ID == "" {
		t.Error("account.ID is empty; want non-empty UUID")
	}
	if account.Name != "Acme Corp" {
		t.Errorf("account.Name = %q; want %q", account.Name, "Acme Corp")
	}
	if account.DeletedAt != nil {
		t.Errorf("account.DeletedAt = %v; want nil for new account", account.DeletedAt)
	}
}

// TestAccountService_Get retrieves an account by ID.
func TestAccountService_Get(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	svc := crm.NewAccountService(db)

	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	created, _ := svc.Create(context.Background(), crm.CreateAccountInput{
		WorkspaceID: wsID,
		Name:        "Test Account",
		OwnerID:     ownerID,
	})

	// Act
	retrieved, err := svc.Get(context.Background(), wsID, created.ID)

	// Assert
	if err != nil {
		t.Fatalf("Get() error = %v; want nil", err)
	}
	if retrieved.ID != created.ID {
		t.Errorf("ID mismatch: got %q, want %q", retrieved.ID, created.ID)
	}
	if retrieved.Name != "Test Account" {
		t.Errorf("Name = %q; want %q", retrieved.Name, "Test Account")
	}
}

// TestAccountService_GetNotFound returns error when account doesn't exist.
func TestAccountService_GetNotFound(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	svc := crm.NewAccountService(db)

	wsID := createWorkspace(t, db)

	// Act
	_, err := svc.Get(context.Background(), wsID, "nonexistent-id")

	// Assert
	if err != sql.ErrNoRows {
		t.Errorf("Get(nonexistent) error = %v; want sql.ErrNoRows", err)
	}
}

// TestAccountService_List retrieves accounts with pagination.
func TestAccountService_List(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	svc := crm.NewAccountService(db)

	wsID, ownerID := setupWorkspaceAndOwner(t, db)

	// Create 3 accounts
	for i := 1; i <= 3; i++ {
		_, err := svc.Create(context.Background(), crm.CreateAccountInput{
			WorkspaceID: wsID,
			Name:        "Account " + string(rune('0'+byte(i))),
			OwnerID:     ownerID,
		})
		if err != nil {
			t.Fatalf("Create() error in seed account %d = %v; want nil", i, err)
		}
	}

	// Act: List with limit 2
	accounts, total, err := svc.List(context.Background(), wsID, crm.ListAccountsInput{
		Limit:  2,
		Offset: 0,
	})

	// Assert
	if err != nil {
		t.Fatalf("List() error = %v; want nil", err)
	}
	if len(accounts) != 2 {
		t.Errorf("List() returned %d accounts; want 2", len(accounts))
	}
	if total != 3 {
		t.Errorf("List() total = %d; want 3", total)
	}
}

// TestAccountService_ListExcludesDeleted verifies soft-deleted accounts are not returned.
func TestAccountService_ListExcludesDeleted(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	svc := crm.NewAccountService(db)

	wsID, ownerID := setupWorkspaceAndOwner(t, db)

	// Create 2 accounts
	acc1, _ := svc.Create(context.Background(), crm.CreateAccountInput{
		WorkspaceID: wsID,
		Name:        "Active Account",
		OwnerID:     ownerID,
	})
	acc2, _ := svc.Create(context.Background(), crm.CreateAccountInput{
		WorkspaceID: wsID,
		Name:        "Deleted Account",
		OwnerID:     ownerID,
	})

	// Delete one
	if err := svc.Delete(context.Background(), wsID, acc2.ID); err != nil {
		t.Fatalf("Delete() error = %v; want nil", err)
	}

	// Act: List should return only 1
	accounts, _, _ := svc.List(context.Background(), wsID, crm.ListAccountsInput{
		Limit:  10,
		Offset: 0,
	})

	// Assert
	if len(accounts) != 1 {
		t.Errorf("List() after delete = %d accounts; want 1", len(accounts))
	}
	if accounts[0].ID != acc1.ID {
		t.Errorf("Returned account ID = %q; want %q", accounts[0].ID, acc1.ID)
	}
}

// TestAccountService_Update modifies an account.
func TestAccountService_Update(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	svc := crm.NewAccountService(db)

	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	created, _ := svc.Create(context.Background(), crm.CreateAccountInput{
		WorkspaceID: wsID,
		Name:        "Old Name",
		OwnerID:     ownerID,
	})

	// Act
	updated, err := svc.Update(context.Background(), wsID, created.ID, crm.UpdateAccountInput{
		Name:     "New Name",
		Industry: "Finance",
		OwnerID:  ownerID, // Must pass valid FK
	})

	// Assert
	if err != nil {
		t.Fatalf("Update() error = %v; want nil", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("Updated name = %q; want %q", updated.Name, "New Name")
	}
	if updated.Industry == nil || *updated.Industry != "Finance" {
		t.Errorf("Updated industry = %v; want pointer to %q", updated.Industry, "Finance")
	}
}

// TestAccountService_Delete performs soft delete.
func TestAccountService_Delete(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	svc := crm.NewAccountService(db)

	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	created, _ := svc.Create(context.Background(), crm.CreateAccountInput{
		WorkspaceID: wsID,
		Name:        "To Delete",
		OwnerID:     ownerID,
	})

	// Act
	err := svc.Delete(context.Background(), wsID, created.ID)

	// Assert
	if err != nil {
		t.Fatalf("Delete() error = %v; want nil", err)
	}

	// Verify deleted_at is set
	_, err = svc.Get(context.Background(), wsID, created.ID)
	if err != sql.ErrNoRows {
		t.Errorf("After Delete(), Get() error = %v; want sql.ErrNoRows", err)
	}
}

// TestAccountService_ListByOwner returns accounts owned by a user.
func TestAccountService_ListByOwner(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	svc := crm.NewAccountService(db)

	wsID, owner1 := setupWorkspaceAndOwner(t, db)
	owner2 := createUser(t, db, wsID)

	// Create accounts: 2 for owner1, 1 for owner2
	if _, err := svc.Create(context.Background(), crm.CreateAccountInput{
		WorkspaceID: wsID,
		Name:        "Owner1 Account 1",
		OwnerID:     owner1,
	}); err != nil {
		t.Fatalf("Create() owner1 account 1 error = %v; want nil", err)
	}
	if _, err := svc.Create(context.Background(), crm.CreateAccountInput{
		WorkspaceID: wsID,
		Name:        "Owner1 Account 2",
		OwnerID:     owner1,
	}); err != nil {
		t.Fatalf("Create() owner1 account 2 error = %v; want nil", err)
	}
	if _, err := svc.Create(context.Background(), crm.CreateAccountInput{
		WorkspaceID: wsID,
		Name:        "Owner2 Account 1",
		OwnerID:     owner2,
	}); err != nil {
		t.Fatalf("Create() owner2 account 1 error = %v; want nil", err)
	}

	// Act
	accounts, err := svc.ListByOwner(context.Background(), wsID, owner1)

	// Assert
	if err != nil {
		t.Fatalf("ListByOwner() error = %v; want nil", err)
	}
	if len(accounts) != 2 {
		t.Errorf("ListByOwner() returned %d accounts; want 2", len(accounts))
	}
}

// --- helpers ---

// mustOpenDBWithMigrations opens an in-memory DB with migrations applied.
func mustOpenDBWithMigrations(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sqlite.NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB error = %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp error = %v", err)
	}

	return db
}

// createWorkspace creates a test workspace.
func createWorkspace(t *testing.T, db *sql.DB) string {
	t.Helper()
	id := "ws-" + randID()
	_, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES (?, ?, ?, datetime('now'), datetime('now'))
	`, id, "Test Workspace", "test-"+randID())
	if err != nil {
		t.Fatalf("createWorkspace error = %v", err)
	}
	return id
}

// createUser creates a test user in a workspace.
func createUser(t *testing.T, db *sql.DB, workspaceID string) string {
	t.Helper()
	id := "user-" + randID()
	_, err := db.Exec(`
		INSERT INTO user_account (id, workspace_id, email, display_name, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'active', datetime('now'), datetime('now'))
	`, id, workspaceID, "user-"+randID()+"@example.com", "Test User")
	if err != nil {
		t.Fatalf("createUser error = %v", err)
	}
	return id
}

// setupWorkspaceAndOwner creates both a workspace and an owner user.
func setupWorkspaceAndOwner(t *testing.T, db *sql.DB) (workspaceID, ownerID string) {
	t.Helper()
	wsID := createWorkspace(t, db)
	userID := createUser(t, db, wsID)
	return wsID, userID
}

// randID generates a unique random string for test IDs using time + counter.
var randIDCounter int64 = 0

func randID() string {
	randIDCounter++
	return time.Now().Format("20060102150405") + "-" + fmt.Sprintf("%d", randIDCounter)
}
