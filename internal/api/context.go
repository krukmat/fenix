// Task 1.3.8: Shared context helpers for API middleware
package api

import (
	"context"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
)

// WithWorkspaceID adds workspace_id to the request context.
// Uses ctxkeys.WorkspaceID — shared key used by middleware and handlers alike.
func WithWorkspaceID(ctx context.Context, wsID string) context.Context {
	return context.WithValue(ctx, ctxkeys.WorkspaceID, wsID)
}

// GetWorkspaceID retrieves workspace_id from context.
func GetWorkspaceID(ctx context.Context) (string, error) {
	wsID, ok := ctx.Value(ctxkeys.WorkspaceID).(string)
	if !ok || wsID == "" {
		return "", ErrMissingWorkspaceID
	}
	return wsID, nil
}
