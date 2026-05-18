package agents

import (
	"context"

	"github.com/matiasleandrokruk/fenix/internal/domain/blackboard"
)

// PolicyAgent derives policy constraints from workspace observations.
type PolicyAgent struct{}

func NewPolicyAgent() *PolicyAgent {
	return &PolicyAgent{}
}

func (a *PolicyAgent) start(ctx context.Context, rt *Runtime, attachment *blackboard.Attachment) {
	ch := attachment.Bus.Subscribe(blackboard.EventTypeObservation)
	rt.launch(ch, func(ctx context.Context, event blackboard.ReasoningEvent) {
		payload := buildArtifactPayload(
			PolicyAgentID,
			"policy_constraint",
			"Derived policy constraint from workspace observation.",
			event,
		)
		persistDerivedArtifact(
			ctx,
			attachment,
			PolicyAgentID,
			blackboard.EventTypeRisk,
			memoryKeyFor(PolicyAgentID),
			payload,
		)
	}, ctx)
}
