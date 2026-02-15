// Traces: FR-001
package crm_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
)

func TestActivityService_Create_ReturnsTimelineConstraintError(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	svc := crm.NewActivityService(db)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)

	_, err := svc.Create(context.Background(), crm.CreateActivityInput{
		WorkspaceID:  wsID,
		ActivityType: "task",
		EntityType:   "account",
		EntityID:     "acc-1",
		OwnerID:      ownerID,
		Subject:      "Call customer",
	})
	if err == nil {
		t.Fatalf("expected timeline constraint error, got nil")
	}
	if !strings.Contains(err.Error(), "create activity timeline") {
		t.Fatalf("expected create activity timeline error, got %v", err)
	}
	if !strings.Contains(err.Error(), "CHECK constraint failed") {
		t.Fatalf("expected CHECK constraint failed, got %v", err)
	}
}

func TestActivityService_GetAndList_WithSeededRows(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	svc := crm.NewActivityService(db)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := db.Exec(`
		INSERT INTO activity (
			id, workspace_id, activity_type, entity_type, entity_id, owner_id,
			subject, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "act-1", wsID, "task", "account", "acc-1", ownerID, "Seeded activity", "pending", now, now)
	if err != nil {
		t.Fatalf("seed activity insert error = %v", err)
	}

	got, err := svc.Get(context.Background(), wsID, "act-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Subject != "Seeded activity" {
		t.Fatalf("expected subject %q, got %q", "Seeded activity", got.Subject)
	}

	list, total, err := svc.List(context.Background(), wsID, crm.ListActivitiesInput{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total < 1 || len(list) < 1 {
		t.Fatalf("expected activities, got total=%d len=%d", total, len(list))
	}
}

func TestActivityService_Update_ReturnsTimelineConstraintErrorButPersistsUpdate(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	svc := crm.NewActivityService(db)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := db.Exec(`
		INSERT INTO activity (
			id, workspace_id, activity_type, entity_type, entity_id, owner_id,
			subject, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "act-upd-1", wsID, "task", "account", "acc-1", ownerID, "Before update", "pending", now, now)
	if err != nil {
		t.Fatalf("seed activity insert error = %v", err)
	}

	_, err = svc.Update(context.Background(), wsID, "act-upd-1", crm.UpdateActivityInput{
		ActivityType: "task",
		EntityType:   "account",
		EntityID:     "acc-1",
		OwnerID:      ownerID,
		Subject:      "After update",
		Status:       "completed",
	})
	if err == nil {
		t.Fatalf("expected update timeline error, got nil")
	}
	if !strings.Contains(err.Error(), "update activity timeline") {
		t.Fatalf("expected update activity timeline error, got %v", err)
	}

	got, getErr := svc.Get(context.Background(), wsID, "act-upd-1")
	if getErr != nil {
		t.Fatalf("Get() after failed update timeline error = %v", getErr)
	}
	if got.Subject != "After update" {
		t.Fatalf("expected subject to be updated, got %q", got.Subject)
	}
	if got.Status != "completed" {
		t.Fatalf("expected status completed, got %q", got.Status)
	}
}

func TestActivityService_Delete_ReturnsTimelineConstraintErrorButDeletesRow(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	svc := crm.NewActivityService(db)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := db.Exec(`
		INSERT INTO activity (
			id, workspace_id, activity_type, entity_type, entity_id, owner_id,
			subject, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "act-del-1", wsID, "task", "account", "acc-1", ownerID, "To delete", "pending", now, now)
	if err != nil {
		t.Fatalf("seed activity insert error = %v", err)
	}

	err = svc.Delete(context.Background(), wsID, "act-del-1")
	if err == nil {
		t.Fatalf("expected delete timeline error, got nil")
	}
	if !strings.Contains(err.Error(), "delete activity timeline") {
		t.Fatalf("expected delete activity timeline error, got %v", err)
	}

	_, getErr := svc.Get(context.Background(), wsID, "act-del-1")
	if getErr == nil {
		t.Fatalf("expected row to be deleted")
	}
}
