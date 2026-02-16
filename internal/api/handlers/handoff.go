// Package handlers — Handoff Manager HTTP handler.
// Task 3.8: Human handoff endpoints (FR-232).
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
)

// HandoffHandler handles HTTP requests for agent-to-human escalation.
type HandoffHandler struct {
	service *agent.HandoffService
}

// NewHandoffHandler creates a new HandoffHandler.
func NewHandoffHandler(service *agent.HandoffService) *HandoffHandler {
	return &HandoffHandler{service: service}
}

// GetHandoffPackage handles GET /api/v1/agents/runs/{id}/handoff?case_id=<id>
// Returns the handoff context for an escalated agent run (read-only).
func (h *HandoffHandler) GetHandoffPackage(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := r.Context().Value(ctxkeys.WorkspaceID).(string)
	if !ok || workspaceID == "" {
		writeError(w, http.StatusUnauthorized, errMissingWorkspaceContext)
		return
	}

	runID := chi.URLParam(r, paramID)
	caseID := r.URL.Query().Get("case_id")

	pkg, err := h.service.GetHandoffPackage(r.Context(), workspaceID, runID, caseID)
	if err != nil {
		h.writeHandoffError(w, err)
		return
	}

	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": pkg})
}

// InitiateHandoff handles POST /api/v1/agents/runs/{id}/handoff
// Builds the handoff package, updates the case status, and emits an event.
func (h *HandoffHandler) InitiateHandoff(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := r.Context().Value(ctxkeys.WorkspaceID).(string)
	if !ok || workspaceID == "" {
		writeError(w, http.StatusUnauthorized, errMissingWorkspaceContext)
		return
	}

	runID := chi.URLParam(r, paramID)

	var req struct {
		CaseID string `json:"case_id"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errInvalidBody)
		return
	}

	pkg, err := h.service.InitiateHandoff(r.Context(), workspaceID, runID, req.CaseID, req.Reason)
	if err != nil {
		h.writeHandoffError(w, err)
		return
	}

	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": pkg})
}

// writeHandoffError maps domain errors to HTTP status codes.
func (h *HandoffHandler) writeHandoffError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, agent.ErrAgentRunNotFound):
		writeError(w, http.StatusNotFound, "agent run not found")
	case errors.Is(err, agent.ErrHandoffCaseNotFound):
		writeError(w, http.StatusNotFound, "case not found")
	default:
		writeError(w, http.StatusInternalServerError, "handoff failed")
	}
}
