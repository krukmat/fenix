package relationship

import (
	"context"
	"fmt"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
)

const lifecycleRetentionWindow = 90 * 24 * time.Hour

// PIIRedactor reuses the existing policy-engine redaction path.
type PIIRedactor interface {
	RedactPII(ctx context.Context, evidence []knowledge.Evidence) ([]knowledge.Evidence, error)
}

// LifecycleService applies privacy and retention rules to relationship-memory artifacts.
type LifecycleService struct {
	repo     LifecycleRepository
	redactor PIIRedactor
}

func NewLifecycleService(repo LifecycleRepository, redactor PIIRedactor) *LifecycleService {
	return &LifecycleService{repo: repo, redactor: redactor}
}

func (s *LifecycleService) DecayWorkspace(ctx context.Context, workspaceID string, now time.Time) error {
	cutoff := now.UTC().Add(-lifecycleRetentionWindow)

	memories, err := s.repo.ListStaleMemories(ctx, workspaceID, cutoff)
	if err != nil {
		return fmt.Errorf("lifecycle decay memories: %w", err)
	}
	decayErr := s.decayMemories(ctx, memories)
	if decayErr != nil {
		return decayErr
	}

	signals, err := s.repo.ListStaleSignals(ctx, workspaceID, cutoff)
	if err != nil {
		return fmt.Errorf("lifecycle decay signals: %w", err)
	}
	return s.decaySignals(ctx, signals)
}

func (s *LifecycleService) EraseEntityMemory(ctx context.Context, workspaceID string, entityType EntityType, entityID string) error {
	if entityType == "" || entityID == "" {
		return fmt.Errorf("entityType and entityID are required")
	}
	if err := s.repo.EraseEntityArtifacts(ctx, workspaceID, entityType, entityID); err != nil {
		return fmt.Errorf("lifecycle erase entity memory: %w", err)
	}
	return nil
}

func (s *LifecycleService) RedactSignalSummary(ctx context.Context, summary string) (string, bool, error) {
	if summary == "" {
		return "", false, nil
	}

	evidence := []knowledge.Evidence{{Snippet: &summary}}
	redacted, err := s.redactor.RedactPII(ctx, evidence)
	if err != nil {
		return "", false, fmt.Errorf("redact signal summary: %w", err)
	}
	if len(redacted) == 0 || redacted[0].Snippet == nil {
		return summary, false, nil
	}

	next := *redacted[0].Snippet
	return next, redacted[0].PiiRedacted && next != summary, nil
}

func (s *LifecycleService) decayMemories(ctx context.Context, memories []Memory) error {
	for _, item := range memories {
		summary, changed, redactErr := s.RedactSignalSummary(ctx, item.Summary)
		if redactErr != nil {
			return fmt.Errorf("lifecycle redact memory %s: %w", item.ID, redactErr)
		}
		if !changed {
			continue
		}
		updateErr := s.repo.UpdateMemorySummary(ctx, item.ID, summary)
		if updateErr != nil {
			return fmt.Errorf("lifecycle update memory %s: %w", item.ID, updateErr)
		}
	}
	return nil
}

func (s *LifecycleService) decaySignals(ctx context.Context, signals []InteractionSignal) error {
	for _, item := range signals {
		summary, changed, redactErr := s.RedactSignalSummary(ctx, item.Summary)
		if redactErr != nil {
			return fmt.Errorf("lifecycle redact signal %s: %w", item.ID, redactErr)
		}
		if !changed {
			continue
		}
		updateErr := s.repo.UpdateSignalSummary(ctx, item.ID, summary)
		if updateErr != nil {
			return fmt.Errorf("lifecycle update signal %s: %w", item.ID, updateErr)
		}
	}
	return nil
}
