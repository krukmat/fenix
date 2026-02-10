package crm

import (
	"context"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite/sqlcgen"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

func createTimelineEvent(ctx context.Context, q sqlcgen.Querier, workspaceID, entityType, entityID, actorID, eventType string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	return q.CreateTimelineEvent(ctx, sqlcgen.CreateTimelineEventParams{
		ID:          uuid.NewV7().String(),
		WorkspaceID: workspaceID,
		EntityType:  entityType,
		EntityID:    entityID,
		ActorID:     nullString(actorID),
		EventType:   eventType,
		CreatedAt:   now,
	})
}
