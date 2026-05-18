package blackboard_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/blackboard"
	isqlite "github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
	_ "modernc.org/sqlite"
)

func setupPlannerDB(t *testing.T) (*sql.DB, string) {
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
		VALUES ('ws-plan', 'Plan WS', 'plan-ws', datetime('now'), datetime('now'))`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO cognitive_workspace (id, workspace_id, status, created_at)
		VALUES ('cw-plan', 'ws-plan', 'active', datetime('now'))`); err != nil {
		t.Fatalf("cognitive_workspace insert: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })
	return db, "cw-plan"
}

func persistMemoryJSON(t *testing.T, store blackboard.MemoryStore, cwID, key string, now time.Time, payload any) {
	t.Helper()

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal(%s): %v", key, err)
	}
	if err := store.Set(context.Background(), blackboard.AgentMemory{
		ID:                   uuid.NewV7().String(),
		CognitiveWorkspaceID: cwID,
		Key:                  key,
		Value:                raw,
		Scope:                blackboard.MemoryScopeSession,
		CreatedAt:            now,
		UpdatedAt:            now,
	}); err != nil {
		t.Fatalf("store.Set(%s): %v", key, err)
	}
}

func rankedResult(cwID string, now time.Time, ranked ...blackboard.RankedHypothesis) blackboard.ArbitrationResult {
	return blackboard.ArbitrationResult{
		CognitiveWorkspaceID: cwID,
		GeneratedAt:          now,
		Ranked:               ranked,
	}
}

func TestPlannerBuildWorkspacePlan_ReadyWhenEvidenceExistsAndScoreIsHigh(t *testing.T) {
	db, cwID := setupPlannerDB(t)
	store := blackboard.NewMemoryStore(db)
	now := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)

	persistMemoryJSON(t, store, cwID, blackboard.DefaultArbitrationMemoryKey, now, rankedResult(cwID, now,
		blackboard.RankedHypothesis{
			Rank:  1,
			Score: 0.91,
			Hypothesis: blackboard.SignalHypothesis{
				ID:                   "hyp-ready",
				CognitiveWorkspaceID: cwID,
				Content:              "respond to case escalation",
				Confidence:           0.91,
				Status:               blackboard.HypothesisStatusOpen,
				CreatedAt:            now.Add(-1 * time.Hour),
			},
		},
	))
	persistMemoryJSON(t, store, cwID, "specialized_agents/blackboard-evidence-agent/last_artifact", now, map[string]any{
		"contributor":   "blackboard-evidence-agent",
		"artifact_type": "evidence_finding",
		"summary":       "Evidence confirms the case escalation context.",
	})

	result, err := blackboard.NewPlanner(db).BuildWorkspacePlan(context.Background(), cwID, blackboard.PlanningConfig{
		Now: now,
	})
	if err != nil {
		t.Fatalf("BuildWorkspacePlan(): %v", err)
	}
	if result.State != blackboard.PlanningStateReady {
		t.Fatalf("state = %q; want %q", result.State, blackboard.PlanningStateReady)
	}
	if result.SelectedProposal == nil || result.SelectedProposal.State != blackboard.PlanningStateReady {
		t.Fatalf("selected proposal = %#v; want ready", result.SelectedProposal)
	}
	if len(result.SelectedProposal.Steps) != 3 {
		t.Fatalf("steps = %d; want 3", len(result.SelectedProposal.Steps))
	}
}

func TestPlannerBuildWorkspacePlan_PendingApprovalWhenPolicyConstraintExists(t *testing.T) {
	db, cwID := setupPlannerDB(t)
	store := blackboard.NewMemoryStore(db)
	now := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)

	persistMemoryJSON(t, store, cwID, blackboard.DefaultArbitrationMemoryKey, now, rankedResult(cwID, now,
		blackboard.RankedHypothesis{
			Rank:  1,
			Score: 0.88,
			Hypothesis: blackboard.SignalHypothesis{
				ID:                   "hyp-policy",
				CognitiveWorkspaceID: cwID,
				Content:              "perform governed reply",
				Confidence:           0.88,
				Status:               blackboard.HypothesisStatusOpen,
				CreatedAt:            now.Add(-1 * time.Hour),
			},
		},
	))
	persistMemoryJSON(t, store, cwID, "specialized_agents/blackboard-evidence-agent/last_artifact", now, map[string]any{
		"contributor":   "blackboard-evidence-agent",
		"artifact_type": "evidence_finding",
		"summary":       "Evidence pack is sufficient.",
	})
	persistMemoryJSON(t, store, cwID, "specialized_agents/blackboard-policy-agent/last_artifact", now, map[string]any{
		"contributor":   "blackboard-policy-agent",
		"artifact_type": "policy_constraint",
		"summary":       "Sensitive reply requires approval before execution.",
	})

	result, err := blackboard.NewPlanner(db).BuildWorkspacePlan(context.Background(), cwID, blackboard.PlanningConfig{
		Now: now,
	})
	if err != nil {
		t.Fatalf("BuildWorkspacePlan(): %v", err)
	}
	if result.State != blackboard.PlanningStatePendingApproval {
		t.Fatalf("state = %q; want %q", result.State, blackboard.PlanningStatePendingApproval)
	}
	lastStep := result.SelectedProposal.Steps[len(result.SelectedProposal.Steps)-2]
	if !lastStep.RequiresApproval {
		t.Fatalf("approval step = %#v; want RequiresApproval=true", lastStep)
	}
	if len(result.SelectedProposal.Constraints) == 0 {
		t.Fatal("expected explicit policy constraints")
	}
}

func TestPlannerBuildWorkspacePlan_AwaitingEvidenceWhenEvidenceArtifactMissing(t *testing.T) {
	db, cwID := setupPlannerDB(t)
	store := blackboard.NewMemoryStore(db)
	now := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)

	persistMemoryJSON(t, store, cwID, blackboard.DefaultArbitrationMemoryKey, now, rankedResult(cwID, now,
		blackboard.RankedHypothesis{
			Rank:  1,
			Score: 0.83,
			Hypothesis: blackboard.SignalHypothesis{
				ID:                   "hyp-awaiting-evidence",
				CognitiveWorkspaceID: cwID,
				Content:              "needs more evidence",
				Confidence:           0.83,
				Status:               blackboard.HypothesisStatusOpen,
				CreatedAt:            now.Add(-1 * time.Hour),
			},
		},
	))

	result, err := blackboard.NewPlanner(db).BuildWorkspacePlan(context.Background(), cwID, blackboard.PlanningConfig{
		Now: now,
	})
	if err != nil {
		t.Fatalf("BuildWorkspacePlan(): %v", err)
	}
	if result.State != blackboard.PlanningStateAwaitingEvidence {
		t.Fatalf("state = %q; want %q", result.State, blackboard.PlanningStateAwaitingEvidence)
	}
}

func TestPlannerBuildWorkspacePlan_NeedsReviewBelowReadyThreshold(t *testing.T) {
	db, cwID := setupPlannerDB(t)
	store := blackboard.NewMemoryStore(db)
	now := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)

	persistMemoryJSON(t, store, cwID, blackboard.DefaultArbitrationMemoryKey, now, rankedResult(cwID, now,
		blackboard.RankedHypothesis{
			Rank:  1,
			Score: 0.41,
			Hypothesis: blackboard.SignalHypothesis{
				ID:                   "hyp-needs-review",
				CognitiveWorkspaceID: cwID,
				Content:              "low-score candidate",
				Confidence:           0.41,
				Status:               blackboard.HypothesisStatusOpen,
				CreatedAt:            now.Add(-1 * time.Hour),
			},
		},
	))
	persistMemoryJSON(t, store, cwID, "specialized_agents/blackboard-evidence-agent/last_artifact", now, map[string]any{
		"contributor":   "blackboard-evidence-agent",
		"artifact_type": "evidence_finding",
		"summary":       "Evidence exists but confidence remains low.",
	})

	result, err := blackboard.NewPlanner(db).BuildWorkspacePlan(context.Background(), cwID, blackboard.PlanningConfig{
		Now:           now,
		MinReadyScore: 0.6,
	})
	if err != nil {
		t.Fatalf("BuildWorkspacePlan(): %v", err)
	}
	if result.State != blackboard.PlanningStateNeedsReview {
		t.Fatalf("state = %q; want %q", result.State, blackboard.PlanningStateNeedsReview)
	}
}

func TestPlannerBuildWorkspacePlan_PreservesArbitrationOrderAndPersistsResult(t *testing.T) {
	db, cwID := setupPlannerDB(t)
	store := blackboard.NewMemoryStore(db)
	now := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)
	arbitration := rankedResult(cwID, now,
		blackboard.RankedHypothesis{
			Rank:  1,
			Score: 0.72,
			Hypothesis: blackboard.SignalHypothesis{
				ID:                   "hyp-first",
				CognitiveWorkspaceID: cwID,
				Content:              "first ranked",
				Confidence:           0.72,
				Status:               blackboard.HypothesisStatusOpen,
				CreatedAt:            now.Add(-1 * time.Hour),
			},
		},
		blackboard.RankedHypothesis{
			Rank:  2,
			Score: 0.69,
			Hypothesis: blackboard.SignalHypothesis{
				ID:                   "hyp-second",
				CognitiveWorkspaceID: cwID,
				Content:              "second ranked",
				Confidence:           0.69,
				Status:               blackboard.HypothesisStatusOpen,
				CreatedAt:            now.Add(-2 * time.Hour),
			},
		},
	)
	persistMemoryJSON(t, store, cwID, blackboard.DefaultArbitrationMemoryKey, now, arbitration)
	persistMemoryJSON(t, store, cwID, "specialized_agents/blackboard-evidence-agent/last_artifact", now, map[string]any{
		"contributor":   "blackboard-evidence-agent",
		"artifact_type": "evidence_finding",
		"summary":       "Evidence exists for both candidates.",
	})

	result, err := blackboard.NewPlanner(db).BuildWorkspacePlan(context.Background(), cwID, blackboard.PlanningConfig{
		Now: now,
	})
	if err != nil {
		t.Fatalf("BuildWorkspacePlan(): %v", err)
	}
	if result.SelectedProposal == nil || result.SelectedProposal.HypothesisID != "hyp-first" {
		t.Fatalf("selected proposal = %#v; want hyp-first", result.SelectedProposal)
	}
	if len(result.Proposals) != 2 || result.Proposals[1].HypothesisID != "hyp-second" {
		t.Fatalf("proposal order = %#v; want hyp-first then hyp-second", result.Proposals)
	}

	entry, err := store.Get(context.Background(), cwID, "planning/last_collaborative_plan")
	if err != nil {
		t.Fatalf("Memory.Get(planning result): %v", err)
	}
	var persisted blackboard.CollaborativePlanningResult
	if err := json.Unmarshal(entry.Value, &persisted); err != nil {
		t.Fatalf("json.Unmarshal(planning result): %v", err)
	}
	if persisted.SelectedProposal == nil || persisted.SelectedProposal.HypothesisID != "hyp-first" {
		t.Fatalf("persisted selected proposal = %#v; want hyp-first", persisted.SelectedProposal)
	}

	arbEntry, err := store.Get(context.Background(), cwID, blackboard.DefaultArbitrationMemoryKey)
	if err != nil {
		t.Fatalf("Memory.Get(arbitration): %v", err)
	}
	var unchanged blackboard.ArbitrationResult
	if err := json.Unmarshal(arbEntry.Value, &unchanged); err != nil {
		t.Fatalf("json.Unmarshal(arbitration): %v", err)
	}
	if len(unchanged.Ranked) != 2 || unchanged.Ranked[0].Hypothesis.ID != "hyp-first" {
		t.Fatalf("arbitration memory mutated: %#v", unchanged.Ranked)
	}
}
