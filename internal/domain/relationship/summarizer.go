// Package relationship — Task B.2.3: Summarizer service.
// Subscribes to three event bus topics, calls LLM, delegates persistence to SignalRepository.
package relationship

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
	"github.com/matiasleandrokruk/fenix/internal/infra/llm"
)

// summarizerSystemPrompt instructs the LLM to return only structured JSON.
const summarizerSystemPrompt = `You are a CRM interaction analyzer.
Given event text, return ONLY valid JSON:
{"summary": "<one sentence>", "sentiment": "positive|neutral|negative"}.
No explanation. No markdown.`

// llmResponse is the expected JSON shape from the LLM.
type llmResponse struct {
	Summary   string `json:"summary"`
	Sentiment string `json:"sentiment"`
}

// eventInput holds the parsed fields extracted from an event payload.
type eventInput struct {
	workspaceID      string
	entityType       EntityType
	entityID         string
	rawText          string
	sourceEntityType string
	sourceEntityID   string
	occurredAt       time.Time
}

// Summarizer subscribes to CRM event bus topics and writes relationship signals.
// Task B.2.3: no SQL, no HTTP — pure domain service.
type Summarizer struct {
	bus  eventbus.EventBus
	llm  llm.LLMProvider
	repo SignalRepository
}

// NewSummarizer constructs a Summarizer with its required dependencies.
func NewSummarizer(bus eventbus.EventBus, provider llm.LLMProvider, repo SignalRepository) *Summarizer {
	return &Summarizer{bus: bus, llm: provider, repo: repo}
}

// Run subscribes to all supported topics and processes events until ctx is cancelled.
// Task B.2.3: select loop — handle is synchronous to preserve per-channel ordering.
func (s *Summarizer) Run(ctx context.Context) {
	chActivity := s.bus.Subscribe(TopicActivityCreated)
	chNote := s.bus.Subscribe(TopicNoteCreated)
	chCase := s.bus.Subscribe(TopicCaseUpdated)
	chDeal := s.bus.Subscribe(TopicDealUpdated)

	for {
		select {
		case ev := <-chActivity:
			s.handle(ctx, ev)
		case ev := <-chNote:
			s.handle(ctx, ev)
		case ev := <-chCase:
			s.handle(ctx, ev)
		case ev := <-chDeal:
			s.handle(ctx, ev)
		case <-ctx.Done():
			return
		}
	}
}

// handle extracts the event payload, calls the LLM, and persists via the repository.
func (s *Summarizer) handle(ctx context.Context, ev eventbus.Event) {
	input, err := parseEventPayload(ev)
	if err != nil {
		log.Printf("relationship.Summarizer: parse payload topic=%s err=%v", ev.Topic, err)
		return
	}

	sigType, err := signalTypeFor(ev.Topic, ev.Payload)
	if err != nil {
		log.Printf("relationship.Summarizer: reject topic=%s err=%v", ev.Topic, err)
		return
	}

	summary, sentiment, err := s.callLLM(ctx, input.rawText)
	if err != nil {
		log.Printf("relationship.Summarizer: LLM call topic=%s err=%v", ev.Topic, err)
		return
	}

	mem, err := s.repo.UpsertMemory(ctx, input.workspaceID, input.entityType, input.entityID, summary)
	if err != nil {
		log.Printf("relationship.Summarizer: UpsertMemory topic=%s err=%v", ev.Topic, err)
		return
	}

	signalID, insertErr := s.repo.InsertSignal(ctx, mem.ID, sigType, SentimentType(sentiment),
		summary, input.sourceEntityType, input.sourceEntityID, input.occurredAt)
	if insertErr != nil {
		log.Printf("relationship.Summarizer: InsertSignal topic=%s err=%v", ev.Topic, insertErr)
		// non-fatal: memory already upserted
		return
	}

	s.bus.Publish(TopicInteractionSignalCreated, map[string]any{
		"workspace_id": input.workspaceID,
		"memory_id":    mem.ID,
		"signal_id":    signalID,
		"summary":      summary,
	})
}

// callLLM builds the prompt, calls the provider, and validates the response JSON.
func (s *Summarizer) callLLM(ctx context.Context, rawText string) (summary, sentiment string, err error) {
	resp, err := s.llm.ChatCompletion(ctx, llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "system", Content: summarizerSystemPrompt},
			{Role: "user", Content: rawText},
		},
		Temperature: 0.2,
		MaxTokens:   256,
	})
	if err != nil {
		return "", "", fmt.Errorf("callLLM: chat completion: %w", err)
	}

	var result llmResponse
	if unmarshalErr := json.Unmarshal([]byte(resp.Content), &result); unmarshalErr != nil {
		return "", "", fmt.Errorf("callLLM: unmarshal JSON: %w", unmarshalErr)
	}

	if result.Sentiment != "positive" && result.Sentiment != "neutral" && result.Sentiment != "negative" {
		return "", "", fmt.Errorf("callLLM: invalid sentiment %q", result.Sentiment)
	}

	return result.Summary, result.Sentiment, nil
}

// parseEventPayload extracts structured fields from an event's map[string]any payload.
// Task B.2.3: all fields are best-effort strings; occurredAt defaults to now if missing/invalid.
func parseEventPayload(ev eventbus.Event) (eventInput, error) {
	m, ok := ev.Payload.(map[string]any)
	if !ok {
		return eventInput{}, fmt.Errorf("payload is not map[string]any: %T", ev.Payload)
	}

	str := func(key string) string {
		v, _ := m[key].(string)
		return v
	}

	entityType := EntityType(str("entity_type"))
	rawText := str("raw_text")
	if rawText == "" {
		return eventInput{}, fmt.Errorf("payload missing raw_text")
	}

	occurredAt := time.Now().UTC()
	if ts, tsOk := m["occurred_at"].(string); tsOk && ts != "" {
		if parsed, parseErr := time.Parse(time.RFC3339, ts); parseErr == nil {
			occurredAt = parsed.UTC()
		}
	}

	return eventInput{
		workspaceID:      str("workspace_id"),
		entityType:       entityType,
		entityID:         str("entity_id"),
		rawText:          rawText,
		sourceEntityType: str("source_entity_type"),
		sourceEntityID:   str("source_entity_id"),
		occurredAt:       occurredAt,
	}, nil
}

// signalTypeFor maps an event bus topic and payload to its SignalType constant.
func signalTypeFor(topic string, payload any) (SignalType, error) {
	if topic == TopicActivityCreated {
		return signalTypeForActivity(payload)
	}

	signalType, ok := signalTypeByTopic[topic]
	if !ok {
		return "", fmt.Errorf("unknown topic %q", topic)
	}
	return signalType, nil
}

var signalTypeByTopic = map[string]SignalType{
	TopicNoteCreated: SignalNote,
	TopicCaseUpdated: SignalCaseUpdate,
	TopicDealUpdated: SignalDealUpdate,
}

func signalTypeForActivity(payload any) (SignalType, error) {
	switch strings.ToLower(stringFromPayload(payload, "activity_type")) {
	case "", "email":
		return SignalEmail, nil
	case "call":
		return SignalCall, nil
	case "meeting", "event":
		return SignalMeeting, nil
	default:
		return "", fmt.Errorf("unknown activity_type %q", stringFromPayload(payload, "activity_type"))
	}
}

func stringFromPayload(payload any, key string) string {
	m, ok := payload.(map[string]any)
	if !ok {
		return ""
	}
	v, _ := m[key].(string)
	return v
}
