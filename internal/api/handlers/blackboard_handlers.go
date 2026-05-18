package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/domain/blackboard"
)

const routeParamCognitiveWorkspaceID = "cwID"

type blackboardPipelineRunner interface {
	RunPipeline(ctx context.Context, cognitiveWorkspaceID string) (*blackboard.ExecutionOutcome, error)
}

type BlackboardHandler struct {
	orchestrator blackboardPipelineRunner
	authz        ActionAuthorizer
}

func NewBlackboardHandlerWithAuthorizer(orchestrator blackboardPipelineRunner, authz ActionAuthorizer) *BlackboardHandler {
	return &BlackboardHandler{orchestrator: orchestrator, authz: authz}
}

func (h *BlackboardHandler) RunPipeline(w http.ResponseWriter, r *http.Request) {
	if !checkActionAuthorization(w, r, h.authz, resourceAPI, "admin.blackboard.plan") {
		return
	}

	cognitiveWorkspaceID := chi.URLParam(r, routeParamCognitiveWorkspaceID)
	if cognitiveWorkspaceID == "" {
		writeError(w, http.StatusBadRequest, "cognitive workspace id is required")
		return
	}

	outcome, err := h.orchestrator.RunPipeline(r.Context(), cognitiveWorkspaceID)
	if err != nil {
		writeBlackboardPipelineError(w, err)
		return
	}

	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(outcome)
}

func writeBlackboardPipelineError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, blackboard.ErrPipelineAlreadyRunning):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, blackboard.ErrCognitiveWorkspaceNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "failed to run blackboard pipeline")
	}
}
