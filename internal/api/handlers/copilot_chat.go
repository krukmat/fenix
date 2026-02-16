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

type chatRequestError struct {
	status  int
	message string
}

func (e chatRequestError) Error() string { return e.message }

func (h *CopilotChatHandler) Chat(w http.ResponseWriter, r *http.Request) {
	input, err := buildCopilotChatInput(r)
	if err != nil {
		writeCopilotChatError(w, err)
		return
	}

	stream, err := h.chatService.Chat(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "chat failed")
		return
	}

	bw, flusher, err := prepareCopilotChatStream(w)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}
	streamCopilotChunks(bw, flusher, stream)
}

func buildCopilotChatInput(r *http.Request) (copilot.ChatInput, error) {
	ctx := r.Context()
	wsID, err := getWorkspaceID(ctx)
	if err != nil {
		return copilot.ChatInput{}, chatRequestError{status: http.StatusUnauthorized, message: "missing workspace context"}
	}

	userID, _ := ctx.Value(ctxkeys.UserID).(string)
	if userID == "" {
		return copilot.ChatInput{}, chatRequestError{status: http.StatusUnauthorized, message: "missing user context"}
	}

	var req copilotChatRequest
	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		return copilot.ChatInput{}, chatRequestError{status: http.StatusBadRequest, message: "invalid request body"}
	}
	if req.Query == "" {
		return copilot.ChatInput{}, chatRequestError{status: http.StatusBadRequest, message: "query is required"}
	}

	return copilot.ChatInput{
		WorkspaceID: wsID,
		UserID:      userID,
		Query:       req.Query,
		EntityType:  req.EntityType,
		EntityID:    req.EntityID,
	}, nil
}

func prepareCopilotChatStream(w http.ResponseWriter) (*bufio.Writer, http.Flusher, error) {
	w.Header().Set(headerContentType, "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, nil, fmt.Errorf("response writer does not implement http.Flusher")
	}

	return bufio.NewWriter(w), flusher, nil
}

func streamCopilotChunks(bw *bufio.Writer, flusher http.Flusher, stream <-chan copilot.StreamChunk) {
	for chunk := range stream {
		b, _ := json.Marshal(chunk)
		if _, err := fmt.Fprintf(bw, "data: %s\n\n", string(b)); err != nil {
			return
		}
		_ = bw.Flush()
		flusher.Flush()
	}
}

func writeCopilotChatError(w http.ResponseWriter, err error) {
	var reqErr chatRequestError
	if ok := errorAs(err, &reqErr); ok {
		writeError(w, reqErr.status, reqErr.message)
		return
	}
	writeError(w, http.StatusInternalServerError, "chat failed")
}

func errorAs(err error, target *chatRequestError) bool {
	reqErr, ok := err.(chatRequestError)
	if !ok {
		return false
	}
	*target = reqErr
	return true
}
