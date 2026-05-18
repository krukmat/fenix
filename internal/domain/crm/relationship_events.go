package crm

import (
	"fmt"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/relationship"
	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
)

func publishActivityCreated(bus eventbus.EventBus, activity *Activity) {
	if bus == nil || activity == nil {
		return
	}
	bus.Publish(relationship.TopicActivityCreated, map[string]any{
		"workspace_id":       activity.WorkspaceID,
		"entity_type":        activity.EntityType,
		"entity_id":          activity.EntityID,
		"raw_text":           joinNonEmpty(activity.Subject, stringValue(activity.Body)),
		"source_entity_type": "activity",
		"source_entity_id":   activity.ID,
		"occurred_at":        activity.CreatedAt.UTC().Format(time.RFC3339),
		"activity_type":      activity.ActivityType,
	})
}

func publishNoteCreated(bus eventbus.EventBus, note *Note) {
	if bus == nil || note == nil {
		return
	}
	bus.Publish(relationship.TopicNoteCreated, map[string]any{
		"workspace_id":       note.WorkspaceID,
		"entity_type":        note.EntityType,
		"entity_id":          note.EntityID,
		"raw_text":           note.Content,
		"source_entity_type": "note",
		"source_entity_id":   note.ID,
		"occurred_at":        note.CreatedAt.UTC().Format(time.RFC3339),
	})
}

func publishDealUpdated(bus eventbus.EventBus, deal *Deal) {
	if bus == nil || deal == nil {
		return
	}
	bus.Publish(relationship.TopicDealUpdated, map[string]any{
		"workspace_id":       deal.WorkspaceID,
		"entity_type":        "deal",
		"entity_id":          deal.ID,
		"raw_text":           buildDealRelationshipText(deal),
		"source_entity_type": "deal",
		"source_entity_id":   deal.ID,
		"occurred_at":        deal.UpdatedAt.UTC().Format(time.RFC3339),
	})
}

func publishCaseUpdated(bus eventbus.EventBus, ticket *CaseTicket) {
	if bus == nil || ticket == nil {
		return
	}
	bus.Publish(relationship.TopicCaseUpdated, map[string]any{
		"workspace_id":       ticket.WorkspaceID,
		"entity_type":        "case",
		"entity_id":          ticket.ID,
		"raw_text":           buildCaseRelationshipText(ticket),
		"source_entity_type": "case",
		"source_entity_id":   ticket.ID,
		"occurred_at":        ticket.UpdatedAt.UTC().Format(time.RFC3339),
	})
}

func joinNonEmpty(values ...string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return strings.Join(parts, "\n\n")
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func buildDealRelationshipText(deal *Deal) string {
	parts := []string{
		fmt.Sprintf("Deal: %s", deal.Title),
		fmt.Sprintf("Status: %s", deal.Status),
		fmt.Sprintf("Stage: %s", deal.StageID),
	}
	if deal.Amount != nil {
		parts = append(parts, fmt.Sprintf("Amount: %.2f", *deal.Amount))
	}
	if deal.Currency != nil && *deal.Currency != "" {
		parts = append(parts, "Currency: "+*deal.Currency)
	}
	return strings.Join(parts, "\n")
}

func buildCaseRelationshipText(ticket *CaseTicket) string {
	return joinNonEmpty(
		"Case: "+ticket.Subject,
		"Status: "+ticket.Status,
		"Priority: "+ticket.Priority,
		stringValue(ticket.Description),
	)
}
