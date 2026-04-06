// Traces: FR-200, FR-201
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/copilot"
)

type copilotChatServiceStub struct {
	chunks []copilot.StreamChunk
	err    error
}

func (s *copilotChatServiceStub) Chat(_ context.Context, _ copilot.ChatInput) (<-chan copilot.StreamChunk, error) {
	if s.err != nil {
		return nil, s.err
	}
	out := make(chan copilot.StreamChunk, len(s.chunks))
	for _, c := range s.chunks {
		out <- c
	}
	close(out)
	return out, nil
}

func TestCopilotChatHandler_SSE_OK(t *testing.T) {
	h := NewCopilotChatHandler(&copilotChatServiceStub{chunks: []copilot.StreamChunk{
		{Type: "evidence", Meta: map[string]any{"schema_version": "v1", "source_count": 1}},
		{Type: "token", Delta: "hola "},
		{Type: "done", Done: true, Meta: map[string]any{"answer_type": "grounded_answer"}},
	}})

	body, _ := json.Marshal(map[string]any{"query": "estado del caso"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/copilot/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), ctxkeys.WorkspaceID, "ws_1")
	ctx = context.WithValue(ctx, ctxkeys.UserID, "u_1")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.Chat(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("expected text/event-stream, got %q", ct)
	}
	if !strings.Contains(rr.Body.String(), "data: {") {
		t.Fatalf("expected SSE data frames, got %q", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "\"answer_type\":\"grounded_answer\"") {
		t.Fatalf("expected done metadata to be preserved, got %q", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "\"schema_version\":\"v1\"") {
		t.Fatalf("expected evidence metadata to be preserved, got %q", rr.Body.String())
	}
}

// TestChatRequestError_Error covers the Error() method on chatRequestError.
func TestChatRequestError_Error(t *testing.T) {
	t.Parallel()

	e := chatRequestError{status: 400, message: "bad request"}
	if got := e.Error(); got != "bad request" {
		t.Fatalf("expected %q, got %q", "bad request", got)
	}
}

// TestCopilotChatHandler_Chat_ServiceError returns 500 when chat service fails.
func TestCopilotChatHandler_Chat_ServiceError(t *testing.T) {
	t.Parallel()

	h := NewCopilotChatHandler(&copilotChatServiceStub{err: errors.New("provider down")})

	body, _ := json.Marshal(map[string]any{"query": "help"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/copilot/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), ctxkeys.WorkspaceID, "ws_1")
	ctx = context.WithValue(ctx, ctxkeys.UserID, "u_1")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.Chat(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rr.Code, rr.Body.String())
	}
}

// TestCopilotChatHandler_Chat_MissingUser returns 401 when user context is absent.
func TestCopilotChatHandler_Chat_MissingUser(t *testing.T) {
	t.Parallel()

	h := NewCopilotChatHandler(&copilotChatServiceStub{})

	body, _ := json.Marshal(map[string]any{"query": "help"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/copilot/chat", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), ctxkeys.WorkspaceID, "ws_1"))

	rr := httptest.NewRecorder()
	h.Chat(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestCopilotChatHandler_Validation(t *testing.T) {
	h := NewCopilotChatHandler(&copilotChatServiceStub{})

	t.Run("missing workspace", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/copilot/chat", bytes.NewBufferString(`{"query":"x"}`))
		req = req.WithContext(context.WithValue(req.Context(), ctxkeys.UserID, "u_1"))
		rr := httptest.NewRecorder()
		h.Chat(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("missing query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/copilot/chat", bytes.NewBufferString(`{}`))
		ctx := context.WithValue(req.Context(), ctxkeys.WorkspaceID, "ws_1")
		ctx = context.WithValue(ctx, ctxkeys.UserID, "u_1")
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()
		h.Chat(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})
}
