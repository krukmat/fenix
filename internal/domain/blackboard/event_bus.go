// Package blackboard — workspace-scoped event bus (Task A.2, ADR-100).
// Dual-path: persists each event to reasoning_event table AND delivers in-memory.
package blackboard

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sync"
)

const (
	busBufferSize    = 100
	emptyJSONPayload = "{}"
)

// ErrBusClosed is returned by Publish when the bus has already been closed.
var ErrBusClosed = errors.New("workspace bus closed")

// WorkspaceBus is the interface for the workspace-scoped publish/subscribe bus.
type WorkspaceBus interface {
	Publish(ctx context.Context, event ReasoningEvent) error
	Subscribe(eventType EventType) <-chan ReasoningEvent
	Close()
}

// workspaceBus is the concrete implementation of WorkspaceBus.
type workspaceBus struct {
	cognitiveWorkspaceID string
	db                   *sql.DB

	mu          sync.RWMutex
	subscribers map[EventType][]chan ReasoningEvent
	closed      bool
}

// NewWorkspaceBus creates a new bus scoped to cognitiveWorkspaceID.
func NewWorkspaceBus(cognitiveWorkspaceID string, db *sql.DB) WorkspaceBus {
	return &workspaceBus{
		cognitiveWorkspaceID: cognitiveWorkspaceID,
		db:                   db,
		subscribers:          make(map[EventType][]chan ReasoningEvent),
	}
}

// Publish persists the event to reasoning_event and delivers it to all subscribers.
// Returns ErrBusClosed if the bus is already closed. In-memory delivery is best-effort (drops on full buffer).
func (b *workspaceBus) Publish(ctx context.Context, event ReasoningEvent) error {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return ErrBusClosed
	}
	b.mu.RUnlock()

	if err := b.persist(ctx, event); err != nil {
		return fmt.Errorf("workspace bus publish: %w", err)
	}
	b.deliver(event)
	return nil
}

// Subscribe returns a read-only channel that receives events of the given type.
// The caller must consume the channel to avoid blocking future deliveries.
func (b *workspaceBus) Subscribe(eventType EventType) <-chan ReasoningEvent {
	ch := make(chan ReasoningEvent, busBufferSize)
	b.mu.Lock()
	b.subscribers[eventType] = append(b.subscribers[eventType], ch)
	b.mu.Unlock()
	return ch
}

// Close drains and closes all subscriber channels.
func (b *workspaceBus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	for _, chans := range b.subscribers {
		for _, ch := range chans {
			close(ch)
		}
	}
	b.subscribers = make(map[EventType][]chan ReasoningEvent)
}

// persist writes the event to the reasoning_event table.
func (b *workspaceBus) persist(ctx context.Context, event ReasoningEvent) error {
	payload := event.Payload
	if len(payload) == 0 {
		payload = []byte(emptyJSONPayload)
	}

	_, err := b.db.ExecContext(ctx,
		`INSERT INTO reasoning_event (id, cognitive_workspace_id, actor_agent_id, event_type, payload, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		event.ID,
		event.CognitiveWorkspaceID,
		event.ActorAgentID,
		string(event.EventType),
		string(payload),
		event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert reasoning_event: %w", err)
	}
	return nil
}

// deliver fans out the event to all subscribers of the matching event type.
// Holds RLock for the entire fan-out to prevent Close() from closing channels mid-send.
// Drops silently on full buffer with a warning log.
func (b *workspaceBus) deliver(event ReasoningEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closed {
		return
	}
	for _, ch := range b.subscribers[event.EventType] {
		select {
		case ch <- event:
		default:
			log.Printf("workspace bus: subscriber buffer full for event_type=%s workspace=%s — event dropped",
				event.EventType, b.cognitiveWorkspaceID)
		}
	}
}
