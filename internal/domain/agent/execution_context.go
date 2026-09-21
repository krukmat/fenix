package agent

import (
	"context"
	"strings"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
)

func withAgentRunExecutionContext(ctx context.Context, run *Run) context.Context {
	if run == nil {
		return ctx
	}
	if strings.TrimSpace(run.WorkspaceID) != "" {
		ctx = ctxkeys.WithValue(ctx, ctxkeys.WorkspaceID, run.WorkspaceID)
	}
	if strings.TrimSpace(run.ID) != "" {
		ctx = ctxkeys.WithValue(ctx, ctxkeys.RunID, run.ID)
	}
	if run.TraceID != nil && strings.TrimSpace(*run.TraceID) != "" {
		ctx = ctxkeys.WithValue(ctx, ctxkeys.TraceID, strings.TrimSpace(*run.TraceID))
	}
	if _, exists := ctx.Value(ctxkeys.UserID).(string); !exists &&
		run.TriggeredByUserID != nil && strings.TrimSpace(*run.TriggeredByUserID) != "" {
		ctx = ctxkeys.WithValue(ctx, ctxkeys.UserID, strings.TrimSpace(*run.TriggeredByUserID))
	}
	return ctx
}
