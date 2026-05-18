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
		persistErr := a.persistResult(ctx, result, normalizedConfig(config))
		if persistErr != nil {
			return nil, persistErr
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
		return rankedHypothesisLess(ranked[i], ranked[j])
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

	hypotheses := make([]SignalHypothesis, 0)
	for rows.Next() {
		h, scanErr := scanHypothesisRow(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		hypotheses = append(hypotheses, h)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("arbitrator rows error: %w", rowsErr)
	}
	return hypotheses, nil
}

func scanHypothesisRow(rows *sql.Rows) (SignalHypothesis, error) {
	var h SignalHypothesis
	var sourceAgentID, resolvedAt sql.NullString
	var status, createdAt string

	if err := rows.Scan(
		&h.ID, &h.CognitiveWorkspaceID, &sourceAgentID,
		&h.Content, &h.Confidence, &status, &createdAt, &resolvedAt,
	); err != nil {
		return h, fmt.Errorf("arbitrator scan hypothesis: %w", err)
	}
	if sourceAgentID.Valid {
		h.SourceAgentID = &sourceAgentID.String
	}
	h.Status = HypothesisStatus(status)
	parsedCreatedAt, err := parseTime(createdAt)
	if err != nil {
		return h, fmt.Errorf("arbitrator parse created_at: %w", err)
	}
	h.CreatedAt = parsedCreatedAt
	if resolvedAt.Valid {
		parsedResolvedAt, parseErr := parseTime(resolvedAt.String)
		if parseErr != nil {
			return h, fmt.Errorf("arbitrator parse resolved_at: %w", parseErr)
		}
		h.ResolvedAt = &parsedResolvedAt
	}
	return h, nil
}

func (a *sqliteArbitrator) persistResult(ctx context.Context, result *ArbitrationResult, config ArbitrationConfig) error {
	raw, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("arbitrator marshal result: %w", err)
	}

	setErr := a.memory.Set(ctx, AgentMemory{
		ID:                   uuid.NewV7().String(),
		CognitiveWorkspaceID: result.CognitiveWorkspaceID,
		Key:                  config.MemoryKey,
		Value:                raw,
		Scope:                MemoryScopeSession,
		CreatedAt:            config.Now,
		UpdatedAt:            config.Now,
	})
	if setErr != nil {
		return fmt.Errorf("arbitrator persist result: %w", setErr)
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

func rankedHypothesisLess(a, b RankedHypothesis) bool {
	if !almostEqual(a.Score, b.Score) {
		return a.Score > b.Score
	}
	if !almostEqual(a.Breakdown.Confidence, b.Breakdown.Confidence) {
		return a.Breakdown.Confidence > b.Breakdown.Confidence
	}
	if !a.Hypothesis.CreatedAt.Equal(b.Hypothesis.CreatedAt) {
		return a.Hypothesis.CreatedAt.After(b.Hypothesis.CreatedAt)
	}
	if a.Hypothesis.ID != b.Hypothesis.ID {
		return a.Hypothesis.ID < b.Hypothesis.ID
	}
	return a.Hypothesis.Content < b.Hypothesis.Content
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
