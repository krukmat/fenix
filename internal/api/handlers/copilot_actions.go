package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/copilot"
)

type CopilotActionsService interface {
	SuggestActions(ctx context.Context, in copilot.SuggestActionsInput) ([]copilot.SuggestedAction, error)
	Summarize(ctx context.Context, in copilot.SummarizeInput) (string, error)
}

type CopilotActionsHandler struct {
	service CopilotActionsService
}

func NewCopilotActionsHandler(service CopilotActionsService) *CopilotActionsHandler {
	return &CopilotActionsHandler{service: service}
}

type copilotEntityRequest struct {
	EntityType string `json:"entityType"`
	EntityID   string `json:"entityId"`
}

type copilotSuggestActionsResponse struct {
	Data struct {
		Actions []copilot.SuggestedAction `json:"actions"`
	} `json:"data"`
}

type copilotSummarizeResponse struct {
	Data struct {
		Summary string `json:"summary"`
	} `json:"data"`
}

type actionRequestError struct {
	status  int
	message string
}

func (e actionRequestError) Error() string { return e.message }

func (h *CopilotActionsHandler) SuggestActions(w http.ResponseWriter, r *http.Request) {
	common, err := buildCopilotEntityInput(r)
	if err != nil {
		writeCopilotActionsError(w, err)
		return
	}

	actions, err := h.service.SuggestActions(r.Context(), copilot.SuggestActionsInput{
		WorkspaceID: common.WorkspaceID,
		UserID:      common.UserID,
		EntityType:  common.EntityType,
		EntityID:    common.EntityID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "suggest-actions failed")
		return
	}

	resp := copilotSuggestActionsResponse{}
	resp.Data.Actions = actions

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *CopilotActionsHandler) Summarize(w http.ResponseWriter, r *http.Request) {
	common, err := buildCopilotEntityInput(r)
	if err != nil {
		writeCopilotActionsError(w, err)
		return
	}

	summary, err := h.service.Summarize(r.Context(), copilot.SummarizeInput{
		WorkspaceID: common.WorkspaceID,
		UserID:      common.UserID,
		EntityType:  common.EntityType,
		EntityID:    common.EntityID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "summarize failed")
		return
	}

	resp := copilotSummarizeResponse{}
	resp.Data.Summary = summary

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

type copilotEntityInput struct {
	WorkspaceID string
	UserID      string
	EntityType  string
	EntityID    string
}

func buildCopilotEntityInput(r *http.Request) (copilotEntityInput, error) {
	ctx := r.Context()
	wsID, err := getWorkspaceID(ctx)
	if err != nil {
		return copilotEntityInput{}, actionRequestError{status: http.StatusUnauthorized, message: "missing workspace context"}
	}

	userID, _ := ctx.Value(ctxkeys.UserID).(string)
	if userID == "" {
		return copilotEntityInput{}, actionRequestError{status: http.StatusUnauthorized, message: "missing user context"}
	}

	var req copilotEntityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return copilotEntityInput{}, actionRequestError{status: http.StatusBadRequest, message: "invalid request body"}
	}
	if req.EntityType == "" || req.EntityID == "" {
		return copilotEntityInput{}, actionRequestError{status: http.StatusBadRequest, message: "entityType and entityId are required"}
	}

	return copilotEntityInput{
		WorkspaceID: wsID,
		UserID:      userID,
		EntityType:  req.EntityType,
		EntityID:    req.EntityID,
	}, nil
}

func writeCopilotActionsError(w http.ResponseWriter, err error) {
	var reqErr actionRequestError
	if ok := errorAsAction(err, &reqErr); ok {
		writeError(w, reqErr.status, reqErr.message)
		return
	}
	writeError(w, http.StatusInternalServerError, "request failed")
}

func errorAsAction(err error, target *actionRequestError) bool {
	reqErr, ok := err.(actionRequestError)
	if !ok {
		return false
	}
	*target = reqErr
	return true
}
