// Package relationship wires event-driven trust recomputation on top of TrustEngine.
package relationship

import (
	"context"
	"fmt"
	"log"

	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
)

// TrustSignalRepository combines the read/write capabilities needed by TrustDriver.
type TrustSignalRepository interface {
	TrustRepository
	ListSignalsByMemory(ctx context.Context, memoryID string) ([]InteractionSignal, error)
}

type trustDriverInput struct {
	memoryID string
	signalID string
}

// TrustDriver subscribes to interaction_signal.created and recomputes trust scores.
type TrustDriver struct {
	bus    eventbus.EventBus
	repo   TrustSignalRepository
	engine *TrustEngine
}

// NewTrustDriver constructs a TrustDriver backed by the shared event bus.
func NewTrustDriver(bus eventbus.EventBus, repo TrustSignalRepository) *TrustDriver {
	return &TrustDriver{
		bus:    bus,
		repo:   repo,
		engine: NewTrustEngine(repo),
	}
}

// Run processes interaction signal events until ctx is cancelled.
func (d *TrustDriver) Run(ctx context.Context) {
	ch := d.bus.Subscribe(TopicInteractionSignalCreated)
	for {
		select {
		case ev := <-ch:
			d.handle(ctx, ev)
		case <-ctx.Done():
			return
		}
	}
}

func (d *TrustDriver) handle(ctx context.Context, ev eventbus.Event) {
	input, err := parseTrustDriverPayload(ev)
	if err != nil {
		log.Printf("relationship.TrustDriver: parse payload topic=%s err=%v", ev.Topic, err)
		return
	}

	signals, err := d.repo.ListSignalsByMemory(ctx, input.memoryID)
	if err != nil {
		log.Printf("relationship.TrustDriver: ListSignalsByMemory topic=%s signal_id=%s err=%v", ev.Topic, input.signalID, err)
		return
	}
	if scoreErr := d.engine.Score(ctx, input.memoryID, signals); scoreErr != nil {
		log.Printf("relationship.TrustDriver: Score topic=%s signal_id=%s err=%v", ev.Topic, input.signalID, scoreErr)
	}
}

func parseTrustDriverPayload(ev eventbus.Event) (trustDriverInput, error) {
	m, ok := ev.Payload.(map[string]any)
	if !ok {
		return trustDriverInput{}, fmt.Errorf("payload is not map[string]any: %T", ev.Payload)
	}

	str := func(key string) string {
		v, _ := m[key].(string)
		return v
	}

	input := trustDriverInput{
		memoryID: str("memory_id"),
		signalID: str("signal_id"),
	}
	if input.memoryID == "" || input.signalID == "" {
		return trustDriverInput{}, fmt.Errorf("payload missing memory_id or signal_id")
	}
	return input, nil
}
