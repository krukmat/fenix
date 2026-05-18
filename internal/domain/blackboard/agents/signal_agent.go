package agents

import (
	"context"

	"github.com/matiasleandrokruk/fenix/internal/domain/blackboard"
)

// SignalAgent derives signal hypotheses from workspace observations.
type SignalAgent struct{}

func NewSignalAgent() *SignalAgent {
	return &SignalAgent{}
}

func (a *SignalAgent) start(ctx context.Context, rt *Runtime, attachment *blackboard.Attachment) {
	ch := attachment.Bus.Subscribe(blackboard.EventTypeObservation)
	rt.launch(ch, func(ctx context.Context, event blackboard.ReasoningEvent) {
		payload := buildArtifactPayload(
			SignalAgentID,
			"signal_hypothesis",
			"Derived signal hypothesis from workspace observation.",
			event,
		)
		persistDerivedArtifact(
			ctx,
			attachment,
			SignalAgentID,
			blackboard.EventTypeHypothesis,
			memoryKeyFor(SignalAgentID),
			payload,
		)
	}, ctx)
}
