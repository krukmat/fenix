// Package eventbus — Task 2.2: In-memory publish/subscribe event bus.
// Used by IngestService to notify downstream consumers (Task 2.4 embedder)
// after a knowledge item is ingested.
//
// Design:
//   - Buffered Go channel per topic (buffer=100).
//   - Publish is non-blocking: drops the event silently if the buffer is full.
//   - Subscribe returns a read-only channel; the caller owns the consumption loop.
//   - No persistence: events are fire-and-forget (MVP constraint).
//   - EventBus interface for testability.
package eventbus

import "sync"

// Event is a single published message.
type Event struct {
	Topic   string
	Payload any
}

// EventBus is the interface for publishing and subscribing to topics.
type EventBus interface {
	Publish(topic string, payload any)
	Subscribe(topic string) <-chan Event
}

const defaultBufferSize = 100

// Bus is the in-memory implementation of EventBus.
type Bus struct {
	mu          sync.RWMutex
	subscribers map[string][]chan Event
}

// New returns a new in-memory Bus.
func New() *Bus {
	return &Bus{
		subscribers: make(map[string][]chan Event),
	}
}

// Subscribe registers a new subscriber for topic and returns a read-only channel.
// The caller must consume the channel to prevent blocking on future Publish calls.
func (b *Bus) Subscribe(topic string) <-chan Event {
	ch := make(chan Event, defaultBufferSize)
	b.mu.Lock()
	b.subscribers[topic] = append(b.subscribers[topic], ch)
	b.mu.Unlock()
	return ch
}

// Publish sends an Event to all subscribers of topic.
// If a subscriber's buffer is full the event is dropped (non-blocking).
func (b *Bus) Publish(topic string, payload any) {
	evt := Event{Topic: topic, Payload: payload}
	b.mu.RLock()
	subs := b.subscribers[topic]
	b.mu.RUnlock()
	for _, ch := range subs {
		select {
		case ch <- evt:
		default:
			// buffer full — drop event (fire-and-forget)
		}
	}
}
