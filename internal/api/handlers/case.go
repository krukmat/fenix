package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
)

type CaseHandler struct {
	service       *crm.CaseService
	signalCounter activeSignalCounter
}

func NewCaseHandler(service *crm.CaseService) *CaseHandler { return &CaseHandler{service: service} }

func NewCaseHandlerWithSignalCounter(service *crm.CaseService, signalCounter activeSignalCounter) *CaseHandler {
	return &CaseHandler{service: service, signalCounter: signalCounter}
}

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
	wsID, ok := requireWorkspaceForCaseMutation(w, r)
	if !ok {
		return
	}
	req, ok := decodeCreateCaseRequest(w, r)
	if !ok {
		return
	}
	out, svcErr := h.service.Create(r.Context(), buildCreateCaseInput(wsID, req))
	if handleCaseCreateError(w, svcErr) {
		return
	}
	writeCreatedJSON(w, out)
}

func requireWorkspaceForCaseMutation(w http.ResponseWriter, r *http.Request) (string, bool) {
	wsID, wsErr := getWorkspaceID(r.Context())
	if wsErr != nil {
		writeError(w, http.StatusBadRequest, errMissingWorkspaceID)
		return "", false
	}
	return wsID, true
}

func decodeCreateCaseRequest(w http.ResponseWriter, r *http.Request) (CreateCaseRequest, bool) {
	var req CreateCaseRequest
	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		writeError(w, http.StatusBadRequest, errInvalidBody)
		return CreateCaseRequest{}, false
	}
	if req.OwnerID == "" || req.Subject == "" {
		writeError(w, http.StatusBadRequest, "ownerId and subject are required")
		return CreateCaseRequest{}, false
	}
	return req, true
}

func buildCreateCaseInput(workspaceID string, req CreateCaseRequest) crm.CreateCaseInput {
	return crm.CreateCaseInput{
		WorkspaceID: workspaceID,
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
	}
}

func handleCaseCreateError(w http.ResponseWriter, svcErr error) bool {
	if svcErr == nil {
		return false
	}
	if errors.Is(svcErr, crm.ErrInvalidCaseInput) {
		writeError(w, http.StatusBadRequest, svcErr.Error())
		return true
	}
	writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create case: %v", svcErr))
	return true
}

func writeCreatedJSON(w http.ResponseWriter, out any) {
	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(http.StatusCreated)
	if encodeErr := json.NewEncoder(w).Encode(out); encodeErr != nil {
		writeError(w, http.StatusInternalServerError, errFailedToEncode)
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
	h.attachActiveSignalCount(r.Context(), wsID, out)
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
	input, err := parseCaseListInput(r, page)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	items, total, svcErr := h.service.List(r.Context(), wsID, input)
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list cases: %v", svcErr))
		return
	}
	counts := countActiveSignalsByEntity(r.Context(), h.signalCounter, wsID, entityTypeCase, collectEntityIDs(items, func(item *crm.CaseTicket) string {
		return item.ID
	}))
	for _, item := range items {
		if count, ok := counts[item.ID]; ok {
			item.ActiveSignalCount = intPtr(count)
		}
	}
	if !writePaginatedOr500(w, items, total, page) {
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
		if errors.Is(svcErr, crm.ErrInvalidCaseInput) {
			writeError(w, http.StatusBadRequest, svcErr.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update case: %v", svcErr))
		return
	}
	json.NewEncoder(w).Encode(out) //nolint:errcheck
}

func (h *CaseHandler) DeleteCase(w http.ResponseWriter, r *http.Request) {
	handleDeleteWithNotFound(w, r, "case id is required", sql.ErrNoRows, errCaseNotFound, "failed to delete case: %v", h.service.Delete)
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

func parseCaseListInput(r *http.Request, page paginationParams) (crm.ListCasesInput, error) {
	q := r.URL.Query()
	status := strings.TrimSpace(q.Get(queryStatus))
	priority := strings.TrimSpace(q.Get("priority"))
	ownerID := strings.TrimSpace(q.Get(queryOwnerID))
	accountID := strings.TrimSpace(q.Get(queryAccountID))
	sortParam := strings.TrimSpace(q.Get("sort"))
	if sortParam == "" {
		sortParam = querySortDesc
	}
	if sortParam != querySortDesc && sortParam != querySortAsc {
		return crm.ListCasesInput{}, fmt.Errorf("invalid sort. allowed: %s, %s", querySortDesc, querySortAsc)
	}

	filterCount := 0
	for _, v := range []string{status, ownerID, accountID} {
		if v != "" {
			filterCount++
		}
	}
	if filterCount > 1 {
		return crm.ListCasesInput{}, fmt.Errorf("only one filter is allowed: status, owner_id, account_id")
	}

	return crm.ListCasesInput{
		Limit:     page.Limit,
		Offset:    page.Offset,
		Status:    status,
		Priority:  priority,
		OwnerID:   ownerID,
		AccountID: accountID,
		Sort:      sortParam,
	}, nil
}

func (h *CaseHandler) attachActiveSignalCount(ctx context.Context, workspaceID string, item *crm.CaseTicket) {
	if item == nil || item.ID == "" {
		return
	}
	counts := countActiveSignalsByEntity(ctx, h.signalCounter, workspaceID, entityTypeCase, []string{item.ID})
	if count, ok := counts[item.ID]; ok {
		item.ActiveSignalCount = intPtr(count)
	}
}
