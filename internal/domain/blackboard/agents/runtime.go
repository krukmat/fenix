package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/blackboard"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

const (
	SignalAgentID   = "blackboard-signal-agent"
	EvidenceAgentID = "blackboard-evidence-agent"
	PolicyAgentID   = "blackboard-policy-agent"
)

type artifactPayload struct {
	Contributor     string               `json:"contributor"`
	ArtifactType    string               `json:"artifact_type"`
	SourceEventID   string               `json:"source_event_id"`
	SourceEventType blackboard.EventType `json:"source_event_type"`
	SourceAgentID   *string              `json:"source_agent_id,omitempty"`
	Summary         string               `json:"summary"`
}

type specializedAgent interface {
	start(context.Context, *Runtime, *blackboard.Attachment)
}

// Runtime owns the specialized blackboard subscribers for one attachment.
type Runtime struct {
	attachment *blackboard.Attachment
	wg         sync.WaitGroup
	closeOnce  sync.Once
}

// Start wires all specialized agents onto the workspace bus for this attachment.
// ctx is propagated into every event handler so cancellation is respected.
func Start(ctx context.Context, attachment *blackboard.Attachment) *Runtime {
	if attachment == nil || attachment.Bus == nil || attachment.Memory == nil || attachment.Timeline == nil {
		return nil
	}

	rt := &Runtime{attachment: attachment}
	for _, agent := range []specializedAgent{
		NewSignalAgent(),
		NewEvidenceAgent(),
		NewPolicyAgent(),
	} {
		agent.start(ctx, rt, attachment)
	}
	return rt
}

// Close shuts down subscribers and waits for in-flight event handling to finish.
func (rt *Runtime) Close() {
	if rt == nil {
		return
	}

	rt.closeOnce.Do(func() {
		rt.attachment.Close()
		rt.wg.Wait()
	})
}

func (rt *Runtime) launch(ch <-chan blackboard.ReasoningEvent, handle func(context.Context, blackboard.ReasoningEvent), ctx context.Context) {
	rt.wg.Add(1)
	go func() {
		defer rt.wg.Done()
		for event := range ch {
			handle(ctx, event)
		}
	}()
}

func buildArtifactPayload(contributor, artifactType, summary string, source blackboard.ReasoningEvent) artifactPayload {
	return artifactPayload{
		Contributor:     contributor,
		ArtifactType:    artifactType,
		SourceEventID:   source.ID,
		SourceEventType: source.EventType,
		SourceAgentID:   source.ActorAgentID,
		Summary:         summary,
	}
}

func persistDerivedArtifact(
	ctx context.Context,
	attachment *blackboard.Attachment,
	actorID string,
	eventType blackboard.EventType,
	memoryKey string,
	payload artifactPayload,
) {
	raw, err := json.Marshal(payload)
	if err != nil {
		log.Printf("blackboard specialized agents: marshal payload actor=%s: %v", actorID, err)
		return
	}

	now := time.Now().UTC()
	actor := actorID
	if err := attachment.Timeline.Append(ctx, blackboard.ReasoningEvent{
		ID:                   uuid.NewV7().String(),
		CognitiveWorkspaceID: attachment.CognitiveWorkspaceID,
		ActorAgentID:         &actor,
		EventType:            eventType,
		Payload:              raw,
		CreatedAt:            now,
	}); err != nil {
		log.Printf("blackboard specialized agents: append timeline actor=%s: %v", actorID, err)
		return
	}

	if err := attachment.Memory.Set(ctx, blackboard.AgentMemory{
		ID:                   uuid.NewV7().String(),
		CognitiveWorkspaceID: attachment.CognitiveWorkspaceID,
		Key:                  memoryKey,
		Value:                raw,
		Scope:                blackboard.MemoryScopeSession,
		CreatedAt:            now,
		UpdatedAt:            now,
	}); err != nil {
		log.Printf("blackboard specialized agents: set memory actor=%s: %v", actorID, err)
	}
}

func memoryKeyFor(actorID string) string {
	return fmt.Sprintf("specialized_agents/%s/last_artifact", actorID)
}
