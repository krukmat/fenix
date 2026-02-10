package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
)

type PipelineHandler struct{ service *crm.PipelineService }

func NewPipelineHandler(service *crm.PipelineService) *PipelineHandler {
	return &PipelineHandler{service: service}
}

type CreatePipelineRequest struct {
	Name       string `json:"name"`
	EntityType string `json:"entityType"`
	Settings   string `json:"settings,omitempty"`
}

type UpdatePipelineRequest = CreatePipelineRequest

type CreatePipelineStageRequest struct {
	Name           string   `json:"name"`
	Position       int64    `json:"position"`
	Probability    *float64 `json:"probability,omitempty"`
	SlaHours       *int64   `json:"slaHours,omitempty"`
	RequiredFields string   `json:"requiredFields,omitempty"`
}

type UpdatePipelineStageRequest = CreatePipelineStageRequest

func (h *PipelineHandler) CreatePipeline(w http.ResponseWriter, r *http.Request) {
	wsID, err := getWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}
	var req CreatePipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.EntityType == "" {
		writeError(w, http.StatusBadRequest, "name and entityType are required")
		return
	}
	out, err := h.service.Create(r.Context(), crm.CreatePipelineInput{WorkspaceID: wsID, Name: req.Name, EntityType: req.EntityType, Settings: req.Settings})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create pipeline: %v", err))
		return
	}
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(out); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
}

func (h *PipelineHandler) ListPipelines(w http.ResponseWriter, r *http.Request) {
	wsID, err := getWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}
	page := parsePaginationParams(r)
	items, total, err := h.service.List(r.Context(), wsID, crm.ListPipelinesInput{Limit: page.Limit, Offset: page.Offset})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list pipelines: %v", err))
		return
	}
	if err := json.NewEncoder(w).Encode(map[string]any{"data": items, "meta": Meta{Total: total, Limit: page.Limit, Offset: page.Offset}}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
}

func (h *PipelineHandler) GetPipeline(w http.ResponseWriter, r *http.Request) {
	wsID, err := getWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}
	id := chi.URLParam(r, "id")
	out, err := h.service.Get(r.Context(), wsID, id)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "pipeline not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get pipeline: %v", err))
		return
	}
	if err := json.NewEncoder(w).Encode(out); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
}

func (h *PipelineHandler) UpdatePipeline(w http.ResponseWriter, r *http.Request) {
	wsID, err := getWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}
	id := chi.URLParam(r, "id")
	existing, err := h.service.Get(r.Context(), wsID, id)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "pipeline not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get pipeline: %v", err))
		return
	}
	var req UpdatePipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	out, err := h.service.Update(r.Context(), wsID, id, crm.UpdatePipelineInput{
		Name:       coalesce(req.Name, existing.Name),
		EntityType: coalesce(req.EntityType, existing.EntityType),
		Settings:   req.Settings,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update pipeline: %v", err))
		return
	}
	json.NewEncoder(w).Encode(out) //nolint:errcheck
}

func (h *PipelineHandler) DeletePipeline(w http.ResponseWriter, r *http.Request) {
	wsID, err := getWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.service.Delete(r.Context(), wsID, id); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete pipeline: %v", err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *PipelineHandler) CreateStage(w http.ResponseWriter, r *http.Request) {
	pipelineID := chi.URLParam(r, "id")
	var req CreatePipelineStageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Position == 0 {
		req.Position = 1
	}
	out, err := h.service.CreateStage(r.Context(), crm.CreatePipelineStageInput{
		PipelineID:     pipelineID,
		Name:           req.Name,
		Position:       req.Position,
		Probability:    req.Probability,
		SlaHours:       req.SlaHours,
		RequiredFields: req.RequiredFields,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create stage: %v", err))
		return
	}
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(out); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
}

func (h *PipelineHandler) ListStages(w http.ResponseWriter, r *http.Request) {
	pipelineID := chi.URLParam(r, "id")
	items, err := h.service.ListStages(r.Context(), pipelineID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list stages: %v", err))
		return
	}
	if err := json.NewEncoder(w).Encode(map[string]any{"data": items, "meta": Meta{Total: len(items), Limit: len(items), Offset: 0}}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
}

func (h *PipelineHandler) UpdateStage(w http.ResponseWriter, r *http.Request) {
	stageID := chi.URLParam(r, "stage_id")
	existing, err := h.service.GetStage(r.Context(), stageID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "stage not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get stage: %v", err))
		return
	}
	var req UpdatePipelineStageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		req.Name = existing.Name
	}
	if req.Position == 0 {
		req.Position = existing.Position
	}
	out, err := h.service.UpdateStage(r.Context(), stageID, crm.UpdatePipelineStageInput{
		Name:           req.Name,
		Position:       req.Position,
		Probability:    req.Probability,
		SlaHours:       req.SlaHours,
		RequiredFields: req.RequiredFields,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update stage: %v", err))
		return
	}
	if err := json.NewEncoder(w).Encode(out); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
}

func (h *PipelineHandler) DeleteStage(w http.ResponseWriter, r *http.Request) {
	stageID := chi.URLParam(r, "stage_id")
	if stageID == "" {
		stageID = chi.URLParam(r, "id")
	}
	if err := h.service.DeleteStage(r.Context(), stageID); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete stage: %v", err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
