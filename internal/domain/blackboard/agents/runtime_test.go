package agents_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/blackboard"
	bbagents "github.com/matiasleandrokruk/fenix/internal/domain/blackboard/agents"
	isqlite "github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	_ "modernc.org/sqlite"
)

func setupSpecializedAgentsDB(t *testing.T) (*sql.DB, string) {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := isqlite.MigrateUp(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-specialized', 'Specialized WS', 'specialized-ws', datetime('now'), datetime('now'))`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO cognitive_workspace (id, workspace_id, status, created_at)
		VALUES ('cw-specialized', 'ws-specialized', 'active', datetime('now'))`); err != nil {
		t.Fatalf("cognitive_workspace insert: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })
	return db, "cw-specialized"
}

func TestRuntime_StartPersistsSpecializedArtifacts(t *testing.T) {
	db, cwID := setupSpecializedAgentsDB(t)
	attachment := &blackboard.Attachment{
		CognitiveWorkspaceID: cwID,
		Bus:                  blackboard.NewWorkspaceBus(cwID, db),
		Memory:               blackboard.NewMemoryStore(db),
		Timeline:             blackboard.NewReasoningTimeline(db),
	}

	runtime := bbagents.Start(context.Background(), attachment)
	if runtime == nil {
		t.Fatal("Start() returned nil runtime")
	}

	sourceEvent := blackboard.ReasoningEvent{
		ID:                   "re-specialized-source",
		CognitiveWorkspaceID: cwID,
		EventType:            blackboard.EventTypeObservation,
		Payload:              []byte(`{"case_id":"case-1"}`),
		CreatedAt:            time.Now().UTC(),
	}
	if err := attachment.Bus.Publish(context.Background(), sourceEvent); err != nil {
		t.Fatalf("Publish(): %v", err)
	}

	waitForArtifacts(t, attachment, cwID)
	runtime.Close()

	events, err := attachment.Timeline.List(context.Background(), cwID, blackboard.TimelineFilter{})
	if err != nil {
		t.Fatalf("Timeline.List(): %v", err)
	}
	if len(events) != 4 {
		t.Fatalf("timeline event count = %d; want 4", len(events))
	}

	assertMemoryEntry(t, attachment, cwID, "specialized_agents/"+bbagents.SignalAgentID+"/last_artifact")
	assertMemoryEntry(t, attachment, cwID, "specialized_agents/"+bbagents.EvidenceAgentID+"/last_artifact")
	assertMemoryEntry(t, attachment, cwID, "specialized_agents/"+bbagents.PolicyAgentID+"/last_artifact")
}

func waitForArtifacts(t *testing.T, attachment *blackboard.Attachment, cwID string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		events, err := attachment.Timeline.List(context.Background(), cwID, blackboard.TimelineFilter{})
		if err == nil && len(events) == 4 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("timeout waiting for specialized artifacts")
}

func assertMemoryEntry(t *testing.T, attachment *blackboard.Attachment, cwID, key string) {
	t.Helper()

	entry, err := attachment.Memory.Get(context.Background(), cwID, key)
	if err != nil {
		t.Fatalf("Memory.Get(%q): %v", key, err)
	}
	if len(entry.Value) == 0 {
		t.Fatalf("memory entry %q has empty value", key)
	}
}
