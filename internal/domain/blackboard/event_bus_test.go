// Tests for WorkspaceBus (Task A.2)
package blackboard_test

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/blackboard"
	isqlite "github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	_ "modernc.org/sqlite"
)

func setupBlackboardDB(t *testing.T) (*sql.DB, string) {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := isqlite.MigrateUp(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	_, err = db.Exec(`INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-bus', 'Bus WS', 'bus-ws', datetime('now'), datetime('now'))`)
	if err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	_, err = db.Exec(`INSERT INTO cognitive_workspace (id, workspace_id, status, created_at)
		VALUES ('cw-bus', 'ws-bus', 'active', datetime('now'))`)
	if err != nil {
		t.Fatalf("cognitive_workspace insert: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })
	return db, "cw-bus"
}

func TestWorkspaceBus_PublishPersistsToReasoningEvent(t *testing.T) {
	db, cwID := setupBlackboardDB(t)
	bus := blackboard.NewWorkspaceBus(cwID, db)
	defer bus.Close()

	evt := blackboard.ReasoningEvent{
		ID:                   "re-persist",
		CognitiveWorkspaceID: cwID,
		EventType:            blackboard.EventTypeObservation,
		Payload:              []byte(`{"note":"test"}`),
		CreatedAt:            time.Now().UTC(),
	}

	if err := bus.Publish(context.Background(), evt); err != nil {
		t.Fatalf("Publish() error = %v; want nil", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM reasoning_event WHERE id = ?", evt.ID).Scan(&count); err != nil {
		t.Fatalf("query reasoning_event: %v", err)
	}
	if count != 1 {
		t.Errorf("reasoning_event count = %d; want 1 after Publish", count)
	}
}

func TestWorkspaceBus_PublishDeliversToSubscriber(t *testing.T) {
	db, cwID := setupBlackboardDB(t)
	bus := blackboard.NewWorkspaceBus(cwID, db)
	defer bus.Close()

	ch := bus.Subscribe(blackboard.EventTypeObservation)

	evt := blackboard.ReasoningEvent{
		ID:                   "re-deliver",
		CognitiveWorkspaceID: cwID,
		EventType:            blackboard.EventTypeObservation,
		Payload:              []byte(`{}`),
		CreatedAt:            time.Now().UTC(),
	}

	if err := bus.Publish(context.Background(), evt); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	select {
	case received := <-ch:
		if received.ID != evt.ID {
			t.Errorf("received event ID = %q; want %q", received.ID, evt.ID)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("timeout: subscriber did not receive event within 200ms")
	}
}

func TestWorkspaceBus_MultipleSubscribers_AllReceive(t *testing.T) {
	db, cwID := setupBlackboardDB(t)
	bus := blackboard.NewWorkspaceBus(cwID, db)
	defer bus.Close()

	ch1 := bus.Subscribe(blackboard.EventTypeHypothesis)
	ch2 := bus.Subscribe(blackboard.EventTypeHypothesis)

	evt := blackboard.ReasoningEvent{
		ID:                   "re-multi",
		CognitiveWorkspaceID: cwID,
		EventType:            blackboard.EventTypeHypothesis,
		Payload:              []byte(`{}`),
		CreatedAt:            time.Now().UTC(),
	}

	if err := bus.Publish(context.Background(), evt); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	for i, ch := range []<-chan blackboard.ReasoningEvent{ch1, ch2} {
		select {
		case received := <-ch:
			if received.ID != evt.ID {
				t.Errorf("subscriber %d: received ID = %q; want %q", i, received.ID, evt.ID)
			}
		case <-time.After(200 * time.Millisecond):
			t.Errorf("subscriber %d: timeout waiting for event", i)
		}
	}
}

func TestWorkspaceBus_DifferentEventTypes_NoInterference(t *testing.T) {
	db, cwID := setupBlackboardDB(t)
	bus := blackboard.NewWorkspaceBus(cwID, db)
	defer bus.Close()

	chObs := bus.Subscribe(blackboard.EventTypeObservation)
	chRisk := bus.Subscribe(blackboard.EventTypeRisk)

	evt := blackboard.ReasoningEvent{
		ID:                   "re-noint",
		CognitiveWorkspaceID: cwID,
		EventType:            blackboard.EventTypeObservation,
		Payload:              []byte(`{}`),
		CreatedAt:            time.Now().UTC(),
	}

	if err := bus.Publish(context.Background(), evt); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	select {
	case <-chObs:
		// correct
	case <-time.After(200 * time.Millisecond):
		t.Error("observation subscriber: timeout waiting for event")
	}

	// risk channel must remain empty
	select {
	case unexpected := <-chRisk:
		t.Errorf("risk subscriber received unexpected event: %v", unexpected)
	default:
		// correct — no event on wrong type channel
	}
}

func TestWorkspaceBus_NonBlocking_FullBuffer(t *testing.T) {
	db, cwID := setupBlackboardDB(t)
	bus := blackboard.NewWorkspaceBus(cwID, db)
	defer bus.Close()

	// Subscribe but never consume
	_ = bus.Subscribe(blackboard.EventTypeIntent)

	done := make(chan struct{})
	go func() {
		for i := 0; i < 120; i++ {
			_ = bus.Publish(context.Background(), blackboard.ReasoningEvent{
				ID:                   "re-overflow-" + itoa(i),
				CognitiveWorkspaceID: cwID,
				EventType:            blackboard.EventTypeIntent,
				Payload:              []byte(`{}`),
				CreatedAt:            time.Now().UTC(),
			})
		}
		close(done)
	}()

	select {
	case <-done:
		// correct — Publish never blocked
	case <-time.After(2 * time.Second):
		t.Error("Publish blocked when subscriber buffer was full (should be non-blocking)")
	}
}

func TestWorkspaceBus_Close_DrainsSubs(t *testing.T) {
	db, cwID := setupBlackboardDB(t)
	bus := blackboard.NewWorkspaceBus(cwID, db)

	ch := bus.Subscribe(blackboard.EventTypeRisk)
	bus.Close()

	// Channel must be closed after bus.Close()
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("channel still open after bus.Close(); want closed")
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("channel not closed after bus.Close()")
	}
}

func TestWorkspaceBus_PublishInvalidEventType_Rejected(t *testing.T) {
	db, cwID := setupBlackboardDB(t)
	bus := blackboard.NewWorkspaceBus(cwID, db)
	defer bus.Close()

	evt := blackboard.ReasoningEvent{
		ID:                   "re-invalid",
		CognitiveWorkspaceID: cwID,
		EventType:            "invalid_type",
		Payload:              []byte(`{}`),
		CreatedAt:            time.Now().UTC(),
	}

	err := bus.Publish(context.Background(), evt)
	if err == nil {
		t.Error("Publish() with invalid event_type returned nil; want error")
	}
}

func TestWorkspaceBus_PublishAfterClose_ReturnsErrBusClosed(t *testing.T) {
	db, cwID := setupBlackboardDB(t)
	bus := blackboard.NewWorkspaceBus(cwID, db)
	bus.Close()

	evt := blackboard.ReasoningEvent{
		ID:                   "re-after-close",
		CognitiveWorkspaceID: cwID,
		EventType:            blackboard.EventTypeObservation,
		Payload:              []byte(`{}`),
		CreatedAt:            time.Now().UTC(),
	}

	err := bus.Publish(context.Background(), evt)
	if err == nil {
		t.Fatal("Publish() after Close() returned nil; want ErrBusClosed")
	}
	if err != blackboard.ErrBusClosed {
		t.Errorf("Publish() after Close() = %v; want ErrBusClosed", err)
	}

	// Confirm nothing was written to DB after Close.
	var count int
	if scanErr := db.QueryRow("SELECT COUNT(*) FROM reasoning_event WHERE id = ?", evt.ID).Scan(&count); scanErr != nil {
		t.Fatalf("query reasoning_event: %v", scanErr)
	}
	if count != 0 {
		t.Errorf("reasoning_event count = %d; want 0 after Publish post-Close", count)
	}
}

func TestWorkspaceBus_ConcurrentPublishAndClose_NoRaceNoPanic(t *testing.T) {
	db, cwID := setupBlackboardDB(t)
	bus := blackboard.NewWorkspaceBus(cwID, db)
	_ = bus.Subscribe(blackboard.EventTypeObservation)

	const publishers = 100
	startGun := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < publishers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-startGun
			_ = bus.Publish(context.Background(), blackboard.ReasoningEvent{
				ID:                   "re-race-" + itoa(i),
				CognitiveWorkspaceID: cwID,
				EventType:            blackboard.EventTypeObservation,
				Payload:              []byte(`{}`),
				CreatedAt:            time.Now().UTC(),
			})
		}(i)
	}

	close(startGun)              // all goroutines race to publish
	time.Sleep(time.Millisecond) // let some publish before Close
	bus.Close()
	wg.Wait() // must complete without panic or data race
}

// itoa is a minimal int-to-string helper to avoid importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
