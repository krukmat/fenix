package ctxkeys

import (
	"context"
	"testing"
)

func TestWithValue_SetsAndGetsTypedKey(t *testing.T) {
	t.Parallel()

	ctx := WithValue(context.Background(), WorkspaceID, "ws-999")
	got, ok := ctx.Value(WorkspaceID).(string)
	if !ok {
		t.Fatalf("expected string value")
	}
	if got != "ws-999" {
		t.Fatalf("expected ws-999, got %q", got)
	}
}
