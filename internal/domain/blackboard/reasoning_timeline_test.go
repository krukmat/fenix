// Tests for ReasoningTimeline — append-only read path over reasoning_event (Task A.4, ADR-100).
package blackboard_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/blackboard"
	isqlite "github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
)

// setupTimelineDB creates an isolated in-memory SQLite DB with a workspace and
// cognitive_workspace row seeded. Returns the DB and the cognitive_workspace ID.
func setupTimelineDB(t *testing.T, cwID, wsID string) (*sql.DB, string) {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := isqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES (?, ?, ?, datetime('now'), datetime('now'))`, wsID, "Test WS "+wsID, "ws-"+wsID); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO cognitive_workspace (id, workspace_id, status, created_at)
		VALUES (?, ?, 'active', datetime('now'))`, cwID, wsID); err != nil {
		t.Fatalf("cognitive_workspace insert: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })
	return db, cwID
}

func TestReasoningTimeline_AppendAndList(t *testing.T) {
	t.Parallel()

	db, cwID := setupTimelineDB(t, "cw-tl-appendlist", "ws-tl-appendlist")
	tl := blackboard.NewReasoningTimeline(db)
	ctx := context.Background()

	base := time.Now().UTC()
	events := []blackboard.ReasoningEvent{
		{ID: "re-al-0000-0000-000000000001", CognitiveWorkspaceID: cwID, EventType: blackboard.EventTypeObservation, Payload: []byte(`{"n":1}`), CreatedAt: base},
		{ID: "re-al-0000-0000-000000000002", CognitiveWorkspaceID: cwID, EventType: blackboard.EventTypeHypothesis, Payload: []byte(`{"n":2}`), CreatedAt: base.Add(1 * time.Second)},
		{ID: "re-al-0000-0000-000000000003", CognitiveWorkspaceID: cwID, EventType: blackboard.EventTypeRisk, Payload: []byte(`{"n":3}`), CreatedAt: base.Add(2 * time.Second)},
	}

	for _, e := range events {
		if err := tl.Append(ctx, e); err != nil {
			t.Fatalf("Append(%s) error = %v; want nil", e.ID, err)
		}
	}

	got, err := tl.List(ctx, cwID, blackboard.TimelineFilter{})
	if err != nil {
		t.Fatalf("List() error = %v; want nil", err)
	}
	if len(got) != 3 {
		t.Fatalf("List() count = %d; want 3", len(got))
	}
	for i, e := range events {
		if got[i].ID != e.ID {
			t.Errorf("List()[%d].ID = %q; want %q", i, got[i].ID, e.ID)
		}
	}
}

func TestReasoningTimeline_List_Empty(t *testing.T) {
	t.Parallel()

	db, cwID := setupTimelineDB(t, "cw-tl-empty", "ws-tl-empty")
	tl := blackboard.NewReasoningTimeline(db)
	ctx := context.Background()

	got, err := tl.List(ctx, cwID, blackboard.TimelineFilter{})
	if err != nil {
		t.Fatalf("List() error = %v; want nil", err)
	}
	if got == nil {
		t.Error("List() returned nil; want empty non-nil slice")
	}
	if len(got) != 0 {
		t.Errorf("List() count = %d; want 0", len(got))
	}
}

func TestReasoningTimeline_List_FilterByType(t *testing.T) {
	t.Parallel()

	db, cwID := setupTimelineDB(t, "cw-tl-filter", "ws-tl-filter")
	tl := blackboard.NewReasoningTimeline(db)
	ctx := context.Background()

	base := time.Now().UTC()
	events := []blackboard.ReasoningEvent{
		{ID: "re-ft-0000-0000-000000000001", CognitiveWorkspaceID: cwID, EventType: blackboard.EventTypeHypothesis, Payload: []byte(`{}`), CreatedAt: base},
		{ID: "re-ft-0000-0000-000000000002", CognitiveWorkspaceID: cwID, EventType: blackboard.EventTypeObservation, Payload: []byte(`{}`), CreatedAt: base.Add(1 * time.Second)},
		{ID: "re-ft-0000-0000-000000000003", CognitiveWorkspaceID: cwID, EventType: blackboard.EventTypeObservation, Payload: []byte(`{}`), CreatedAt: base.Add(2 * time.Second)},
	}
	for _, e := range events {
		if err := tl.Append(ctx, e); err != nil {
			t.Fatalf("Append error: %v", err)
		}
	}

	got, err := tl.List(ctx, cwID, blackboard.TimelineFilter{EventType: blackboard.EventTypeObservation})
	if err != nil {
		t.Fatalf("List(filter=observation) error = %v; want nil", err)
	}
	if len(got) != 2 {
		t.Fatalf("List() count = %d; want 2", len(got))
	}
	for _, e := range got {
		if e.EventType != blackboard.EventTypeObservation {
			t.Errorf("List() event type = %q; want observation", e.EventType)
		}
	}
}

func TestReasoningTimeline_List_Limit(t *testing.T) {
	t.Parallel()

	db, cwID := setupTimelineDB(t, "cw-tl-limit", "ws-tl-limit")
	tl := blackboard.NewReasoningTimeline(db)
	ctx := context.Background()

	base := time.Now().UTC()
	ids := []string{
		"re-lim-0000-0000-000000000001",
		"re-lim-0000-0000-000000000002",
		"re-lim-0000-0000-000000000003",
		"re-lim-0000-0000-000000000004",
		"re-lim-0000-0000-000000000005",
	}
	for i, id := range ids {
		e := blackboard.ReasoningEvent{
			ID: id, CognitiveWorkspaceID: cwID,
			EventType: blackboard.EventTypeIntent, Payload: []byte(`{}`),
			CreatedAt: base.Add(time.Duration(i) * time.Second),
		}
		if err := tl.Append(ctx, e); err != nil {
			t.Fatalf("Append error: %v", err)
		}
	}

	got, err := tl.List(ctx, cwID, blackboard.TimelineFilter{Limit: 2})
	if err != nil {
		t.Fatalf("List(limit=2) error = %v; want nil", err)
	}
	if len(got) != 2 {
		t.Fatalf("List(limit=2) count = %d; want 2", len(got))
	}
	// Must return the two oldest (ASC order)
	if got[0].ID != ids[0] {
		t.Errorf("List(limit=2)[0].ID = %q; want %q (oldest)", got[0].ID, ids[0])
	}
	if got[1].ID != ids[1] {
		t.Errorf("List(limit=2)[1].ID = %q; want %q", got[1].ID, ids[1])
	}
}

func TestReasoningTimeline_Append_DuplicateID(t *testing.T) {
	t.Parallel()

	db, cwID := setupTimelineDB(t, "cw-tl-dupid", "ws-tl-dupid")
	tl := blackboard.NewReasoningTimeline(db)
	ctx := context.Background()

	e := blackboard.ReasoningEvent{
		ID: "re-dup-0000-0000-000000000001", CognitiveWorkspaceID: cwID,
		EventType: blackboard.EventTypeObservation, Payload: []byte(`{}`),
		CreatedAt: time.Now().UTC(),
	}

	if err := tl.Append(ctx, e); err != nil {
		t.Fatalf("first Append() error = %v; want nil", err)
	}
	if err := tl.Append(ctx, e); err == nil {
		t.Error("second Append() with duplicate ID returned nil; want PK constraint error")
	}
}

func TestReasoningTimeline_WorkspaceIsolation(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() { _ = db.Close() })

	if err := isqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp: %v", err)
	}
	for _, wsID := range []string{"ws-tl-iso-a", "ws-tl-iso-b"} {
		if _, err := db.Exec(`INSERT INTO workspace (id, name, slug, created_at, updated_at)
			VALUES (?, ?, ?, datetime('now'), datetime('now'))`, wsID, "WS "+wsID, wsID); err != nil {
			t.Fatalf("workspace insert %s: %v", wsID, err)
		}
	}
	for _, pair := range [][2]string{{"cw-tl-iso-a", "ws-tl-iso-a"}, {"cw-tl-iso-b", "ws-tl-iso-b"}} {
		if _, err := db.Exec(`INSERT INTO cognitive_workspace (id, workspace_id, status, created_at)
			VALUES (?, ?, 'active', datetime('now'))`, pair[0], pair[1]); err != nil {
			t.Fatalf("cognitive_workspace insert %s: %v", pair[0], err)
		}
	}

	tl := blackboard.NewReasoningTimeline(db)
	ctx := context.Background()
	base := time.Now().UTC()

	evtA := blackboard.ReasoningEvent{
		ID: "re-iso-a-0000-0000-000000000001", CognitiveWorkspaceID: "cw-tl-iso-a",
		EventType: blackboard.EventTypeObservation, Payload: []byte(`{"ws":"a"}`),
		CreatedAt: base,
	}
	evtB := blackboard.ReasoningEvent{
		ID: "re-iso-b-0000-0000-000000000002", CognitiveWorkspaceID: "cw-tl-iso-b",
		EventType: blackboard.EventTypeObservation, Payload: []byte(`{"ws":"b"}`),
		CreatedAt: base,
	}

	if err := tl.Append(ctx, evtA); err != nil {
		t.Fatalf("Append(A) error: %v", err)
	}
	if err := tl.Append(ctx, evtB); err != nil {
		t.Fatalf("Append(B) error: %v", err)
	}

	gotA, err := tl.List(ctx, "cw-tl-iso-a", blackboard.TimelineFilter{})
	if err != nil {
		t.Fatalf("List(A) error: %v", err)
	}
	if len(gotA) != 1 || gotA[0].ID != evtA.ID {
		t.Errorf("List(A) = %v; want [%s]", gotA, evtA.ID)
	}

	gotB, err := tl.List(ctx, "cw-tl-iso-b", blackboard.TimelineFilter{})
	if err != nil {
		t.Fatalf("List(B) error: %v", err)
	}
	if len(gotB) != 1 || gotB[0].ID != evtB.ID {
		t.Errorf("List(B) = %v; want [%s]", gotB, evtB.ID)
	}
}

func TestReasoningTimeline_OrderedByCreatedAt(t *testing.T) {
	t.Parallel()

	db, cwID := setupTimelineDB(t, "cw-tl-order", "ws-tl-order")
	tl := blackboard.NewReasoningTimeline(db)
	ctx := context.Background()

	base := time.Now().UTC()
	// Insert out of chronological order
	outOfOrder := []blackboard.ReasoningEvent{
		{ID: "re-ord-0000-0000-000000000003", CognitiveWorkspaceID: cwID, EventType: blackboard.EventTypeRisk, Payload: []byte(`{"seq":3}`), CreatedAt: base.Add(2 * time.Second)},
		{ID: "re-ord-0000-0000-000000000001", CognitiveWorkspaceID: cwID, EventType: blackboard.EventTypeRisk, Payload: []byte(`{"seq":1}`), CreatedAt: base},
		{ID: "re-ord-0000-0000-000000000002", CognitiveWorkspaceID: cwID, EventType: blackboard.EventTypeRisk, Payload: []byte(`{"seq":2}`), CreatedAt: base.Add(1 * time.Second)},
	}
	for _, e := range outOfOrder {
		if err := tl.Append(ctx, e); err != nil {
			t.Fatalf("Append error: %v", err)
		}
	}

	got, err := tl.List(ctx, cwID, blackboard.TimelineFilter{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("List() count = %d; want 3", len(got))
	}

	wantOrder := []string{
		"re-ord-0000-0000-000000000001",
		"re-ord-0000-0000-000000000002",
		"re-ord-0000-0000-000000000003",
	}
	for i, id := range wantOrder {
		if got[i].ID != id {
			t.Errorf("List()[%d].ID = %q; want %q (ASC by created_at)", i, got[i].ID, id)
		}
	}
}
