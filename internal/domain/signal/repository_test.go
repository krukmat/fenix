package signal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	isqlite "github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	_ "modernc.org/sqlite"
)

func TestRepository_CreateAndGetByID(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)

	created, err := repo.Create(context.Background(), CreateInput{
		ID:          "sig-1",
		WorkspaceID: "ws_test",
		EntityType:  "lead",
		EntityID:    "lead-1",
		SignalType:  "intent_high",
		Confidence:  0.92,
		EvidenceIDs: []string{"ev-1", "ev-2"},
		SourceType:  "workflow",
		SourceID:    "wf-1",
		Metadata:    json.RawMessage(`{"reason":"scored"}`),
		Status:      StatusActive,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Status != StatusActive {
		t.Fatalf("status = %s, want %s", created.Status, StatusActive)
	}

	got, err := repo.GetByID(context.Background(), "ws_test", created.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.SignalType != "intent_high" {
		t.Fatalf("signal_type = %s, want intent_high", got.SignalType)
	}
	if len(got.EvidenceIDs) != 2 {
		t.Fatalf("len(evidence_ids) = %d, want 2", len(got.EvidenceIDs))
	}
}

func TestRepository_GetByID_NotFound(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)

	_, err := repo.GetByID(context.Background(), "ws_test", "missing")
	if !errors.Is(err, ErrSignalNotFound) {
		t.Fatalf("expected ErrSignalNotFound, got %v", err)
	}
}

func TestRepository_ListAndGetByEntity(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)

	inputs := []CreateInput{
		{ID: "sig-1", WorkspaceID: "ws_test", EntityType: "lead", EntityID: "lead-1", SignalType: "intent_high", Confidence: 0.9, EvidenceIDs: []string{"ev-1"}, SourceType: "workflow", SourceID: "wf-1", Status: StatusActive},
		{ID: "sig-2", WorkspaceID: "ws_test", EntityType: "lead", EntityID: "lead-1", SignalType: "intent_high", Confidence: 0.8, EvidenceIDs: []string{"ev-2"}, SourceType: "workflow", SourceID: "wf-1", Status: StatusDismissed},
		{ID: "sig-3", WorkspaceID: "ws_test", EntityType: "case", EntityID: "case-1", SignalType: "escalation_risk", Confidence: 0.7, EvidenceIDs: []string{"ev-3"}, SourceType: "agent_run", SourceID: "run-1", Status: StatusActive},
	}
	for _, input := range inputs {
		if _, err := repo.Create(context.Background(), input); err != nil {
			t.Fatalf("Create(%s) error = %v", input.ID, err)
		}
	}

	all, err := repo.List(context.Background(), "ws_test", Filters{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("len(all) = %d, want 3", len(all))
	}

	active := StatusActive
	activeList, err := repo.List(context.Background(), "ws_test", Filters{Status: &active})
	if err != nil {
		t.Fatalf("List(status) error = %v", err)
	}
	if len(activeList) != 2 {
		t.Fatalf("len(activeList) = %d, want 2", len(activeList))
	}

	entityList, err := repo.GetByEntity(context.Background(), "ws_test", "lead", "lead-1")
	if err != nil {
		t.Fatalf("GetByEntity() error = %v", err)
	}
	if len(entityList) != 2 {
		t.Fatalf("len(entityList) = %d, want 2", len(entityList))
	}
}

func TestRepository_Dismiss(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)

	_, err := repo.Create(context.Background(), CreateInput{
		ID:          "sig-dismiss",
		WorkspaceID: "ws_test",
		EntityType:  "lead",
		EntityID:    "lead-1",
		SignalType:  "intent_high",
		Confidence:  0.9,
		EvidenceIDs: []string{"ev-1"},
		SourceType:  "workflow",
		SourceID:    "wf-1",
		Status:      StatusActive,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	dismissed, err := repo.Dismiss(context.Background(), "ws_test", "sig-dismiss", "user-1")
	if err != nil {
		t.Fatalf("Dismiss() error = %v", err)
	}
	if dismissed.Status != StatusDismissed {
		t.Fatalf("status = %s, want %s", dismissed.Status, StatusDismissed)
	}
	if dismissed.DismissedBy == nil || *dismissed.DismissedBy != "user-1" {
		t.Fatalf("dismissed_by = %+v, want user-1", dismissed.DismissedBy)
	}
	if dismissed.DismissedAt == nil {
		t.Fatal("dismissed_at = nil, want timestamp")
	}
}

func TestRepository_Dismiss_NotFound(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)

	_, err := repo.Dismiss(context.Background(), "ws_test", "missing", "user-1")
	if !errors.Is(err, ErrSignalNotFound) {
		t.Fatalf("expected ErrSignalNotFound, got %v", err)
	}
}

func TestBuildCountActiveQuery(t *testing.T) {
	t.Parallel()
	cases := []struct {
		n    int
		want string
	}{
		{1, "SELECT entity_id, COUNT(*) FROM signal WHERE workspace_id = ? AND entity_type = ? AND status = ? AND entity_id IN (?) GROUP BY entity_id"},
		{3, "SELECT entity_id, COUNT(*) FROM signal WHERE workspace_id = ? AND entity_type = ? AND status = ? AND entity_id IN (?,?,?) GROUP BY entity_id"},
	}
	for _, tc := range cases {
		got := buildCountActiveQuery(tc.n)
		if got != tc.want {
			t.Errorf("buildCountActiveQuery(%d) = %q, want %q", tc.n, got, tc.want)
		}
	}
}

func TestRepository_CountActiveByEntities(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	for _, id := range []string{"sig-c1", "sig-c2"} {
		_, err := repo.Create(ctx, CreateInput{
			ID: id, WorkspaceID: "ws_c", EntityType: "deal",
			EntityID: "deal-c1", SignalType: "churn_risk", Confidence: 0.8,
			EvidenceIDs: []string{"ev-c"}, SourceType: "manual", SourceID: "m",
			Status: StatusActive,
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	counts, err := repo.CountActiveByEntities(ctx, "ws_c", "deal", []string{"deal-c1", "deal-c2"})
	if err != nil {
		t.Fatalf("CountActiveByEntities() error = %v", err)
	}
	if counts["deal-c1"] != 2 {
		t.Errorf("counts[deal-c1] = %d, want 2", counts["deal-c1"])
	}
	if counts["deal-c2"] != 0 {
		t.Errorf("counts[deal-c2] = %d, want 0", counts["deal-c2"])
	}

	empty, err := repo.CountActiveByEntities(ctx, "ws_c", "deal", nil)
	if err != nil {
		t.Fatalf("CountActiveByEntities(nil) error = %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected empty map for nil ids, got %v", empty)
	}
}

func TestRepository_Create_WithExpiresAt(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	repo := NewRepository(db)
	expiresAt := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)

	created, err := repo.Create(context.Background(), CreateInput{
		ID:          "sig-exp",
		WorkspaceID: "ws_test",
		EntityType:  "deal",
		EntityID:    "deal-1",
		SignalType:  "churn_risk",
		Confidence:  0.6,
		EvidenceIDs: []string{"ev-9"},
		SourceType:  "manual",
		SourceID:    "manual-1",
		Status:      StatusActive,
		ExpiresAt:   &expiresAt,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ExpiresAt == nil {
		t.Fatal("expires_at = nil, want timestamp")
	}
}

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	if err = isqlite.MigrateUp(db); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	if _, err = db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws_test', 'Signal Test', 'signal-test', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
