package crm

import (
	"context"
	"fmt"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite/sqlcgen"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

func createTimelineEvent(ctx context.Context, q sqlcgen.Querier, workspaceID, entityType, entityID, actorID, eventType string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	if err := q.CreateTimelineEvent(ctx, sqlcgen.CreateTimelineEventParams{
		ID:          uuid.NewV7().String(),
		WorkspaceID: workspaceID,
		EntityType:  entityType,
		EntityID:    entityID,
		ActorID:     nullString(actorID),
		EventType:   eventType,
		CreatedAt:   now,
	}); err != nil {
		return fmt.Errorf("create timeline event: %w", err)
	}
	return nil
}
