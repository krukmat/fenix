package agent

import (
	"context"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
)

func TestWithAgentRunExecutionContext_PropagatesRunAndTrace(t *testing.T) {
	traceID := "trace-1"
	userID := "user-1"
	run := &Run{
		ID:                "run-1",
		WorkspaceID:       "ws-1",
		TriggeredByUserID: &userID,
		TraceID:           &traceID,
	}

	ctx := WithRunExecutionContext(context.Background(), run)

	if got, _ := ctx.Value(ctxkeys.WorkspaceID).(string); got != "ws-1" {
		t.Fatalf("workspace id = %q, want ws-1", got)
	}
	if got, _ := ctx.Value(ctxkeys.RunID).(string); got != "run-1" {
		t.Fatalf("run id = %q, want run-1", got)
	}
	if got, _ := ctx.Value(ctxkeys.TraceID).(string); got != "trace-1" {
		t.Fatalf("trace id = %q, want trace-1", got)
	}
	if got, _ := ctx.Value(ctxkeys.UserID).(string); got != "user-1" {
		t.Fatalf("user id = %q, want user-1", got)
	}
}

func TestWithAgentRunExecutionContext_PreservesExistingUser(t *testing.T) {
	traceID := "trace-1"
	triggeredBy := "trigger-user"
	run := &Run{
		ID:                "run-1",
		WorkspaceID:       "ws-1",
		TriggeredByUserID: &triggeredBy,
		TraceID:           &traceID,
	}

	ctx := context.WithValue(context.Background(), ctxkeys.UserID, "request-user")
	ctx = WithRunExecutionContext(ctx, run)

	if got, _ := ctx.Value(ctxkeys.UserID).(string); got != "request-user" {
		t.Fatalf("user id = %q, want request-user", got)
	}
}
