// Task 2.2: Unit tests for the in-memory event bus.
package eventbus

import (
	"testing"
	"time"
)

func TestEventBus_PublishAndSubscribe(t *testing.T) {
	bus := New()
	ch := bus.Subscribe("test.topic")

	bus.Publish("test.topic", "hello")

	select {
	case evt := <-ch:
		if evt.Topic != "test.topic" {
			t.Errorf("expected topic 'test.topic', got %q", evt.Topic)
		}
		if evt.Payload != "hello" {
			t.Errorf("expected payload 'hello', got %v", evt.Payload)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout: expected event to be received within 100ms")
	}
}

func TestEventBus_MultipleSubscribers_AllReceive(t *testing.T) {
	bus := New()
	ch1 := bus.Subscribe("multi.topic")
	ch2 := bus.Subscribe("multi.topic")

	bus.Publish("multi.topic", 42)

	for i, ch := range []<-chan Event{ch1, ch2} {
		select {
		case evt := <-ch:
			if evt.Payload != 42 {
				t.Errorf("subscriber %d: expected payload 42, got %v", i, evt.Payload)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("subscriber %d: timeout waiting for event", i)
		}
	}
}

func TestEventBus_DifferentTopics_NoInterference(t *testing.T) {
	bus := New()
	chA := bus.Subscribe("topic.a")
	chB := bus.Subscribe("topic.b")

	bus.Publish("topic.a", "for-a")

	select {
	case evt := <-chA:
		if evt.Payload != "for-a" {
			t.Errorf("topic.a: unexpected payload %v", evt.Payload)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("topic.a: timeout waiting for event")
	}

	// topic.b should have received nothing
	select {
	case evt := <-chB:
		t.Errorf("topic.b: received unexpected event: %v", evt)
	default:
		// correct — no event
	}
}

func TestEventBus_NonBlockingPublish_FullBuffer(t *testing.T) {
	bus := New()
	// Subscribe but never consume — buffer will fill up
	_ = bus.Subscribe("overflow.topic")

	// Publish more events than the buffer size — must not block
	done := make(chan struct{})
	go func() {
		for i := 0; i <= defaultBufferSize+10; i++ {
			bus.Publish("overflow.topic", i)
		}
		close(done)
	}()

	select {
	case <-done:
		// correct — publish never blocked
	case <-time.After(500 * time.Millisecond):
		t.Error("Publish blocked when buffer was full (should be non-blocking)")
	}
}
