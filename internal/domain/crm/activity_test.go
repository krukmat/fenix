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
