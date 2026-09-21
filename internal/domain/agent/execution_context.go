package agent

import (
	"context"
	"strings"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
)

// WithRunExecutionContext propagates the stable identity of an agent run into downstream execution context.
func WithRunExecutionContext(ctx context.Context, run *Run) context.Context {
	if run == nil {
		return ctx
	}
	ctx = contextWithNonEmpty(ctx, ctxkeys.WorkspaceID, run.WorkspaceID)
	ctx = contextWithNonEmpty(ctx, ctxkeys.RunID, run.ID)
	ctx = contextWithOptional(ctx, ctxkeys.TraceID, run.TraceID)
	return contextWithTriggeredBy(ctx, run.TriggeredByUserID)
}

func contextWithNonEmpty(ctx context.Context, key ctxkeys.Key, value string) context.Context {
	value = strings.TrimSpace(value)
	if value == "" {
		return ctx
	}
	return ctxkeys.WithValue(ctx, key, value)
}

func contextWithOptional(ctx context.Context, key ctxkeys.Key, value *string) context.Context {
	if value == nil {
		return ctx
	}
	return contextWithNonEmpty(ctx, key, *value)
}

func contextWithTriggeredBy(ctx context.Context, triggeredBy *string) context.Context {
	if _, exists := ctx.Value(ctxkeys.UserID).(string); exists {
		return ctx
	}
	return contextWithOptional(ctx, ctxkeys.UserID, triggeredBy)
}
