package knowledge

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/audit"
	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
)

func TestReindexService_RecordUpdatedCase_RefreshesKnowledge(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	wsID := createWorkspace(t, db)
	ownerID := createUserForReindex(t, db, wsID)

	bus := eventbus.New()
	ingest := NewIngestService(db, bus)
	auditSvc := audit.NewAuditService(db)
	reindex := NewReindexService(db, bus, ingest, auditSvc)

	now := time.Now().Format(time.RFC3339)
	caseID := newID()
	_, err := db.Exec(
		`INSERT INTO case_ticket (id, workspace_id, owner_id, subject, description, priority, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, 'medium', 'open', ?, ?)`,
		caseID, wsID, ownerID, "Cannot login", "old description", now, now,
	)
	if err != nil {
		t.Fatalf("seed case: %v", err)
	}

	entityType := EntityTypeCaseTicket
	entityID := caseID
	_, err = ingest.Ingest(context.Background(), CreateKnowledgeItemInput{
		WorkspaceID: wsID,
		SourceType:  SourceTypeCase,
		Title:       "Cannot login",
		RawContent:  "Subject: Cannot login\nDescription: old description",
		EntityType:  &entityType,
		EntityID:    &entityID,
	})
	if err != nil {
		t.Fatalf("seed ingest: %v", err)
	}

	_, err = db.Exec(
		`UPDATE case_ticket SET description = ?, priority = ?, status = ?, updated_at = ? WHERE id = ? AND workspace_id = ?`,
		"new description from support", "high", "in_progress", time.Now().Format(time.RFC3339), caseID, wsID,
	)
	if err != nil {
		t.Fatalf("update case: %v", err)
	}

	if err := reindex.HandleRecordChange(context.Background(), RecordChangedEvent{
		EntityType:  EntityTypeCaseTicket,
		EntityID:    caseID,
		WorkspaceID: wsID,
		ChangeType:  ChangeTypeUpdated,
		OccurredAt:  time.Now(),
	}); err != nil {
		t.Fatalf("handle reindex: %v", err)
	}

	var raw string
	if err := db.QueryRow(`SELECT raw_content FROM knowledge_item WHERE workspace_id = ? AND entity_type = ? AND entity_id = ?`, wsID, EntityTypeCaseTicket, caseID).Scan(&raw); err != nil {
		t.Fatalf("query refreshed knowledge item: %v", err)
	}
	if raw == "" || raw == "Subject: Cannot login\nDescription: old description" {
		t.Fatalf("expected refreshed content, got: %q", raw)
	}

	var auditCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM audit_event WHERE workspace_id = ? AND action = 'knowledge.reindex'`, wsID).Scan(&auditCount); err != nil {
		t.Fatalf("query audit: %v", err)
	}
	if auditCount == 0 {
		t.Fatal("expected at least one audit event for knowledge.reindex")
	}
}

func TestReindexService_RecordDeletedAccount_SoftDeletesKnowledge(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	wsID := createWorkspace(t, db)
	ownerID := createUserForReindex(t, db, wsID)

	bus := eventbus.New()
	ingest := NewIngestService(db, bus)
	reindex := NewReindexService(db, bus, ingest, audit.NewAuditService(db))

	accountID := newID()
	now := time.Now().Format(time.RFC3339)
	_, err := db.Exec(
		`INSERT INTO account (id, workspace_id, name, domain, owner_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		accountID, wsID, "Acme", "acme.com", ownerID, now, now,
	)
	if err != nil {
		t.Fatalf("seed account: %v", err)
	}

	entityType := EntityTypeAccount
	entityID := accountID
	_, err = ingest.Ingest(context.Background(), CreateKnowledgeItemInput{
		WorkspaceID: wsID,
		SourceType:  SourceTypeDocument,
		Title:       "Acme",
		RawContent:  "Name: Acme",
		EntityType:  &entityType,
		EntityID:    &entityID,
	})
	if err != nil {
		t.Fatalf("seed ingest: %v", err)
	}

	_, err = db.Exec(`UPDATE account SET deleted_at = ?, updated_at = ? WHERE id = ? AND workspace_id = ?`, now, now, accountID, wsID)
	if err != nil {
		t.Fatalf("soft delete account: %v", err)
	}

	if err := reindex.HandleRecordChange(context.Background(), RecordChangedEvent{
		EntityType:  EntityTypeAccount,
		EntityID:    accountID,
		WorkspaceID: wsID,
		ChangeType:  ChangeTypeDeleted,
		OccurredAt:  time.Now(),
	}); err != nil {
		t.Fatalf("handle delete reindex: %v", err)
	}

	var deletedAt time.Time
	if err := db.QueryRow(`SELECT deleted_at FROM knowledge_item WHERE workspace_id = ? AND entity_type = ? AND entity_id = ?`, wsID, EntityTypeAccount, accountID).Scan(&deletedAt); err != nil {
		t.Fatalf("query deleted_at: %v", err)
	}
}

func createUserForReindex(t *testing.T, db *sql.DB, workspaceID string) string {
	t.Helper()
	id := newID()
	_, err := db.Exec(
		`INSERT INTO user_account (id, workspace_id, email, display_name, status, created_at, updated_at) VALUES (?, ?, ?, ?, 'active', ?, ?)`,
		id,
		workspaceID,
		"reindex-"+id+"@example.com",
		"Reindex User",
		time.Now().Format(time.RFC3339),
		time.Now().Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("create user for reindex: %v", err)
	}
	return id
}
