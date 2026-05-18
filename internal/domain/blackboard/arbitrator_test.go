package blackboard_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/blackboard"
	isqlite "github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	_ "modernc.org/sqlite"
)

func setupArbitratorDB(t *testing.T) (*sql.DB, string) {
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
		VALUES ('ws-arb', 'Arb WS', 'arb-ws', datetime('now'), datetime('now'))`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO cognitive_workspace (id, workspace_id, status, created_at)
		VALUES ('cw-arb', 'ws-arb', 'active', datetime('now'))`); err != nil {
		t.Fatalf("cognitive_workspace insert: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })
	return db, "cw-arb"
}

func insertHypothesis(t *testing.T, db *sql.DB, hypothesis blackboard.SignalHypothesis) {
	t.Helper()

	_, err := db.Exec(`
		INSERT INTO signal_hypothesis (
			id, cognitive_workspace_id, source_agent_id, content, confidence, status, created_at, resolved_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		hypothesis.ID,
		hypothesis.CognitiveWorkspaceID,
		hypothesis.SourceAgentID,
		hypothesis.Content,
		hypothesis.Confidence,
		string(hypothesis.Status),
		hypothesis.CreatedAt.UTC().Format(time.RFC3339),
		formatNullableTime(hypothesis.ResolvedAt),
	)
	if err != nil {
		t.Fatalf("insert hypothesis %s: %v", hypothesis.ID, err)
	}
}

func formatNullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339)
}

func TestArbitratorRankWorkspace_ConfidenceDrivesOrdering(t *testing.T) {
	db, cwID := setupArbitratorDB(t)
	now := time.Date(2026, 5, 18, 9, 0, 0, 0, time.UTC)
	agentID := "signal-agent"

	insertHypothesis(t, db, blackboard.SignalHypothesis{
		ID:                   "hyp-low",
		CognitiveWorkspaceID: cwID,
		SourceAgentID:        &agentID,
		Content:              "low confidence",
		Confidence:           0.30,
		Status:               blackboard.HypothesisStatusOpen,
		CreatedAt:            now.Add(-1 * time.Hour),
	})
	insertHypothesis(t, db, blackboard.SignalHypothesis{
		ID:                   "hyp-high",
		CognitiveWorkspaceID: cwID,
		SourceAgentID:        &agentID,
		Content:              "high confidence",
		Confidence:           0.90,
		Status:               blackboard.HypothesisStatusOpen,
		CreatedAt:            now.Add(-1 * time.Hour),
	})

	result, err := blackboard.NewArbitrator(db).RankWorkspace(context.Background(), cwID, blackboard.ArbitrationConfig{
		Now:           now,
		PersistResult: false,
	})
	if err != nil {
		t.Fatalf("RankWorkspace(): %v", err)
	}
	if len(result.Ranked) != 2 {
		t.Fatalf("ranked count = %d; want 2", len(result.Ranked))
	}
	if got := result.Ranked[0].Hypothesis.ID; got != "hyp-high" {
		t.Fatalf("top hypothesis = %q; want hyp-high", got)
	}
}

func TestArbitratorRankWorkspace_RecencyBreaksOtherwiseEqualCandidates(t *testing.T) {
	db, cwID := setupArbitratorDB(t)
	now := time.Date(2026, 5, 18, 9, 0, 0, 0, time.UTC)
	agentID := "signal-agent"

	insertHypothesis(t, db, blackboard.SignalHypothesis{
		ID:                   "hyp-old",
		CognitiveWorkspaceID: cwID,
		SourceAgentID:        &agentID,
		Content:              "older",
		Confidence:           0.8,
		Status:               blackboard.HypothesisStatusOpen,
		CreatedAt:            now.Add(-48 * time.Hour),
	})
	insertHypothesis(t, db, blackboard.SignalHypothesis{
		ID:                   "hyp-new",
		CognitiveWorkspaceID: cwID,
		SourceAgentID:        &agentID,
		Content:              "newer",
		Confidence:           0.8,
		Status:               blackboard.HypothesisStatusOpen,
		CreatedAt:            now.Add(-2 * time.Hour),
	})

	result, err := blackboard.NewArbitrator(db).RankWorkspace(context.Background(), cwID, blackboard.ArbitrationConfig{
		Now:           now,
		PersistResult: false,
	})
	if err != nil {
		t.Fatalf("RankWorkspace(): %v", err)
	}
	if got := result.Ranked[0].Hypothesis.ID; got != "hyp-new" {
		t.Fatalf("top hypothesis = %q; want hyp-new", got)
	}
}

func TestArbitratorRankWorkspace_ReliabilityAffectsOrdering(t *testing.T) {
	db, cwID := setupArbitratorDB(t)
	now := time.Date(2026, 5, 18, 9, 0, 0, 0, time.UTC)
	reliable := "signal-agent"
	unreliable := "policy-agent"

	insertHypothesis(t, db, blackboard.SignalHypothesis{
		ID:                   "hyp-reliable",
		CognitiveWorkspaceID: cwID,
		SourceAgentID:        &reliable,
		Content:              "reliable source",
		Confidence:           0.7,
		Status:               blackboard.HypothesisStatusOpen,
		CreatedAt:            now.Add(-1 * time.Hour),
	})
	insertHypothesis(t, db, blackboard.SignalHypothesis{
		ID:                   "hyp-unreliable",
		CognitiveWorkspaceID: cwID,
		SourceAgentID:        &unreliable,
		Content:              "unreliable source",
		Confidence:           0.9,
		Status:               blackboard.HypothesisStatusOpen,
		CreatedAt:            now.Add(-1 * time.Hour),
	})

	result, err := blackboard.NewArbitrator(db).RankWorkspace(context.Background(), cwID, blackboard.ArbitrationConfig{
		Now: now,
		SourceAgentReliability: map[string]float64{
			reliable:   1.0,
			unreliable: 0.4,
		},
		PersistResult: false,
	})
	if err != nil {
		t.Fatalf("RankWorkspace(): %v", err)
	}
	if got := result.Ranked[0].Hypothesis.ID; got != "hyp-reliable" {
		t.Fatalf("top hypothesis = %q; want hyp-reliable", got)
	}
}

func TestArbitratorRankWorkspace_PersistsResultWithoutMutatingSourceHypotheses(t *testing.T) {
	db, cwID := setupArbitratorDB(t)
	now := time.Date(2026, 5, 18, 9, 0, 0, 0, time.UTC)
	agentID := "signal-agent"

	insertHypothesis(t, db, blackboard.SignalHypothesis{
		ID:                   "hyp-persist",
		CognitiveWorkspaceID: cwID,
		SourceAgentID:        &agentID,
		Content:              "persist me",
		Confidence:           0.85,
		Status:               blackboard.HypothesisStatusOpen,
		CreatedAt:            now.Add(-1 * time.Hour),
	})

	result, err := blackboard.NewArbitrator(db).RankWorkspace(context.Background(), cwID, blackboard.ArbitrationConfig{
		Now: now,
	})
	if err != nil {
		t.Fatalf("RankWorkspace(): %v", err)
	}

	store := blackboard.NewMemoryStore(db)
	entry, err := store.Get(context.Background(), cwID, "arbitration/last_ranked_hypotheses")
	if err != nil {
		t.Fatalf("Memory.Get(): %v", err)
	}

	var persisted blackboard.ArbitrationResult
	if err := json.Unmarshal(entry.Value, &persisted); err != nil {
		t.Fatalf("json.Unmarshal(): %v", err)
	}
	if len(persisted.Ranked) != 1 || persisted.Ranked[0].Hypothesis.ID != result.Ranked[0].Hypothesis.ID {
		t.Fatalf("persisted result = %#v; want top hypothesis %q", persisted.Ranked, result.Ranked[0].Hypothesis.ID)
	}

	var status string
	if err := db.QueryRow(`SELECT status FROM signal_hypothesis WHERE id = ?`, "hyp-persist").Scan(&status); err != nil {
		t.Fatalf("select hypothesis status: %v", err)
	}
	if status != string(blackboard.HypothesisStatusOpen) {
		t.Fatalf("status = %q; want %q", status, blackboard.HypothesisStatusOpen)
	}
}

func TestArbitratorRankWorkspace_DeterministicTieBreaker(t *testing.T) {
	db, cwID := setupArbitratorDB(t)
	now := time.Date(2026, 5, 18, 9, 0, 0, 0, time.UTC)
	agentID := "signal-agent"
	createdAt := now.Add(-1 * time.Hour)

	insertHypothesis(t, db, blackboard.SignalHypothesis{
		ID:                   "hyp-b",
		CognitiveWorkspaceID: cwID,
		SourceAgentID:        &agentID,
		Content:              "candidate b",
		Confidence:           0.75,
		Status:               blackboard.HypothesisStatusOpen,
		CreatedAt:            createdAt,
	})
	insertHypothesis(t, db, blackboard.SignalHypothesis{
		ID:                   "hyp-a",
		CognitiveWorkspaceID: cwID,
		SourceAgentID:        &agentID,
		Content:              "candidate a",
		Confidence:           0.75,
		Status:               blackboard.HypothesisStatusOpen,
		CreatedAt:            createdAt,
	})

	result, err := blackboard.NewArbitrator(db).RankWorkspace(context.Background(), cwID, blackboard.ArbitrationConfig{
		Now:           now,
		PersistResult: false,
	})
	if err != nil {
		t.Fatalf("RankWorkspace(): %v", err)
	}
	if len(result.Ranked) != 2 {
		t.Fatalf("ranked count = %d; want 2", len(result.Ranked))
	}
	if got := result.Ranked[0].Hypothesis.ID; got != "hyp-a" {
		t.Fatalf("first hypothesis = %q; want hyp-a", got)
	}
	if got := result.Ranked[1].Hypothesis.ID; got != "hyp-b" {
		t.Fatalf("second hypothesis = %q; want hyp-b", got)
	}
}
