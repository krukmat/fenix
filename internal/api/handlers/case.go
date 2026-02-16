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
	SLAConfig   string `json:"slaConfig,omitempty"`
	SLADeadline string `json:"slaDeadline,omitempty"`
	Metadata    string `json:"metadata,omitempty"`
}

type UpdateCaseRequest = CreateCaseRequest

func (h *CaseHandler) CreateCase(w http.ResponseWriter, r *http.Request) {
	wsID, wsErr := getWorkspaceID(r.Context())
	if wsErr != nil {
		writeError(w, http.StatusBadRequest, errMissingWorkspaceID)
		return
	}
	var req CreateCaseRequest
	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		writeError(w, http.StatusBadRequest, errInvalidBody)
		return
	}
	if req.OwnerID == "" || req.Subject == "" {
		writeError(w, http.StatusBadRequest, "ownerId and subject are required")
		return
	}
	out, svcErr := h.service.Create(r.Context(), crm.CreateCaseInput{
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
		SLAConfig:   req.SLAConfig,
		SLADeadline: req.SLADeadline,
		Metadata:    req.Metadata,
	})
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create case: %v", svcErr))
		return
	}
	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(http.StatusCreated)
	if encodeErr := json.NewEncoder(w).Encode(out); encodeErr != nil {
		writeError(w, http.StatusInternalServerError, errFailedToEncode)
		return
	}
}

func (h *CaseHandler) GetCase(w http.ResponseWriter, r *http.Request) {
	wsID, wsErr := getWorkspaceID(r.Context())
	if wsErr != nil {
		writeError(w, http.StatusBadRequest, errMissingWorkspaceID)
		return
	}
	id := chi.URLParam(r, paramID)
	out, svcErr := h.service.Get(r.Context(), wsID, id)
	if errors.Is(svcErr, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, errCaseNotFound)
		return
	}
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get case: %v", svcErr))
		return
	}
	if encodeErr := json.NewEncoder(w).Encode(out); encodeErr != nil {
		writeError(w, http.StatusInternalServerError, errFailedToEncode)
		return
	}
}

func (h *CaseHandler) ListCases(w http.ResponseWriter, r *http.Request) {
	wsID, wsErr := getWorkspaceID(r.Context())
	if wsErr != nil {
		writeError(w, http.StatusBadRequest, errMissingWorkspaceID)
		return
	}
	page := parsePaginationParams(r)
	items, total, svcErr := h.service.List(r.Context(), wsID, crm.ListCasesInput{Limit: page.Limit, Offset: page.Offset})
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list cases: %v", svcErr))
		return
	}
	if encodeErr := json.NewEncoder(w).Encode(map[string]any{"data": items, "meta": Meta{Total: total, Limit: page.Limit, Offset: page.Offset}}); encodeErr != nil {
		writeError(w, http.StatusInternalServerError, errFailedToEncode)
		return
	}
}

func (h *CaseHandler) UpdateCase(w http.ResponseWriter, r *http.Request) {
	wsID, wsErr := getWorkspaceID(r.Context())
	if wsErr != nil {
		writeError(w, http.StatusBadRequest, errMissingWorkspaceID)
		return
	}
	id := chi.URLParam(r, paramID)
	existing, svcErr := h.service.Get(r.Context(), wsID, id)
	if errors.Is(svcErr, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, errCaseNotFound)
		return
	}
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get case: %v", svcErr))
		return
	}
	var req UpdateCaseRequest
	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		writeError(w, http.StatusBadRequest, errInvalidBody)
		return
	}
	out, svcErr := h.service.Update(r.Context(), wsID, id, buildUpdateCaseInput(req, existing))
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update case: %v", svcErr))
		return
	}
	json.NewEncoder(w).Encode(out) //nolint:errcheck
}

func (h *CaseHandler) DeleteCase(w http.ResponseWriter, r *http.Request) {
	wsID, wsErr := getWorkspaceID(r.Context())
	if wsErr != nil {
		writeError(w, http.StatusBadRequest, errMissingWorkspaceID)
		return
	}
	id := chi.URLParam(r, paramID)
	if svcErr := h.service.Delete(r.Context(), wsID, id); svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete case: %v", svcErr))
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
		SLAConfig:   req.SLAConfig,
		SLADeadline: req.SLADeadline,
		Metadata:    req.Metadata,
	}
}
