package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/copilot"
)

type CopilotChatService interface {
	Chat(ctx context.Context, in copilot.ChatInput) (<-chan copilot.StreamChunk, error)
}

type CopilotChatHandler struct {
	chatService CopilotChatService
}

func NewCopilotChatHandler(chatService CopilotChatService) *CopilotChatHandler {
	return &CopilotChatHandler{chatService: chatService}
}

type copilotChatRequest struct {
	Query      string  `json:"query"`
	EntityType *string `json:"entityType,omitempty"`
	EntityID   *string `json:"entityId,omitempty"`
}

func (h *CopilotChatHandler) Chat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID, err := getWorkspaceID(ctx)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing workspace context")
		return
	}

	userID, _ := ctx.Value(ctxkeys.UserID).(string)
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing user context")
		return
	}

	var req copilotChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}

	stream, err := h.chatService.Chat(ctx, copilot.ChatInput{
		WorkspaceID: wsID,
		UserID:      userID,
		Query:       req.Query,
		EntityType:  req.EntityType,
		EntityID:    req.EntityID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "chat failed")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	bw := bufio.NewWriter(w)
	for chunk := range stream {
		b, _ := json.Marshal(chunk)
		if _, err := fmt.Fprintf(bw, "data: %s\n\n", string(b)); err != nil {
			return
		}
		_ = bw.Flush()
		flusher.Flush()
	}
}
