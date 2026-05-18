package agents

import (
	"context"

	"github.com/matiasleandrokruk/fenix/internal/domain/blackboard"
)

// EvidenceAgent records evidence findings derived from workspace observations.
type EvidenceAgent struct{}

func NewEvidenceAgent() *EvidenceAgent {
	return &EvidenceAgent{}
}

func (a *EvidenceAgent) start(ctx context.Context, rt *Runtime, attachment *blackboard.Attachment) {
	ch := attachment.Bus.Subscribe(blackboard.EventTypeObservation)
	rt.launch(ch, func(ctx context.Context, event blackboard.ReasoningEvent) {
		payload := buildArtifactPayload(
			EvidenceAgentID,
			"evidence_finding",
			"Captured supporting evidence from workspace observation.",
			event,
		)
		persistDerivedArtifact(
			ctx,
			attachment,
			EvidenceAgentID,
			blackboard.EventTypeObservation,
			memoryKeyFor(EvidenceAgentID),
			payload,
		)
	}, ctx)
}
