package api

import (
	"context"
	"errors"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
)

func TestWithWorkspaceIDAndGetWorkspaceID_Success(t *testing.T) {
	t.Parallel()

	ctx := WithWorkspaceID(context.Background(), "ws-123")
	got, err := GetWorkspaceID(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "ws-123" {
		t.Fatalf("expected ws-123, got %q", got)
	}
}

func TestGetWorkspaceID_Missing_ReturnsExpectedError(t *testing.T) {
	t.Parallel()

	_, err := GetWorkspaceID(context.Background())
	if !errors.Is(err, ErrMissingWorkspaceID) {
		t.Fatalf("expected ErrMissingWorkspaceID, got %v", err)
	}
}

func TestGetWorkspaceID_EmptyValue_ReturnsExpectedError(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), ctxkeys.WorkspaceID, "")
	_, err := GetWorkspaceID(ctx)
	if !errors.Is(err, ErrMissingWorkspaceID) {
		t.Fatalf("expected ErrMissingWorkspaceID, got %v", err)
	}
}
