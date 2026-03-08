// Traces: FR-001
package crm_test

import (
	"context"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
)

func TestAttachmentService_Create_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	svc := crm.NewAttachmentService(db)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	size := int64(123)

	att, err := svc.Create(context.Background(), crm.CreateAttachmentInput{
		WorkspaceID: wsID,
		EntityType:  "account",
		EntityID:    "acc-1",
		UploaderID:  ownerID,
		Filename:    "doc.txt",
		ContentType: "text/plain",
		SizeBytes:   &size,
		StoragePath: "/tmp/doc.txt",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if att.Filename != "doc.txt" {
		t.Fatalf("expected filename doc.txt, got %q", att.Filename)
	}
}

func TestAttachmentService_GetAndList_WithSeededRows(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	svc := crm.NewAttachmentService(db)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := db.Exec(`
		INSERT INTO attachment (
			id, workspace_id, entity_type, entity_id, uploader_id,
			filename, storage_path, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, "att-1", wsID, "account", "acc-1", ownerID, "seeded.txt", "/tmp/seeded.txt", now)
	if err != nil {
		t.Fatalf("seed attachment insert error = %v", err)
	}

	got, err := svc.Get(context.Background(), wsID, "att-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Filename != "seeded.txt" {
		t.Fatalf("expected filename seeded.txt, got %q", got.Filename)
	}

	list, total, err := svc.List(context.Background(), wsID, crm.ListAttachmentsInput{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total < 1 || len(list) < 1 {
		t.Fatalf("expected attachments, got total=%d len=%d", total, len(list))
	}
}

func TestAttachmentService_Delete_Existing_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	svc := crm.NewAttachmentService(db)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := db.Exec(`
		INSERT INTO attachment (
			id, workspace_id, entity_type, entity_id, uploader_id,
			filename, storage_path, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, "att-del-1", wsID, "account", "acc-1", ownerID, "to-delete.txt", "/tmp/to-delete.txt", now)
	if err != nil {
		t.Fatalf("seed attachment insert error = %v", err)
	}

	if err = svc.Delete(context.Background(), wsID, "att-del-1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, getErr := svc.Get(context.Background(), wsID, "att-del-1")
	if getErr == nil {
		t.Fatalf("expected attachment to be deleted")
	}
}

func TestAttachmentService_Delete_Missing_SucceedsWithoutTimeline(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	svc := crm.NewAttachmentService(db)
	wsID, _ := setupWorkspaceAndOwner(t, db)

	err := svc.Delete(context.Background(), wsID, "att-missing")
	if err != nil {
		t.Fatalf("expected delete missing to succeed, got %v", err)
	}
}
