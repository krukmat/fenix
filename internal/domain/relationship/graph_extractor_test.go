// Task B.4 — GraphExtractor unit tests.
// No real DB, no real event bus connection — all faked via fakeGraphRepo and eventbus.New().
package relationship

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
)

// --- Test double ---

type upsertEdgeArgs struct {
	workspaceID    string
	fromEntityType string
	fromEntityID   string
	toEntityType   string
	toEntityID     string
	influenceType  InfluenceType
	strength       float64
}

type fakeGraphRepo struct {
	calls []upsertEdgeArgs
	err   error
}

func (f *fakeGraphRepo) UpsertEdge(
	_ context.Context,
	workspaceID, fromEntityType, fromEntityID, toEntityType, toEntityID string,
	influenceType InfluenceType,
	strength float64,
) error {
	f.calls = append(f.calls, upsertEdgeArgs{
		workspaceID:    workspaceID,
		fromEntityType: fromEntityType,
		fromEntityID:   fromEntityID,
		toEntityType:   toEntityType,
		toEntityID:     toEntityID,
		influenceType:  influenceType,
		strength:       strength,
	})
	return f.err
}

// --- Helper ---

func makeApprovalEvent(topic, workspaceID, actorID, entityType, entityID string, extra map[string]any) eventbus.Event {
	payload := map[string]any{
		"workspace_id": workspaceID,
		"actor_id":     actorID,
		"entity_type":  entityType,
		"entity_id":    entityID,
	}
	for k, v := range extra {
		payload[k] = v
	}
	return eventbus.Event{Topic: topic, Payload: payload}
}

// --- Tests ---

// TestGraphExtractor_HandleApprovalRequested verifies that an approval.requested event
// produces an UpsertEdge call with InfluenceApproves and strength 0.6.
func TestGraphExtractor_HandleApprovalRequested(t *testing.T) {
	repo := &fakeGraphRepo{}
	ge := NewGraphExtractor(eventbus.New(), repo)

	ev := makeApprovalEvent(TopicApprovalRequested, "ws-1", "user-99", "deal", "deal-1", nil)
	ge.handle(context.Background(), ev)

	if len(repo.calls) != 1 {
		t.Fatalf("expected 1 UpsertEdge call, got %d", len(repo.calls))
	}
	got := repo.calls[0]
	if got.influenceType != InfluenceApproves {
		t.Errorf("influenceType: want %q, got %q", InfluenceApproves, got.influenceType)
	}
	if got.strength != 0.6 {
		t.Errorf("strength: want 0.6, got %v", got.strength)
	}
	if got.fromEntityID != "user-99" {
		t.Errorf("fromEntityID: want %q, got %q", "user-99", got.fromEntityID)
	}
	if got.fromEntityType != "user" {
		t.Errorf("fromEntityType: want %q, got %q", "user", got.fromEntityType)
	}
	if got.toEntityType != "deal" {
		t.Errorf("toEntityType: want %q, got %q", "deal", got.toEntityType)
	}
	if got.toEntityID != "deal-1" {
		t.Errorf("toEntityID: want %q, got %q", "deal-1", got.toEntityID)
	}
	if got.workspaceID != "ws-1" {
		t.Errorf("workspaceID: want %q, got %q", "ws-1", got.workspaceID)
	}
}

// TestGraphExtractor_HandleApprovalDecidedApproved verifies that an approved decision
// produces an UpsertEdge call with strength 0.9.
func TestGraphExtractor_HandleApprovalDecidedApproved(t *testing.T) {
	repo := &fakeGraphRepo{}
	ge := NewGraphExtractor(eventbus.New(), repo)

	ev := makeApprovalEvent(TopicApprovalDecided, "ws-1", "user-99", "case", "case-1",
		map[string]any{"status": "approved"})
	ge.handle(context.Background(), ev)

	if len(repo.calls) != 1 {
		t.Fatalf("expected 1 UpsertEdge call, got %d", len(repo.calls))
	}
	if got := repo.calls[0].strength; got != 0.9 {
		t.Errorf("strength: want 0.9, got %v", got)
	}
	if got := repo.calls[0].influenceType; got != InfluenceApproves {
		t.Errorf("influenceType: want %q, got %q", InfluenceApproves, got)
	}
}

// TestGraphExtractor_HandleApprovalDecidedRejected verifies that a rejected decision
// does NOT produce an UpsertEdge call (Option A — rejects are not influence signals).
func TestGraphExtractor_HandleApprovalDecidedRejected(t *testing.T) {
	repo := &fakeGraphRepo{}
	ge := NewGraphExtractor(eventbus.New(), repo)

	ev := makeApprovalEvent(TopicApprovalDecided, "ws-1", "user-99", "deal", "deal-1",
		map[string]any{"status": "rejected"})
	ge.handle(context.Background(), ev)

	if len(repo.calls) != 0 {
		t.Errorf("expected 0 UpsertEdge calls on rejected, got %d", len(repo.calls))
	}
}

// TestGraphExtractor_HandleApprovalDecidedCancelled verifies cancelled decisions are skipped.
func TestGraphExtractor_HandleApprovalDecidedCancelled(t *testing.T) {
	repo := &fakeGraphRepo{}
	ge := NewGraphExtractor(eventbus.New(), repo)

	ev := makeApprovalEvent(TopicApprovalDecided, "ws-1", "user-99", "deal", "deal-1",
		map[string]any{"status": "cancelled"})
	ge.handle(context.Background(), ev)

	if len(repo.calls) != 0 {
		t.Errorf("expected 0 UpsertEdge calls on cancelled, got %d", len(repo.calls))
	}
}

// TestGraphExtractor_MissingPayloadSkips verifies that a malformed payload
// does not call UpsertEdge and does not panic.
func TestGraphExtractor_MissingPayloadSkips(t *testing.T) {
	repo := &fakeGraphRepo{}
	ge := NewGraphExtractor(eventbus.New(), repo)

	// nil payload
	ge.handle(context.Background(), eventbus.Event{Topic: TopicApprovalRequested, Payload: nil})

	// string payload (wrong type)
	ge.handle(context.Background(), eventbus.Event{Topic: TopicApprovalRequested, Payload: "bad"})

	if len(repo.calls) != 0 {
		t.Errorf("expected 0 UpsertEdge calls on malformed payload, got %d", len(repo.calls))
	}
}

// TestGraphExtractor_RepoErrorLogsAndContinues verifies that a repo error does not panic
// and that Run continues processing subsequent events.
func TestGraphExtractor_RepoErrorLogsAndContinues(t *testing.T) {
	repo := &fakeGraphRepo{err: errors.New("db down")}
	ge := NewGraphExtractor(eventbus.New(), repo)

	ev := makeApprovalEvent(TopicApprovalRequested, "ws-1", "user-1", "deal", "deal-1", nil)
	// must not panic
	ge.handle(context.Background(), ev)

	// call was attempted (repo logged the error), no second call needed
	if len(repo.calls) != 1 {
		t.Errorf("expected 1 UpsertEdge attempt, got %d", len(repo.calls))
	}
}

// TestGraphExtractor_StopsOnContextCancel verifies that Run exits within 200ms after cancel.
func TestGraphExtractor_StopsOnContextCancel(t *testing.T) {
	repo := &fakeGraphRepo{}
	ge := NewGraphExtractor(eventbus.New(), repo)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		ge.Run(ctx)
		close(done)
	}()

	cancel()
	select {
	case <-done:
		// Run exited cleanly
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Run did not stop after context cancel within 200ms")
	}
}
