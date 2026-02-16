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
	SLAHours       *int64   `json:"slaHours,omitempty"`
	RequiredFields string   `json:"requiredFields,omitempty"`
}

type UpdatePipelineStageRequest = CreatePipelineStageRequest

func (h *PipelineHandler) CreatePipeline(w http.ResponseWriter, r *http.Request) {
	wsID, wsErr := getWorkspaceID(r.Context())
	if wsErr != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}
	var req CreatePipelineRequest
	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.EntityType == "" {
		writeError(w, http.StatusBadRequest, "name and entityType are required")
		return
	}
	out, svcErr := h.service.Create(r.Context(), crm.CreatePipelineInput{WorkspaceID: wsID, Name: req.Name, EntityType: req.EntityType, Settings: req.Settings})
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create pipeline: %v", svcErr))
		return
	}
	w.WriteHeader(http.StatusCreated)
	if encodeErr := json.NewEncoder(w).Encode(out); encodeErr != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
}

func (h *PipelineHandler) ListPipelines(w http.ResponseWriter, r *http.Request) {
	wsID, wsErr := getWorkspaceID(r.Context())
	if wsErr != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}
	page := parsePaginationParams(r)
	items, total, svcErr := h.service.List(r.Context(), wsID, crm.ListPipelinesInput{Limit: page.Limit, Offset: page.Offset})
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list pipelines: %v", svcErr))
		return
	}
	if encodeErr := json.NewEncoder(w).Encode(map[string]any{"data": items, "meta": Meta{Total: total, Limit: page.Limit, Offset: page.Offset}}); encodeErr != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
}

func (h *PipelineHandler) GetPipeline(w http.ResponseWriter, r *http.Request) {
	wsID, wsErr := getWorkspaceID(r.Context())
	if wsErr != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}
	id := chi.URLParam(r, "id")
	out, svcErr := h.service.Get(r.Context(), wsID, id)
	if errors.Is(svcErr, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "pipeline not found")
		return
	}
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get pipeline: %v", svcErr))
		return
	}
	if encodeErr := json.NewEncoder(w).Encode(out); encodeErr != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
}

func (h *PipelineHandler) UpdatePipeline(w http.ResponseWriter, r *http.Request) {
	wsID, wsErr := getWorkspaceID(r.Context())
	if wsErr != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}
	id := chi.URLParam(r, "id")
	existing, svcErr := h.service.Get(r.Context(), wsID, id)
	if errors.Is(svcErr, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "pipeline not found")
		return
	}
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get pipeline: %v", svcErr))
		return
	}
	var req UpdatePipelineRequest
	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	out, svcErr := h.service.Update(r.Context(), wsID, id, crm.UpdatePipelineInput{
		Name:       coalesce(req.Name, existing.Name),
		EntityType: coalesce(req.EntityType, existing.EntityType),
		Settings:   req.Settings,
	})
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update pipeline: %v", svcErr))
		return
	}
	json.NewEncoder(w).Encode(out) //nolint:errcheck
}

func (h *PipelineHandler) DeletePipeline(w http.ResponseWriter, r *http.Request) {
	wsID, wsErr := getWorkspaceID(r.Context())
	if wsErr != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}
	id := chi.URLParam(r, "id")
	if svcErr := h.service.Delete(r.Context(), wsID, id); svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete pipeline: %v", svcErr))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *PipelineHandler) CreateStage(w http.ResponseWriter, r *http.Request) {
	pipelineID := chi.URLParam(r, "id")
	var req CreatePipelineStageRequest
	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
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
	out, svcErr := h.service.CreateStage(r.Context(), crm.CreatePipelineStageInput{
		PipelineID:     pipelineID,
		Name:           req.Name,
		Position:       req.Position,
		Probability:    req.Probability,
		SLAHours:       req.SLAHours,
		RequiredFields: req.RequiredFields,
	})
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create stage: %v", svcErr))
		return
	}
	w.WriteHeader(http.StatusCreated)
	if encodeErr := json.NewEncoder(w).Encode(out); encodeErr != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
}

func (h *PipelineHandler) ListStages(w http.ResponseWriter, r *http.Request) {
	pipelineID := chi.URLParam(r, "id")
	items, svcErr := h.service.ListStages(r.Context(), pipelineID)
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list stages: %v", svcErr))
		return
	}
	if encodeErr := json.NewEncoder(w).Encode(map[string]any{"data": items, "meta": Meta{Total: len(items), Limit: len(items), Offset: 0}}); encodeErr != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
}

func (h *PipelineHandler) UpdateStage(w http.ResponseWriter, r *http.Request) {
	stageID, existing, ok := h.getStageForUpdate(w, r)
	if !ok {
		return
	}
	var req UpdatePipelineStageRequest
	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req = fillStageDefaults(req, existing)
	out, svcErr := h.service.UpdateStage(r.Context(), stageID, crm.UpdatePipelineStageInput{
		Name:           req.Name,
		Position:       req.Position,
		Probability:    req.Probability,
		SLAHours:       req.SLAHours,
		RequiredFields: req.RequiredFields,
	})
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update stage: %v", svcErr))
		return
	}
	if encodeErr := json.NewEncoder(w).Encode(out); encodeErr != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
}

func (h *PipelineHandler) getStageForUpdate(w http.ResponseWriter, r *http.Request) (string, *crm.PipelineStage, bool) {
	stageID := chi.URLParam(r, "stage_id")
	existing, svcErr := h.service.GetStage(r.Context(), stageID)
	if errors.Is(svcErr, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "stage not found")
		return "", nil, false
	}
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get stage: %v", svcErr))
		return "", nil, false
	}
	return stageID, existing, true
}

func fillStageDefaults(req UpdatePipelineStageRequest, existing *crm.PipelineStage) UpdatePipelineStageRequest {
	if req.Name == "" {
		req.Name = existing.Name
	}
	if req.Position == 0 {
		req.Position = existing.Position
	}
	return req
}

func (h *PipelineHandler) DeleteStage(w http.ResponseWriter, r *http.Request) {
	stageID := chi.URLParam(r, "stage_id")
	if stageID == "" {
		stageID = chi.URLParam(r, "id")
	}
	if svcErr := h.service.DeleteStage(r.Context(), stageID); svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete stage: %v", svcErr))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
