// Package relationship — Task B.4: GraphExtractor service.
// Subscribes to approval.requested and approval.decided events, infers directed
// influence edges between CRM entities, and persists them via GraphRepository.
// No LLM call, no HTTP — pattern matching only.
package relationship

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
)

// Approval topic constants — copied from internal/domain/audit/service.go (unexported there).
// Do NOT import the audit package: it would create a circular dependency between domain layers.
const (
	TopicApprovalRequested = "approval.requested"
	TopicApprovalDecided   = "approval.decided"
	graphPayloadTypeErrFmt = "payload is not map[string]any: %T"
	graphWorkspaceIDKey    = "workspace_id"
)

// GraphRepository is the persistence contract for the GraphExtractor.
// UpsertEdge must be idempotent: same (workspace_id, from*, to*, influence_type) → update, not duplicate.
type GraphRepository interface {
	UpsertEdge(ctx context.Context,
		workspaceID, fromEntityType, fromEntityID,
		toEntityType, toEntityID string,
		influenceType InfluenceType,
		strength float64,
	) error
}

// graphInput holds the fields extracted from an approval event payload.
type graphInput struct {
	workspaceID    string
	fromEntityType string // always "user" — actor is a CRM user
	fromEntityID   string // actor_id from payload
	toEntityType   string // entity_type from payload
	toEntityID     string // entity_id from payload
}

// GraphExtractor subscribes to approval events and writes stakeholder graph edges.
// Task B.4: long-running goroutine, independent of Summarizer and TrustEngine.
type GraphExtractor struct {
	bus  eventbus.EventBus
	repo GraphRepository
}

// NewGraphExtractor constructs a GraphExtractor with its two required dependencies.
func NewGraphExtractor(bus eventbus.EventBus, repo GraphRepository) *GraphExtractor {
	return &GraphExtractor{bus: bus, repo: repo}
}

// Run subscribes to both approval topics and processes events until ctx is cancelled.
func (g *GraphExtractor) Run(ctx context.Context) {
	chRequested := g.bus.Subscribe(TopicApprovalRequested)
	chDecided := g.bus.Subscribe(TopicApprovalDecided)

	for {
		select {
		case ev := <-chRequested:
			g.handle(ctx, ev)
		case ev := <-chDecided:
			g.handle(ctx, ev)
		case <-ctx.Done():
			return
		}
	}
}

// handle extracts the graph input from the event, checks approval gate for decided events,
// then calls UpsertEdge. Errors are logged and execution continues (best-effort).
func (g *GraphExtractor) handle(ctx context.Context, ev eventbus.Event) {
	input, err := parseGraphPayload(ev)
	if err != nil {
		log.Printf("relationship.GraphExtractor: parse payload topic=%s err=%v", ev.Topic, err)
		return
	}

	// Option A: for approval.decided, skip edge if not approved.
	if ev.Topic == TopicApprovalDecided && !isApproved(ev.Payload) {
		return
	}

	influenceType := influenceTypeFor(ev.Topic)
	strength := strengthFor(ev.Topic)

	if upsertErr := g.repo.UpsertEdge(ctx,
		input.workspaceID,
		input.fromEntityType, input.fromEntityID,
		input.toEntityType, input.toEntityID,
		influenceType, strength,
	); upsertErr != nil {
		log.Printf("relationship.GraphExtractor: UpsertEdge topic=%s err=%v", ev.Topic, upsertErr)
	}
}

// parseGraphPayload extracts structured fields from an approval event's map[string]any payload.
func parseGraphPayload(ev eventbus.Event) (graphInput, error) {
	m, ok := ev.Payload.(map[string]any)
	if !ok {
		return graphInput{}, fmt.Errorf(graphPayloadTypeErrFmt, ev.Payload)
	}

	str := func(key string) string {
		v, _ := m[key].(string)
		return v
	}

	workspaceID := str(graphWorkspaceIDKey)
	actorID := str("actor_id")
	entityType := str("entity_type")
	entityID := str("entity_id")

	if workspaceID == "" || actorID == "" {
		return graphInput{}, fmt.Errorf("payload missing workspace_id or actor_id")
	}

	return graphInput{
		workspaceID:    workspaceID,
		fromEntityType: "user",
		fromEntityID:   actorID,
		toEntityType:   entityType,
		toEntityID:     entityID,
	}, nil
}

// influenceTypeFor maps an approval topic to its InfluenceType.
// Both topics map to InfluenceApproves — strength differentiates intent vs. decision weight.
func influenceTypeFor(topic string) InfluenceType {
	switch topic {
	case TopicApprovalRequested, TopicApprovalDecided:
		return InfluenceApproves
	default:
		return InfluenceInfluences
	}
}

// strengthFor maps an approval topic to its edge strength in [0.0, 1.0].
func strengthFor(topic string) float64 {
	switch topic {
	case TopicApprovalRequested:
		return 0.6
	case TopicApprovalDecided:
		return 0.9
	default:
		return 0.5
	}
}

// isApproved returns true only when the payload decision/status/outcome field
// is "approved" or "approve". Any other value (rejected, cancelled, expired) → false.
func isApproved(payload any) bool {
	m, ok := payload.(map[string]any)
	if !ok {
		return false
	}
	for _, key := range []string{"status", "decision", "outcome"} {
		if val, exists := m[key]; exists {
			if s, isStr := val.(string); isStr {
				normalized := strings.ToLower(strings.TrimSpace(s))
				return normalized == "approved" || normalized == "approve"
			}
		}
	}
	return false
}
