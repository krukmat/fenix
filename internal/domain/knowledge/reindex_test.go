package knowledge

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/audit"
	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
)

func TestTopicForChangeType_Mapping(t *testing.T) {
	t.Parallel()

	if got := TopicForChangeType(ChangeTypeCreated); got != TopicRecordCreated {
		t.Fatalf("expected %s, got %s", TopicRecordCreated, got)
	}
	if got := TopicForChangeType(ChangeTypeDeleted); got != TopicRecordDeleted {
		t.Fatalf("expected %s, got %s", TopicRecordDeleted, got)
	}
	if got := TopicForChangeType(ChangeType("unexpected")); got != TopicRecordUpdated {
		t.Fatalf("expected default %s, got %s", TopicRecordUpdated, got)
	}
}

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

func TestReindexService_RecordUpdatedAccount_RefreshesKnowledge(t *testing.T) {
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
		`INSERT INTO account (id, workspace_id, name, domain, industry, owner_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		accountID, wsID, "Acme", "acme.com", "SaaS", ownerID, now, now,
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
		RawContent:  "Name: Acme\nDomain: acme.com\nIndustry: SaaS",
		EntityType:  &entityType,
		EntityID:    &entityID,
	})
	if err != nil {
		t.Fatalf("seed ingest: %v", err)
	}

	_, err = db.Exec(
		`UPDATE account SET domain = ?, industry = ?, updated_at = ? WHERE id = ? AND workspace_id = ?`,
		"acme.ai", "AI", time.Now().Format(time.RFC3339), accountID, wsID,
	)
	if err != nil {
		t.Fatalf("update account: %v", err)
	}

	if err := reindex.HandleRecordChange(context.Background(), RecordChangedEvent{
		EntityType:  EntityTypeAccount,
		EntityID:    accountID,
		WorkspaceID: wsID,
		ChangeType:  ChangeTypeUpdated,
		OccurredAt:  time.Now(),
	}); err != nil {
		t.Fatalf("handle account reindex: %v", err)
	}

	var raw string
	if err := db.QueryRow(`SELECT raw_content FROM knowledge_item WHERE workspace_id = ? AND entity_type = ? AND entity_id = ?`, wsID, EntityTypeAccount, accountID).Scan(&raw); err != nil {
		t.Fatalf("query refreshed account knowledge item: %v", err)
	}
	if !strings.Contains(raw, "Domain: acme.ai") || !strings.Contains(raw, "Industry: AI") {
		t.Fatalf("expected refreshed account content, got: %q", raw)
	}
}

func TestReindexService_QueueWorkspaceReindex_EntityFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	wsID := createWorkspace(t, db)
	ownerID := createUserForReindex(t, db, wsID)

	bus := eventbus.New()
	ingest := NewIngestService(db, bus)
	reindex := NewReindexService(db, bus, ingest, audit.NewAuditService(db))

	now := time.Now().Format(time.RFC3339)
	accountID := newID()
	_, err := db.Exec(
		`INSERT INTO account (id, workspace_id, name, domain, owner_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		accountID, wsID, "Acme", "acme.com", ownerID, now, now,
	)
	if err != nil {
		t.Fatalf("seed account: %v", err)
	}

	caseID := newID()
	_, err = db.Exec(
		`INSERT INTO case_ticket (id, workspace_id, owner_id, subject, description, priority, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, 'medium', 'open', ?, ?)`,
		caseID, wsID, ownerID, "Case A", "description", now, now,
	)
	if err != nil {
		t.Fatalf("seed case: %v", err)
	}

	accType := EntityTypeAccount
	accID := accountID
	if _, err := ingest.Ingest(context.Background(), CreateKnowledgeItemInput{
		WorkspaceID: wsID,
		SourceType:  SourceTypeDocument,
		Title:       "Acme",
		RawContent:  "Name: Acme",
		EntityType:  &accType,
		EntityID:    &accID,
	}); err != nil {
		t.Fatalf("ingest account knowledge: %v", err)
	}

	caseType := EntityTypeCaseTicket
	caseEntityID := caseID
	if _, err := ingest.Ingest(context.Background(), CreateKnowledgeItemInput{
		WorkspaceID: wsID,
		SourceType:  SourceTypeCase,
		Title:       "Case A",
		RawContent:  "Subject: Case A",
		EntityType:  &caseType,
		EntityID:    &caseEntityID,
	}); err != nil {
		t.Fatalf("ingest case knowledge: %v", err)
	}

	sub := bus.Subscribe(TopicRecordUpdated)

	filter := EntityTypeAccount
	queued, err := reindex.QueueWorkspaceReindex(context.Background(), wsID, &filter)
	if err != nil {
		t.Fatalf("queue reindex: %v", err)
	}
	if queued != 1 {
		t.Fatalf("expected 1 queued event, got %d", queued)
	}

	select {
	case evt := <-sub:
		record, ok := evt.Payload.(RecordChangedEvent)
		if !ok {
			t.Fatalf("unexpected payload type %T", evt.Payload)
		}
		if record.EntityType != EntityTypeAccount {
			t.Fatalf("expected account entity event, got %s", record.EntityType)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected a queued record.updated event")
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
