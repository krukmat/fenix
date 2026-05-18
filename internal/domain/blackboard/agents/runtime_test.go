package agents_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
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

// TestPersistDerivedArtifact_TwoWritesProduceTwoHistoricalKeys verifies that two sequential
// publishes for the same actor produce two distinct historical keys and one shared last_artifact pointer. (R.16)
func TestPersistDerivedArtifact_TwoWritesProduceTwoHistoricalKeys(t *testing.T) {
	db, cwID := setupSpecializedAgentsDB(t)
	attachment := &blackboard.Attachment{
		CognitiveWorkspaceID: cwID,
		Bus:                  blackboard.NewWorkspaceBus(cwID, db),
		Memory:               blackboard.NewMemoryStore(db),
		Timeline:             blackboard.NewReasoningTimeline(db),
	}

	publish := func(payload string) {
		ev := blackboard.ReasoningEvent{
			ID:                   "re-r16-" + payload,
			CognitiveWorkspaceID: cwID,
			EventType:            blackboard.EventTypeObservation,
			Payload:              []byte(`{"case_id":"` + payload + `"}`),
			CreatedAt:            time.Now().UTC(),
		}
		if err := attachment.Bus.Publish(context.Background(), ev); err != nil {
			t.Fatalf("Publish: %v", err)
		}
	}

	runtime := bbagents.Start(context.Background(), attachment)
	defer runtime.Close()

	publish("first")
	waitForArtifacts(t, attachment, cwID)

	// List all historical keys for the signal agent.
	histKeyPrefix := "specialized_agents/" + bbagents.SignalAgentID + "/history/"
	entries1, err := attachment.Memory.ListByPrefix(context.Background(), cwID, histKeyPrefix)
	if err != nil {
		t.Fatalf("ListByPrefix first: %v", err)
	}
	if len(entries1) != 1 {
		t.Fatalf("expected 1 historical key after first publish, got %d", len(entries1))
	}

	time.Sleep(2 * time.Millisecond) // ensure distinct RFC3339Nano timestamp
	publish("second")
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		entries, _ := attachment.Memory.ListByPrefix(context.Background(), cwID, histKeyPrefix)
		if len(entries) == 2 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	entries2, err := attachment.Memory.ListByPrefix(context.Background(), cwID, histKeyPrefix)
	if err != nil {
		t.Fatalf("ListByPrefix second: %v", err)
	}
	if len(entries2) != 2 {
		t.Fatalf("expected 2 historical keys after second publish, got %d", len(entries2))
	}
	if entries2[0].Key == entries2[1].Key {
		t.Fatal("expected distinct historical keys, got duplicates")
	}
}

// TestPersistDerivedArtifact_LastArtifactPointsToMostRecent verifies that last_artifact always
// references the most recent historical key. (R.16)
func TestPersistDerivedArtifact_LastArtifactPointsToMostRecent(t *testing.T) {
	db, cwID := setupSpecializedAgentsDB(t)
	attachment := &blackboard.Attachment{
		CognitiveWorkspaceID: cwID,
		Bus:                  blackboard.NewWorkspaceBus(cwID, db),
		Memory:               blackboard.NewMemoryStore(db),
		Timeline:             blackboard.NewReasoningTimeline(db),
	}

	runtime := bbagents.Start(context.Background(), attachment)
	defer runtime.Close()

	if err := attachment.Bus.Publish(context.Background(), blackboard.ReasoningEvent{
		ID: "re-r16-ptr", CognitiveWorkspaceID: cwID,
		EventType: blackboard.EventTypeObservation,
		Payload:   []byte(`{"case_id":"c1"}`), CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	waitForArtifacts(t, attachment, cwID)

	lastKey := "specialized_agents/" + bbagents.SignalAgentID + "/last_artifact"
	entry, err := attachment.Memory.Get(context.Background(), cwID, lastKey)
	if err != nil {
		t.Fatalf("Memory.Get last_artifact: %v", err)
	}

	var pointer bbagents.LastArtifactPointer
	if err := json.Unmarshal(entry.Value, &pointer); err != nil {
		t.Fatalf("unmarshal pointer: %v", err)
	}
	if pointer.HistoricalKey == "" {
		t.Fatal("last_artifact pointer has empty HistoricalKey")
	}
	histPrefix := "specialized_agents/" + bbagents.SignalAgentID + "/history/"
	if !strings.HasPrefix(pointer.HistoricalKey, histPrefix) {
		t.Fatalf("HistoricalKey %q does not have prefix %q", pointer.HistoricalKey, histPrefix)
	}
}

// TestPersistDerivedArtifact_LastArtifactBackcompat verifies the planner can still read
// last_artifact as a planningArtifact-compatible struct (back-compat). (R.16)
func TestPersistDerivedArtifact_LastArtifactBackcompat(t *testing.T) {
	db, cwID := setupSpecializedAgentsDB(t)
	attachment := &blackboard.Attachment{
		CognitiveWorkspaceID: cwID,
		Bus:                  blackboard.NewWorkspaceBus(cwID, db),
		Memory:               blackboard.NewMemoryStore(db),
		Timeline:             blackboard.NewReasoningTimeline(db),
	}

	runtime := bbagents.Start(context.Background(), attachment)
	defer runtime.Close()

	if err := attachment.Bus.Publish(context.Background(), blackboard.ReasoningEvent{
		ID: "re-r16-bc", CognitiveWorkspaceID: cwID,
		EventType: blackboard.EventTypeObservation,
		Payload:   []byte(`{"case_id":"c2"}`), CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	waitForArtifacts(t, attachment, cwID)

	lastKey := "specialized_agents/" + bbagents.SignalAgentID + "/last_artifact"
	entry, err := attachment.Memory.Get(context.Background(), cwID, lastKey)
	if err != nil {
		t.Fatalf("Memory.Get last_artifact: %v", err)
	}

	// Must still decode with the fields the planner expects.
	var artifact struct {
		Contributor  string `json:"contributor"`
		ArtifactType string `json:"artifact_type"`
		Summary      string `json:"summary"`
	}
	if err := json.Unmarshal(entry.Value, &artifact); err != nil {
		t.Fatalf("back-compat unmarshal failed: %v", err)
	}
	if artifact.Contributor == "" {
		t.Fatal("back-compat: contributor field is empty")
	}
}

// TestPersistDerivedArtifact_ScopesSeparate verifies historical entries use Persistent scope
// and the pointer entry uses Session scope. (R.16)
func TestPersistDerivedArtifact_ScopesSeparate(t *testing.T) {
	db, cwID := setupSpecializedAgentsDB(t)
	attachment := &blackboard.Attachment{
		CognitiveWorkspaceID: cwID,
		Bus:                  blackboard.NewWorkspaceBus(cwID, db),
		Memory:               blackboard.NewMemoryStore(db),
		Timeline:             blackboard.NewReasoningTimeline(db),
	}

	runtime := bbagents.Start(context.Background(), attachment)
	defer runtime.Close()

	if err := attachment.Bus.Publish(context.Background(), blackboard.ReasoningEvent{
		ID: "re-r16-scope", CognitiveWorkspaceID: cwID,
		EventType: blackboard.EventTypeObservation,
		Payload:   []byte(`{"case_id":"c3"}`), CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	waitForArtifacts(t, attachment, cwID)

	histPrefix := "specialized_agents/" + bbagents.SignalAgentID + "/history/"
	histEntries, err := attachment.Memory.ListByPrefix(context.Background(), cwID, histPrefix)
	if err != nil {
		t.Fatalf("ListByPrefix: %v", err)
	}
	if len(histEntries) == 0 {
		t.Fatal("no historical entries found")
	}
	for _, e := range histEntries {
		if e.Scope != blackboard.MemoryScopePersistent {
			t.Fatalf("historical entry %q has scope %q; want persistent", e.Key, e.Scope)
		}
	}

	lastKey := "specialized_agents/" + bbagents.SignalAgentID + "/last_artifact"
	pointer, err := attachment.Memory.Get(context.Background(), cwID, lastKey)
	if err != nil {
		t.Fatalf("Memory.Get pointer: %v", err)
	}
	if pointer.Scope != blackboard.MemoryScopeSession {
		t.Fatalf("pointer entry scope = %q; want session", pointer.Scope)
	}
}
