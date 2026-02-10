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

type CaseHandler struct{ service *crm.CaseService }

func NewCaseHandler(service *crm.CaseService) *CaseHandler { return &CaseHandler{service: service} }

type CreateCaseRequest struct {
	AccountID   string `json:"accountId,omitempty"`
	ContactID   string `json:"contactId,omitempty"`
	PipelineID  string `json:"pipelineId,omitempty"`
	StageID     string `json:"stageId,omitempty"`
	OwnerID     string `json:"ownerId"`
	Subject     string `json:"subject"`
	Description string `json:"description,omitempty"`
	Priority    string `json:"priority,omitempty"`
	Status      string `json:"status,omitempty"`
	Channel     string `json:"channel,omitempty"`
	SlaConfig   string `json:"slaConfig,omitempty"`
	SlaDeadline string `json:"slaDeadline,omitempty"`
	Metadata    string `json:"metadata,omitempty"`
}

type UpdateCaseRequest = CreateCaseRequest

func (h *CaseHandler) CreateCase(w http.ResponseWriter, r *http.Request) {
	wsID, err := getWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}
	var req CreateCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.OwnerID == "" || req.Subject == "" {
		writeError(w, http.StatusBadRequest, "ownerId and subject are required")
		return
	}
	out, err := h.service.Create(r.Context(), crm.CreateCaseInput{
		WorkspaceID: wsID,
		AccountID:   req.AccountID,
		ContactID:   req.ContactID,
		PipelineID:  req.PipelineID,
		StageID:     req.StageID,
		OwnerID:     req.OwnerID,
		Subject:     req.Subject,
		Description: req.Description,
		Priority:    req.Priority,
		Status:      req.Status,
		Channel:     req.Channel,
		SlaConfig:   req.SlaConfig,
		SlaDeadline: req.SlaDeadline,
		Metadata:    req.Metadata,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create case: %v", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(out); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
}

func (h *CaseHandler) GetCase(w http.ResponseWriter, r *http.Request) {
	wsID, err := getWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}
	id := chi.URLParam(r, "id")
	out, err := h.service.Get(r.Context(), wsID, id)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "case not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get case: %v", err))
		return
	}
	if err := json.NewEncoder(w).Encode(out); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
}

func (h *CaseHandler) ListCases(w http.ResponseWriter, r *http.Request) {
	wsID, err := getWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}
	page := parsePaginationParams(r)
	items, total, err := h.service.List(r.Context(), wsID, crm.ListCasesInput{Limit: page.Limit, Offset: page.Offset})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list cases: %v", err))
		return
	}
	if err := json.NewEncoder(w).Encode(map[string]any{"data": items, "meta": Meta{Total: total, Limit: page.Limit, Offset: page.Offset}}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
}

func (h *CaseHandler) UpdateCase(w http.ResponseWriter, r *http.Request) {
	wsID, err := getWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}
	id := chi.URLParam(r, "id")
	existing, err := h.service.Get(r.Context(), wsID, id)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "case not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get case: %v", err))
		return
	}
	var req UpdateCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	out, err := h.service.Update(r.Context(), wsID, id, buildUpdateCaseInput(req, existing))
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update case: %v", err))
		return
	}
	json.NewEncoder(w).Encode(out) //nolint:errcheck
}

func (h *CaseHandler) DeleteCase(w http.ResponseWriter, r *http.Request) {
	wsID, err := getWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.service.Delete(r.Context(), wsID, id); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete case: %v", err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// buildUpdateCaseInput merges update request with existing case values.
// Task 1.6.15: Extracted to reduce cyclomatic complexity of UpdateCase (was 10).
func buildUpdateCaseInput(req UpdateCaseRequest, existing *crm.CaseTicket) crm.UpdateCaseInput {
	return crm.UpdateCaseInput{
		AccountID:   req.AccountID,
		ContactID:   req.ContactID,
		PipelineID:  req.PipelineID,
		StageID:     req.StageID,
		OwnerID:     coalesce(req.OwnerID, existing.OwnerID),
		Subject:     coalesce(req.Subject, existing.Subject),
		Description: req.Description,
		Priority:    coalesce(req.Priority, existing.Priority),
		Status:      coalesce(req.Status, existing.Status),
		Channel:     req.Channel,
		SlaConfig:   req.SlaConfig,
		SlaDeadline: req.SlaDeadline,
		Metadata:    req.Metadata,
	}
}
