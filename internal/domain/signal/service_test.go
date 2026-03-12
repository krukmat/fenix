package signal

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
	isqlite "github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	_ "modernc.org/sqlite"
)

func TestService_Create_Succeeds(t *testing.T) {
	t.Parallel()

	db := setupServiceDB(t)
	svc := NewService(db)

	out, err := svc.Create(context.Background(), CreateSignalInput{
		WorkspaceID: "ws_test",
		EntityType:  "lead",
		EntityID:    "lead_1",
		SignalType:  "intent_high",
		Confidence:  0.92,
		EvidenceIDs: []string{"ev-1"},
		SourceType:  "workflow",
		SourceID:    "wf-1",
		Metadata:    map[string]any{"reason": "score"},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if out.Status != StatusActive {
		t.Fatalf("status = %s, want %s", out.Status, StatusActive)
	}
}

func TestService_Create_RejectsMissingEvidence(t *testing.T) {
	t.Parallel()

	db := setupServiceDB(t)
	svc := NewService(db)

	_, err := svc.Create(context.Background(), CreateSignalInput{
		WorkspaceID: "ws_test",
		EntityType:  "lead",
		EntityID:    "lead_1",
		SignalType:  "intent_high",
		Confidence:  0.92,
		SourceType:  "workflow",
		SourceID:    "wf-1",
	})
	if !errors.Is(err, ErrInvalidSignalInput) {
		t.Fatalf("expected ErrInvalidSignalInput, got %v", err)
	}
}

func TestService_Create_RejectsConfidenceOutOfRange(t *testing.T) {
	t.Parallel()

	db := setupServiceDB(t)
	svc := NewService(db)

	_, err := svc.Create(context.Background(), CreateSignalInput{
		WorkspaceID: "ws_test",
		EntityType:  "lead",
		EntityID:    "lead_1",
		SignalType:  "intent_high",
		Confidence:  1.5,
		EvidenceIDs: []string{"ev-1"},
		SourceType:  "workflow",
		SourceID:    "wf-1",
	})
	if !errors.Is(err, ErrInvalidSignalInput) {
		t.Fatalf("expected ErrInvalidSignalInput, got %v", err)
	}
}

func TestService_Create_RejectsInvalidEntity(t *testing.T) {
	t.Parallel()

	db := setupServiceDB(t)
	svc := NewService(db)

	_, err := svc.Create(context.Background(), CreateSignalInput{
		WorkspaceID: "ws_test",
		EntityType:  "lead",
		EntityID:    "missing",
		SignalType:  "intent_high",
		Confidence:  0.9,
		EvidenceIDs: []string{"ev-1"},
		SourceType:  "workflow",
		SourceID:    "wf-1",
	})
	if !errors.Is(err, ErrInvalidSignalInput) {
		t.Fatalf("expected ErrInvalidSignalInput, got %v", err)
	}
}

func TestService_ListAndGetByEntity(t *testing.T) {
	t.Parallel()

	db := setupServiceDB(t)
	repo := NewRepository(db)
	svc := NewServiceWithRepository(db, repo)

	for _, input := range []CreateInput{
		{ID: "sig-1", WorkspaceID: "ws_test", EntityType: "lead", EntityID: "lead_1", SignalType: "intent_high", Confidence: 0.9, EvidenceIDs: []string{"ev-1"}, SourceType: "workflow", SourceID: "wf-1", Status: StatusActive},
		{ID: "sig-2", WorkspaceID: "ws_test", EntityType: "lead", EntityID: "lead_1", SignalType: "upsell_opportunity", Confidence: 0.7, EvidenceIDs: []string{"ev-2"}, SourceType: "workflow", SourceID: "wf-1", Status: StatusDismissed},
	} {
		if _, err := repo.Create(context.Background(), input); err != nil {
			t.Fatalf("repo.Create(%s) error = %v", input.ID, err)
		}
	}

	all, err := svc.List(context.Background(), "ws_test", Filters{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("len(all) = %d, want 2", len(all))
	}

	entity, err := svc.GetByEntity(context.Background(), "ws_test", "lead", "lead_1")
	if err != nil {
		t.Fatalf("GetByEntity() error = %v", err)
	}
	if len(entity) != 2 {
		t.Fatalf("len(entity) = %d, want 2", len(entity))
	}
}

func TestService_Dismiss_Succeeds(t *testing.T) {
	t.Parallel()

	db := setupServiceDB(t)
	svc := NewService(db)

	created, err := svc.Create(context.Background(), CreateSignalInput{
		WorkspaceID: "ws_test",
		EntityType:  "lead",
		EntityID:    "lead_1",
		SignalType:  "intent_high",
		Confidence:  0.92,
		EvidenceIDs: []string{"ev-1"},
		SourceType:  "workflow",
		SourceID:    "wf-1",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := svc.Dismiss(context.Background(), "ws_test", created.ID, "user-1"); err != nil {
		t.Fatalf("Dismiss() error = %v", err)
	}

	got, err := svc.repo.GetByID(context.Background(), "ws_test", created.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Status != StatusDismissed {
		t.Fatalf("status = %s, want %s", got.Status, StatusDismissed)
	}
}

func TestService_Dismiss_RejectsNonActive(t *testing.T) {
	t.Parallel()

	db := setupServiceDB(t)
	repo := NewRepository(db)
	svc := NewServiceWithRepository(db, repo)

	_, err := repo.Create(context.Background(), CreateInput{
		ID:          "sig-1",
		WorkspaceID: "ws_test",
		EntityType:  "lead",
		EntityID:    "lead_1",
		SignalType:  "intent_high",
		Confidence:  0.9,
		EvidenceIDs: []string{"ev-1"},
		SourceType:  "workflow",
		SourceID:    "wf-1",
		Status:      StatusDismissed,
	})
	if err != nil {
		t.Fatalf("repo.Create() error = %v", err)
	}

	err = svc.Dismiss(context.Background(), "ws_test", "sig-1", "user-1")
	if !errors.Is(err, ErrSignalDismissInvalid) {
		t.Fatalf("expected ErrSignalDismissInvalid, got %v", err)
	}
}

func TestService_Create_PublishesEvent(t *testing.T) {
	t.Parallel()

	db := setupServiceDB(t)
	bus := eventbus.New()
	repo := NewRepository(db)
	svc := NewServiceWithBus(db, repo, bus)
	ch := bus.Subscribe(TopicSignalCreated)

	created, err := svc.Create(context.Background(), CreateSignalInput{
		WorkspaceID: "ws_test",
		EntityType:  "lead",
		EntityID:    "lead_1",
		SignalType:  "intent_high",
		Confidence:  0.92,
		EvidenceIDs: []string{"ev-1"},
		SourceType:  "workflow",
		SourceID:    "wf-1",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	select {
	case evt := <-ch:
		payload, ok := evt.Payload.(CreatedEventPayload)
		if !ok {
			t.Fatalf("payload type = %T, want CreatedEventPayload", evt.Payload)
		}
		if payload.SignalID != created.ID {
			t.Fatalf("payload.SignalID = %s, want %s", payload.SignalID, created.ID)
		}
		if payload.Status != StatusActive {
			t.Fatalf("payload.Status = %s, want %s", payload.Status, StatusActive)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for signal.created event")
	}
}

func TestService_Dismiss_PublishesEvent(t *testing.T) {
	t.Parallel()

	db := setupServiceDB(t)
	bus := eventbus.New()
	repo := NewRepository(db)
	svc := NewServiceWithBus(db, repo, bus)
	ch := bus.Subscribe(TopicSignalDismissed)

	created, err := svc.Create(context.Background(), CreateSignalInput{
		WorkspaceID: "ws_test",
		EntityType:  "lead",
		EntityID:    "lead_1",
		SignalType:  "intent_high",
		Confidence:  0.92,
		EvidenceIDs: []string{"ev-1"},
		SourceType:  "workflow",
		SourceID:    "wf-1",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := svc.Dismiss(context.Background(), "ws_test", created.ID, "user-1"); err != nil {
		t.Fatalf("Dismiss() error = %v", err)
	}

	select {
	case evt := <-ch:
		payload, ok := evt.Payload.(DismissedEventPayload)
		if !ok {
			t.Fatalf("payload type = %T, want DismissedEventPayload", evt.Payload)
		}
		if payload.SignalID != created.ID {
			t.Fatalf("payload.SignalID = %s, want %s", payload.SignalID, created.ID)
		}
		if payload.DismissedBy != "user-1" {
			t.Fatalf("payload.DismissedBy = %s, want user-1", payload.DismissedBy)
		}
		if payload.Status != StatusDismissed {
			t.Fatalf("payload.Status = %s, want %s", payload.Status, StatusDismissed)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for signal.dismissed event")
	}
}

func TestService_Create_RejectsMissingSourceFields(t *testing.T) {
	t.Parallel()

	db := setupServiceDB(t)
	svc := NewService(db)

	base := CreateSignalInput{
		WorkspaceID: "ws_test",
		EntityType:  "lead",
		EntityID:    "lead_1",
		SignalType:  "intent_high",
		Confidence:  0.9,
		EvidenceIDs: []string{"ev-1"},
		SourceType:  "workflow",
		SourceID:    "wf-1",
	}

	for _, tc := range []struct {
		name   string
		mutate func(*CreateSignalInput)
	}{
		{"empty signal_type", func(i *CreateSignalInput) { i.SignalType = "" }},
		{"empty source_type", func(i *CreateSignalInput) { i.SourceType = "" }},
		{"empty source_id", func(i *CreateSignalInput) { i.SourceID = "" }},
		{"empty entity_id", func(i *CreateSignalInput) { i.EntityID = "" }},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			input := base
			tc.mutate(&input)
			_, err := svc.Create(context.Background(), input)
			if !errors.Is(err, ErrInvalidSignalInput) {
				t.Fatalf("%s: expected ErrInvalidSignalInput, got %v", tc.name, err)
			}
		})
	}
}

func TestService_Create_SucceedsForContactDealCase(t *testing.T) {
	t.Parallel()

	db := setupServiceDB(t)
	svc := NewService(db)

	for _, tc := range []struct {
		entityType string
		entityID   string
	}{
		{"contact", "contact_1"},
		{"deal", "deal_1"},
		{"case", "case_1"},
	} {
		tc := tc
		t.Run(tc.entityType, func(t *testing.T) {
			t.Parallel()
			_, err := svc.Create(context.Background(), CreateSignalInput{
				WorkspaceID: "ws_test",
				EntityType:  tc.entityType,
				EntityID:    tc.entityID,
				SignalType:  "test_signal",
				Confidence:  0.8,
				EvidenceIDs: []string{"ev-1"},
				SourceType:  "workflow",
				SourceID:    "wf-1",
			})
			if err != nil {
				t.Fatalf("Create(%s) error = %v", tc.entityType, err)
			}
		})
	}
}

func TestService_Dismiss_RejectsMissingActorID(t *testing.T) {
	t.Parallel()

	db := setupServiceDB(t)
	svc := NewService(db)

	err := svc.Dismiss(context.Background(), "ws_test", "sig-1", "")
	if !errors.Is(err, ErrInvalidSignalInput) {
		t.Fatalf("expected ErrInvalidSignalInput, got %v", err)
	}
}

func setupServiceDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	// IMPORTANT: :memory: databases are per-connection in SQLite.
	// Without this, the pool may open a second connection that sees an empty DB.
	db.SetMaxOpenConns(1)
	if err = isqlite.MigrateUp(db); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	mustExec(t, db, `INSERT INTO workspace (id, name, slug, created_at, updated_at) VALUES ('ws_test', 'Signal Service Test', 'signal-service-test', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExec(t, db, `INSERT INTO user_account (id, workspace_id, email, display_name, status, created_at, updated_at) VALUES ('owner_1', 'ws_test', 'owner@example.com', 'Owner', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExec(t, db, `INSERT INTO account (id, workspace_id, name, owner_id, created_at, updated_at) VALUES ('account_1', 'ws_test', 'Acme', 'owner_1', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExec(t, db, `INSERT INTO contact (id, workspace_id, account_id, first_name, last_name, email, status, owner_id, created_at, updated_at) VALUES ('contact_1', 'ws_test', 'account_1', 'Ada', 'Lovelace', 'ada@example.com', 'active', 'owner_1', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExec(t, db, `INSERT INTO lead (id, workspace_id, contact_id, account_id, source, status, owner_id, created_at, updated_at) VALUES ('lead_1', 'ws_test', 'contact_1', 'account_1', 'email', 'new', 'owner_1', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExec(t, db, `INSERT INTO pipeline (id, workspace_id, name, entity_type, created_at, updated_at) VALUES ('pipe_1', 'ws_test', 'Sales', 'deal', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExec(t, db, `INSERT INTO pipeline_stage (id, pipeline_id, name, position, probability, created_at, updated_at) VALUES ('stage_1', 'pipe_1', 'Open', 1, 0.5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExec(t, db, `INSERT INTO deal (id, workspace_id, account_id, pipeline_id, stage_id, owner_id, title, status, created_at, updated_at) VALUES ('deal_1', 'ws_test', 'account_1', 'pipe_1', 'stage_1', 'owner_1', 'Deal', 'open', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExec(t, db, `INSERT INTO case_ticket (id, workspace_id, owner_id, subject, priority, status, created_at, updated_at) VALUES ('case_1', 'ws_test', 'owner_1', 'Case', 'high', 'open', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func mustExec(t *testing.T, db *sql.DB, query string) {
	t.Helper()
	if _, err := db.Exec(query); err != nil {
		t.Fatalf("exec failed: %v", err)
	}
}
