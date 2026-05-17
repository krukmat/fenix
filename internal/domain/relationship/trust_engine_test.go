// Task B.3 — TrustEngine unit tests.
// No real DB, no LLM, no network. fakeTrustRepo captures calls for assertion.
package relationship

import (
	"context"
	"errors"
	"testing"
	"time"
)

// --- Test double ---

type trustUpsertArgs struct {
	memoryID     string
	score        float64
	confidence   ConfidenceLevel
	decayFactor  float64
	lastScoredAt time.Time
}

type fakeTrustRepo struct {
	calls []trustUpsertArgs
	err   error
}

func (r *fakeTrustRepo) UpsertTrustScore(_ context.Context, memoryID string, score float64,
	confidence ConfidenceLevel, decayFactor float64, lastScoredAt time.Time) error {
	r.calls = append(r.calls, trustUpsertArgs{memoryID, score, confidence, decayFactor, lastScoredAt})
	return r.err
}

// --- Helpers ---

func positiveSignal(daysAgo float64) InteractionSignal {
	s := SentimentPositive
	return InteractionSignal{
		Sentiment:  &s,
		OccurredAt: time.Now().UTC().Add(-time.Duration(daysAgo*24) * time.Hour),
	}
}

func negativeSignal(daysAgo float64) InteractionSignal {
	s := SentimentNegative
	return InteractionSignal{
		Sentiment:  &s,
		OccurredAt: time.Now().UTC().Add(-time.Duration(daysAgo*24) * time.Hour),
	}
}

func neutralSignal(daysAgo float64) InteractionSignal {
	s := SentimentNeutral
	return InteractionSignal{
		Sentiment:  &s,
		OccurredAt: time.Now().UTC().Add(-time.Duration(daysAgo*24) * time.Hour),
	}
}

// --- computeScore pure function tests ---

func TestComputeScore_NoSignals(t *testing.T) {
	score, confidence, decayFactor := computeScore(nil)

	if score != 0.5 {
		t.Errorf("score: want 0.5, got %f", score)
	}
	if confidence != ConfidenceLow {
		t.Errorf("confidence: want low, got %s", confidence)
	}
	if decayFactor != 1.0 {
		t.Errorf("decayFactor: want 1.0, got %f", decayFactor)
	}
}

func TestComputeScore_AllPositive(t *testing.T) {
	signals := []InteractionSignal{positiveSignal(1), positiveSignal(2), positiveSignal(3)}
	score, _, _ := computeScore(signals)

	if score <= 0.5 {
		t.Errorf("all positive signals: score should be > 0.5, got %f", score)
	}
}

func TestComputeScore_AllNegative(t *testing.T) {
	signals := []InteractionSignal{negativeSignal(1), negativeSignal(2), negativeSignal(3)}
	score, _, _ := computeScore(signals)

	if score >= 0.5 {
		t.Errorf("all negative signals: score should be < 0.5, got %f", score)
	}
}

func TestComputeScore_Mixed(t *testing.T) {
	signals := []InteractionSignal{positiveSignal(1), negativeSignal(1)}
	score, _, _ := computeScore(signals)

	// symmetric input at equal recency → score ≈ 0.5
	if score < 0.4 || score > 0.6 {
		t.Errorf("mixed equal signals: score should be near 0.5, got %f", score)
	}
}

func TestComputeScore_ConfidenceHigh(t *testing.T) {
	signals := make([]InteractionSignal, 5)
	for i := range signals {
		signals[i] = neutralSignal(float64(i + 1))
	}
	_, confidence, _ := computeScore(signals)

	if confidence != ConfidenceHigh {
		t.Errorf("5 signals: want confidence=high, got %s", confidence)
	}
}

func TestComputeScore_ConfidenceMedium(t *testing.T) {
	signals := []InteractionSignal{neutralSignal(1), neutralSignal(2), neutralSignal(3)}
	_, confidence, _ := computeScore(signals)

	if confidence != ConfidenceMedium {
		t.Errorf("3 signals: want confidence=medium, got %s", confidence)
	}
}

func TestComputeScore_ConfidenceLow(t *testing.T) {
	signals := []InteractionSignal{neutralSignal(1)}
	_, confidence, _ := computeScore(signals)

	if confidence != ConfidenceLow {
		t.Errorf("1 signal: want confidence=low, got %s", confidence)
	}
}

func TestComputeScore_ScoreClamped(t *testing.T) {
	// Many strong positive signals should never exceed 1.0
	signals := make([]InteractionSignal, 20)
	for i := range signals {
		signals[i] = positiveSignal(0.1)
	}
	score, _, _ := computeScore(signals)

	if score > 1.0 || score < 0.0 {
		t.Errorf("score out of [0,1] bounds: got %f", score)
	}
}

// --- TrustEngine integration tests ---

func TestTrustEngine_ScoreCallsRepo(t *testing.T) {
	repo := &fakeTrustRepo{}
	e := NewTrustEngine(repo)

	signals := []InteractionSignal{positiveSignal(1), positiveSignal(2)}
	if err := e.Score(context.Background(), "mem-1", signals); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.calls) != 1 {
		t.Fatalf("expected 1 UpsertTrustScore call, got %d", len(repo.calls))
	}
	got := repo.calls[0]
	if got.memoryID != "mem-1" {
		t.Errorf("memoryID: want mem-1, got %s", got.memoryID)
	}
	if got.score <= 0.5 {
		t.Errorf("positive signals: expected score > 0.5, got %f", got.score)
	}
	if got.confidence != ConfidenceMedium {
		t.Errorf("2 signals: expected confidence=medium, got %s", got.confidence)
	}
}

func TestTrustEngine_RepoErrorPropagated(t *testing.T) {
	repoErr := errors.New("db unavailable")
	repo := &fakeTrustRepo{err: repoErr}
	e := NewTrustEngine(repo)

	err := e.Score(context.Background(), "mem-1", []InteractionSignal{positiveSignal(1)})
	if !errors.Is(err, repoErr) {
		t.Errorf("expected repo error to propagate, got: %v", err)
	}
}
