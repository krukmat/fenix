package agent

import (
	"context"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
)

type recordingGroundsEvidenceBuilder struct {
	pack      *knowledge.EvidencePack
	err       error
	callCount int
}

func (s *recordingGroundsEvidenceBuilder) BuildEvidencePack(_ context.Context, _ knowledge.BuildEvidencePackInput) (*knowledge.EvidencePack, error) {
	s.callCount++
	if s.err != nil {
		return nil, s.err
	}
	return s.pack, nil
}

func TestGroundsValidator(t *testing.T) {
	t.Run("grounds nil returns met true without calling evidence service", func(t *testing.T) {
		builder := &recordingGroundsEvidenceBuilder{}
		validator := NewGroundsValidator(builder)

		got, err := validator.Validate(context.Background(), nil, TriggerAgentInput{})
		if err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		if !got.Met {
			t.Fatalf("expected grounds to be met, got %#v", got)
		}
		if builder.callCount != 0 {
			t.Fatalf("expected evidence service not to be called, got %d call(s)", builder.callCount)
		}
	})

	t.Run("fails when source count is below minimum", func(t *testing.T) {
		builder := &recordingGroundsEvidenceBuilder{
			pack: &knowledge.EvidencePack{
				Sources:    []knowledge.Evidence{{ID: "ev_1"}},
				Confidence: knowledge.ConfidenceHigh,
			},
		}
		validator := NewGroundsValidator(builder)

		got, err := validator.Validate(context.Background(), &CartaGrounds{
			MinSources: 2,
		}, TriggerAgentInput{WorkspaceID: "ws_test"})
		if err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		if got.Met {
			t.Fatalf("expected grounds failure, got %#v", got)
		}
	})

	t.Run("fails when confidence is below minimum", func(t *testing.T) {
		builder := &recordingGroundsEvidenceBuilder{
			pack: &knowledge.EvidencePack{
				Sources:    []knowledge.Evidence{{ID: "ev_1"}},
				Confidence: knowledge.ConfidenceLow,
			},
		}
		validator := NewGroundsValidator(builder)

		got, err := validator.Validate(context.Background(), &CartaGrounds{
			MinSources:    1,
			MinConfidence: knowledge.ConfidenceMedium,
		}, TriggerAgentInput{WorkspaceID: "ws_test"})
		if err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		if got.Met {
			t.Fatalf("expected confidence failure, got %#v", got)
		}
	})

	t.Run("fails when evidence is stale", func(t *testing.T) {
		now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
		builder := &recordingGroundsEvidenceBuilder{
			pack: &knowledge.EvidencePack{
				Sources: []knowledge.Evidence{
					{ID: "ev_1", CreatedAt: now.Add(-48 * time.Hour)},
				},
				Confidence: knowledge.ConfidenceHigh,
			},
		}
		validator := NewGroundsValidator(builder)
		validator.now = func() time.Time { return now }

		got, err := validator.Validate(context.Background(), &CartaGrounds{
			MinSources:   1,
			MaxStaleness: 1,
			MaxAgeUnit:   "days",
		}, TriggerAgentInput{WorkspaceID: "ws_test"})
		if err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		if got.Met {
			t.Fatalf("expected staleness failure, got %#v", got)
		}
	})

	t.Run("passes when all grounds are satisfied", func(t *testing.T) {
		now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
		builder := &recordingGroundsEvidenceBuilder{
			pack: &knowledge.EvidencePack{
				Sources: []knowledge.Evidence{
					{ID: "ev_1", CreatedAt: now.Add(-2 * time.Hour)},
					{ID: "ev_2", CreatedAt: now.Add(-1 * time.Hour)},
				},
				Confidence: knowledge.ConfidenceHigh,
			},
		}
		validator := NewGroundsValidator(builder)
		validator.now = func() time.Time { return now }

		got, err := validator.Validate(context.Background(), &CartaGrounds{
			MinSources:    2,
			MinConfidence: knowledge.ConfidenceMedium,
			MaxStaleness:  1,
			MaxAgeUnit:    "days",
		}, TriggerAgentInput{WorkspaceID: "ws_test"})
		if err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		if !got.Met {
			t.Fatalf("expected grounds success, got %#v", got)
		}
	})
}
