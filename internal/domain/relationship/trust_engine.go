// Package relationship — Task B.3: TrustEngine service.
// Computes weighted recency-decay trust score from interaction signals
// and delegates persistence to TrustRepository. No SQL, no LLM, no HTTP.
package relationship

import (
	"context"
	"math"
	"time"
)

// decayLambda controls how fast older signals lose influence.
// λ=0.05 → ~50% weight at 14 days, ~10% at 46 days.
const decayLambda = 0.05

// TrustRepository is the persistence contract for TrustEngine.
// Implemented by a SQLite adapter (future task); faked in unit tests.
type TrustRepository interface {
	UpsertTrustScore(ctx context.Context, memoryID string, score float64,
		confidence ConfidenceLevel, decayFactor float64, lastScoredAt time.Time) error
}

// TrustEngine computes and persists trust scores for relationship memories.
// Task B.3: stateless — caller supplies signals, engine never queries DB.
type TrustEngine struct {
	repo TrustRepository
}

// NewTrustEngine constructs a TrustEngine backed by the given repository.
func NewTrustEngine(repo TrustRepository) *TrustEngine {
	return &TrustEngine{repo: repo}
}

// Score computes the trust score from signals and upserts it via the repository.
// Returns the repository error if the upsert fails — caller decides retry policy.
func (e *TrustEngine) Score(ctx context.Context, memoryID string, signals []InteractionSignal) error {
	score, confidence, decayFactor := computeScore(signals)
	return e.repo.UpsertTrustScore(ctx, memoryID, score, confidence, decayFactor, time.Now().UTC())
}

// computeScore is a pure function: no receiver, no side effects, fully deterministic.
// Algorithm: weighted sentiment sum with exponential recency decay, normalised to [0,1].
func computeScore(signals []InteractionSignal) (score float64, confidence ConfidenceLevel, decayFactor float64) {
	if len(signals) == 0 {
		return 0.5, ConfidenceLow, 1.0
	}

	now := time.Now().UTC()
	var totalWeight, totalDecay float64

	for i := range signals {
		days := now.Sub(signals[i].OccurredAt).Hours() / 24
		decay := math.Exp(-decayLambda * days)
		totalWeight += sentimentValue(signals[i].Sentiment) * decay
		totalDecay += decay
	}

	n := float64(len(signals))
	raw := totalWeight / n
	score = clamp((raw+1.0)/2.0, 0.0, 1.0)
	decayFactor = clamp(totalDecay/n, 0.0, 1.0)
	confidence = confidenceTier(len(signals))

	return score, confidence, decayFactor
}

// sentimentValue maps a nullable SentimentType to its numeric contribution.
func sentimentValue(s *SentimentType) float64 {
	if s == nil {
		return 0.0
	}
	switch *s {
	case SentimentPositive:
		return 1.0
	case SentimentNegative:
		return -1.0
	default:
		return 0.0
	}
}

// confidenceTier maps signal count to a ConfidenceLevel.
func confidenceTier(n int) ConfidenceLevel {
	switch {
	case n >= 5:
		return ConfidenceHigh
	case n >= 2:
		return ConfidenceMedium
	default:
		return ConfidenceLow
	}
}

// clamp constrains v to [lo, hi].
func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}
