package blackboard

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

const (
	defaultArbitrationHalfLife = 24 * time.Hour

	// DefaultArbitrationMemoryKey stores the latest persisted arbitration result.
	DefaultArbitrationMemoryKey = "arbitration/last_ranked_hypotheses"
)

// Arbitrator ranks persisted signal hypotheses for one cognitive workspace.
type Arbitrator interface {
	RankWorkspace(ctx context.Context, cognitiveWorkspaceID string, config ArbitrationConfig) (*ArbitrationResult, error)
}

type sqliteArbitrator struct {
	db     *sql.DB
	memory MemoryStore
}

// NewArbitrator returns an Arbitrator backed by SQLite signal_hypothesis rows.
func NewArbitrator(db *sql.DB) Arbitrator {
	return &sqliteArbitrator{
		db:     db,
		memory: NewMemoryStore(db),
	}
}

// RankWorkspace loads open signal hypotheses, scores them deterministically, and
// optionally persists the derived result into agent_memory.
func (a *sqliteArbitrator) RankWorkspace(ctx context.Context, cognitiveWorkspaceID string, config ArbitrationConfig) (*ArbitrationResult, error) {
	candidates, err := a.listOpenHypotheses(ctx, cognitiveWorkspaceID)
	if err != nil {
		return nil, err
	}

	result := RankHypotheses(cognitiveWorkspaceID, candidates, config)
	if normalizedConfig(config).PersistResult {
		if err := a.persistResult(ctx, result, normalizedConfig(config)); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// RankHypotheses scores in-memory candidates without mutating them.
func RankHypotheses(cognitiveWorkspaceID string, candidates []SignalHypothesis, config ArbitrationConfig) *ArbitrationResult {
	cfg := normalizedConfig(config)
	ranked := make([]RankedHypothesis, 0, len(candidates))

	for _, candidate := range candidates {
		confidence := clamp01(candidate.Confidence)
		recency := recencyWeight(cfg.Now, candidate.CreatedAt, cfg.RecencyHalfLife)
		reliability := reliabilityWeight(candidate.SourceAgentID, cfg.SourceAgentReliability)
		score := confidence * recency * reliability

		ranked = append(ranked, RankedHypothesis{
			Hypothesis: candidate,
			Score:      score,
			Breakdown: ArbitrationScoreBreakdown{
				Confidence:  confidence,
				Recency:     recency,
				Reliability: reliability,
				Final:       score,
			},
		})
	}

	sort.Slice(ranked, func(i, j int) bool {
		left := ranked[i]
		right := ranked[j]

		if !almostEqual(left.Score, right.Score) {
			return left.Score > right.Score
		}
		if !almostEqual(left.Breakdown.Confidence, right.Breakdown.Confidence) {
			return left.Breakdown.Confidence > right.Breakdown.Confidence
		}
		if !left.Hypothesis.CreatedAt.Equal(right.Hypothesis.CreatedAt) {
			return left.Hypothesis.CreatedAt.After(right.Hypothesis.CreatedAt)
		}
		if left.Hypothesis.ID != right.Hypothesis.ID {
			return left.Hypothesis.ID < right.Hypothesis.ID
		}
		return left.Hypothesis.Content < right.Hypothesis.Content
	})

	for i := range ranked {
		ranked[i].Rank = i + 1
	}

	return &ArbitrationResult{
		CognitiveWorkspaceID: cognitiveWorkspaceID,
		GeneratedAt:          cfg.Now,
		Ranked:               ranked,
	}
}

func (a *sqliteArbitrator) listOpenHypotheses(ctx context.Context, cognitiveWorkspaceID string) ([]SignalHypothesis, error) {
	rows, err := a.db.QueryContext(ctx, `
		SELECT id, cognitive_workspace_id, source_agent_id, content, confidence, status, created_at, resolved_at
		FROM signal_hypothesis
		WHERE cognitive_workspace_id = ? AND status = ?
		ORDER BY created_at ASC, id ASC
	`, cognitiveWorkspaceID, string(HypothesisStatusOpen))
	if err != nil {
		return nil, fmt.Errorf("arbitrator list hypotheses: %w", err)
	}
	defer rows.Close()

	hypotheses := []SignalHypothesis{}
	for rows.Next() {
		var hypothesis SignalHypothesis
		var sourceAgentID, resolvedAt sql.NullString
		var status, createdAt string

		if err := rows.Scan(
			&hypothesis.ID,
			&hypothesis.CognitiveWorkspaceID,
			&sourceAgentID,
			&hypothesis.Content,
			&hypothesis.Confidence,
			&status,
			&createdAt,
			&resolvedAt,
		); err != nil {
			return nil, fmt.Errorf("arbitrator scan hypothesis: %w", err)
		}

		if sourceAgentID.Valid {
			hypothesis.SourceAgentID = &sourceAgentID.String
		}
		hypothesis.Status = HypothesisStatus(status)
		parsedCreatedAt, err := parseTime(createdAt)
		if err != nil {
			return nil, fmt.Errorf("arbitrator parse created_at: %w", err)
		}
		hypothesis.CreatedAt = parsedCreatedAt
		if resolvedAt.Valid {
			parsedResolvedAt, err := parseTime(resolvedAt.String)
			if err != nil {
				return nil, fmt.Errorf("arbitrator parse resolved_at: %w", err)
			}
			hypothesis.ResolvedAt = &parsedResolvedAt
		}

		hypotheses = append(hypotheses, hypothesis)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("arbitrator rows error: %w", err)
	}
	return hypotheses, nil
}

func (a *sqliteArbitrator) persistResult(ctx context.Context, result *ArbitrationResult, config ArbitrationConfig) error {
	raw, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("arbitrator marshal result: %w", err)
	}

	if err := a.memory.Set(ctx, AgentMemory{
		ID:                   uuid.NewV7().String(),
		CognitiveWorkspaceID: result.CognitiveWorkspaceID,
		Key:                  config.MemoryKey,
		Value:                raw,
		Scope:                MemoryScopeSession,
		CreatedAt:            config.Now,
		UpdatedAt:            config.Now,
	}); err != nil {
		return fmt.Errorf("arbitrator persist result: %w", err)
	}
	return nil
}

func normalizedConfig(config ArbitrationConfig) ArbitrationConfig {
	cfg := config
	if cfg.Now.IsZero() {
		cfg.Now = time.Now().UTC()
	} else {
		cfg.Now = cfg.Now.UTC()
	}
	if cfg.RecencyHalfLife <= 0 {
		cfg.RecencyHalfLife = defaultArbitrationHalfLife
	}
	if cfg.SourceAgentReliability == nil {
		cfg.SourceAgentReliability = map[string]float64{}
	}
	if cfg.MemoryKey == "" {
		cfg.MemoryKey = DefaultArbitrationMemoryKey
	}
	if !cfg.PersistResult {
		cfg.PersistResult = true
	}
	return cfg
}

func recencyWeight(now, createdAt time.Time, halfLife time.Duration) float64 {
	if createdAt.IsZero() {
		return 1.0
	}

	age := now.Sub(createdAt)
	if age <= 0 {
		return 1.0
	}

	return math.Exp(-math.Ln2 * age.Seconds() / halfLife.Seconds())
}

func reliabilityWeight(sourceAgentID *string, reliability map[string]float64) float64 {
	if sourceAgentID == nil {
		return 1.0
	}
	value, ok := reliability[*sourceAgentID]
	if !ok {
		return 1.0
	}
	return clamp01(value)
}

func clamp01(value float64) float64 {
	switch {
	case value < 0:
		return 0
	case value > 1:
		return 1
	default:
		return value
	}
}

func almostEqual(left, right float64) bool {
	return math.Abs(left-right) < 1e-9
}
