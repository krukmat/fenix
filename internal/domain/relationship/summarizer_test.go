// Task B.2.4 — Summarizer unit tests: fake LLM + fake repository.
// No real DB, no real LLM, no network.
package relationship

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
	"github.com/matiasleandrokruk/fenix/internal/infra/llm"
)

// --- Test doubles ---

type fakeLLM struct {
	response string
	err      error
}

func (f *fakeLLM) ChatCompletion(_ context.Context, _ llm.ChatRequest) (*llm.ChatResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &llm.ChatResponse{Content: f.response}, nil
}

func (f *fakeLLM) Embed(_ context.Context, _ llm.EmbedRequest) (*llm.EmbedResponse, error) {
	return &llm.EmbedResponse{}, nil
}

func (f *fakeLLM) ModelInfo() llm.ModelMeta { return llm.ModelMeta{} }

func (f *fakeLLM) HealthCheck(_ context.Context) error { return nil }

type upsertArgs struct {
	workspaceID string
	entityType  EntityType
	entityID    string
	summary     string
}

type insertArgs struct {
	signalID         string
	memoryID         string
	signalType       SignalType
	sentiment        SentimentType
	summary          string
	sourceEntityType string
	sourceEntityID   string
	occurredAt       time.Time
}

type fakeRepo struct {
	upsertCalls []upsertArgs
	insertCalls []insertArgs
	upsertErr   error
	insertErr   error
}

func (r *fakeRepo) UpsertMemory(_ context.Context, workspaceID string, entityType EntityType, entityID, summary string) (*Memory, error) {
	r.upsertCalls = append(r.upsertCalls, upsertArgs{workspaceID, entityType, entityID, summary})
	if r.upsertErr != nil {
		return nil, r.upsertErr
	}
	return &Memory{ID: "mem-1"}, nil
}

func (r *fakeRepo) InsertSignal(_ context.Context, memoryID string, signalType SignalType, sentiment SentimentType,
	summary, sourceEntityType, sourceEntityID string, occurredAt time.Time) (string, error) {
	signalID := "sig-1"
	r.insertCalls = append(r.insertCalls, insertArgs{signalID, memoryID, signalType, sentiment, summary, sourceEntityType, sourceEntityID, occurredAt})
	return signalID, r.insertErr
}

// --- Helper ---

func makeEvent(topic, workspaceID, entityType, entityID, rawText string) eventbus.Event {
	return eventbus.Event{
		Topic: topic,
		Payload: map[string]any{
			"workspace_id":       workspaceID,
			"entity_type":        entityType,
			"entity_id":          entityID,
			"raw_text":           rawText,
			"source_entity_type": "activity",
			"source_entity_id":   "src-1",
			"occurred_at":        "2026-05-17T10:00:00Z",
		},
	}
}

// --- Tests ---

func TestSummarizer_HandleActivity(t *testing.T) {
	repo := &fakeRepo{}
	fl := &fakeLLM{response: `{"summary":"closed ticket","sentiment":"positive"}`}
	s := NewSummarizer(eventbus.New(), fl, repo)

	s.handle(context.Background(), makeEvent(TopicActivityCreated, "ws-1", "account", "acc-1", "customer called"))

	if len(repo.insertCalls) != 1 {
		t.Fatalf("expected 1 InsertSignal call, got %d", len(repo.insertCalls))
	}
	got := repo.insertCalls[0]
	if got.signalType != SignalEmail {
		t.Errorf("signalType: want %q, got %q", SignalEmail, got.signalType)
	}
	if got.sentiment != SentimentPositive {
		t.Errorf("sentiment: want %q, got %q", SentimentPositive, got.sentiment)
	}
}

func TestSummarizer_PublishesInteractionSignalCreated(t *testing.T) {
	bus := eventbus.New()
	repo := &fakeRepo{}
	fl := &fakeLLM{response: `{"summary":"closed ticket","sentiment":"positive"}`}
	s := NewSummarizer(bus, fl, repo)

	ch := bus.Subscribe(TopicInteractionSignalCreated)
	s.handle(context.Background(), makeEvent(TopicActivityCreated, "ws-1", "account", "acc-1", "customer called"))

	select {
	case ev := <-ch:
		payload, ok := ev.Payload.(map[string]any)
		if !ok {
			t.Fatalf("payload type = %T; want map[string]any", ev.Payload)
		}
		if ev.Topic != TopicInteractionSignalCreated {
			t.Fatalf("topic = %q; want %q", ev.Topic, TopicInteractionSignalCreated)
		}
		if payload["workspace_id"] != "ws-1" {
			t.Errorf("workspace_id: want %q, got %#v", "ws-1", payload["workspace_id"])
		}
		if payload["signal_id"] != "sig-1" {
			t.Errorf("signal_id: want %q, got %#v", "sig-1", payload["signal_id"])
		}
		if payload["summary"] != "closed ticket" {
			t.Errorf("summary: want %q, got %#v", "closed ticket", payload["summary"])
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected interaction_signal.created event")
	}
}

func TestSummarizer_HandleNote(t *testing.T) {
	repo := &fakeRepo{}
	fl := &fakeLLM{response: `{"summary":"note added","sentiment":"neutral"}`}
	s := NewSummarizer(eventbus.New(), fl, repo)

	s.handle(context.Background(), makeEvent(TopicNoteCreated, "ws-1", "contact", "con-1", "added a note"))

	if len(repo.insertCalls) != 1 {
		t.Fatalf("expected 1 InsertSignal call, got %d", len(repo.insertCalls))
	}
	if got := repo.insertCalls[0].signalType; got != SignalNote {
		t.Errorf("signalType: want %q, got %q", SignalNote, got)
	}
}

func TestSummarizer_HandleCaseUpdate(t *testing.T) {
	repo := &fakeRepo{}
	fl := &fakeLLM{response: `{"summary":"case resolved","sentiment":"positive"}`}
	s := NewSummarizer(eventbus.New(), fl, repo)

	s.handle(context.Background(), makeEvent(TopicCaseUpdated, "ws-1", "case", "case-1", "case was closed"))

	if len(repo.insertCalls) != 1 {
		t.Fatalf("expected 1 InsertSignal call, got %d", len(repo.insertCalls))
	}
	if got := repo.insertCalls[0].signalType; got != SignalCaseUpdate {
		t.Errorf("signalType: want %q, got %q", SignalCaseUpdate, got)
	}
}

func TestSummarizer_LLMErrorSkipsSignal(t *testing.T) {
	repo := &fakeRepo{}
	fl := &fakeLLM{err: errors.New("llm down")}
	s := NewSummarizer(eventbus.New(), fl, repo)

	s.handle(context.Background(), makeEvent(TopicActivityCreated, "ws-1", "account", "acc-1", "some text"))

	if len(repo.insertCalls) != 0 {
		t.Errorf("expected 0 InsertSignal calls on LLM error, got %d", len(repo.insertCalls))
	}
}

func TestSummarizer_InvalidJSONSkipsSignal(t *testing.T) {
	repo := &fakeRepo{}
	fl := &fakeLLM{response: "not valid json"}
	s := NewSummarizer(eventbus.New(), fl, repo)

	s.handle(context.Background(), makeEvent(TopicNoteCreated, "ws-1", "contact", "con-1", "some text"))

	if len(repo.insertCalls) != 0 {
		t.Errorf("expected 0 InsertSignal calls on invalid JSON, got %d", len(repo.insertCalls))
	}
}

func TestSummarizer_StopsOnContextCancel(t *testing.T) {
	repo := &fakeRepo{}
	fl := &fakeLLM{response: `{"summary":"x","sentiment":"neutral"}`}
	s := NewSummarizer(eventbus.New(), fl, repo)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.Run(ctx)
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
