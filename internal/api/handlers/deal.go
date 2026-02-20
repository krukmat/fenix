package handlers

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
)

type DealHandler struct{ service *crm.DealService }

func NewDealHandler(service *crm.DealService) *DealHandler { return &DealHandler{service: service} }

type CreateDealRequest struct {
	AccountID     string   `json:"accountId"`
	ContactID     string   `json:"contactId,omitempty"`
	PipelineID    string   `json:"pipelineId"`
	StageID       string   `json:"stageId"`
	OwnerID       string   `json:"ownerId"`
	Title         string   `json:"title"`
	Amount        *float64 `json:"amount,omitempty"`
	Currency      string   `json:"currency,omitempty"`
	ExpectedClose string   `json:"expectedClose,omitempty"`
	Status        string   `json:"status,omitempty"`
	Metadata      string   `json:"metadata,omitempty"`
}

type UpdateDealRequest = CreateDealRequest

func (h *DealHandler) CreateDeal(w http.ResponseWriter, r *http.Request) {
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	var req CreateDealRequest
	if !decodeBodyJSON(w, r, &req) {
		return
	}
	if !isDealRequestValid(req) {
		writeError(w, http.StatusBadRequest, "accountId, pipelineId, stageId, ownerId and title are required")
		return
	}
	out, svcErr := h.service.Create(r.Context(), crm.CreateDealInput{
		WorkspaceID:   wsID,
		AccountID:     req.AccountID,
		ContactID:     req.ContactID,
		PipelineID:    req.PipelineID,
		StageID:       req.StageID,
		OwnerID:       req.OwnerID,
		Title:         req.Title,
		Amount:        req.Amount,
		Currency:      req.Currency,
		ExpectedClose: req.ExpectedClose,
		Status:        req.Status,
		Metadata:      req.Metadata,
	})
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create deal: %v", svcErr))
		return
	}
	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(http.StatusCreated)
	_ = writeJSONOr500(w, out)
}

// isDealRequestValid checks required fields for CreateDeal.
// Task 1.6.15: Extracted to reduce cyclomatic complexity of CreateDeal (was 9).
func isDealRequestValid(req CreateDealRequest) bool {
	return req.AccountID != "" && req.PipelineID != "" && req.StageID != "" &&
		req.OwnerID != "" && req.Title != ""
}

func (h *DealHandler) GetDeal(w http.ResponseWriter, r *http.Request) {
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, paramID)
	out, svcErr := h.service.Get(r.Context(), wsID, id)
	if handleGetError(w, svcErr, "deal not found", "failed to get deal: %v") {
		return
	}
	if !writeJSONOr500(w, out) {
		return
	}
}

func (h *DealHandler) ListDeals(w http.ResponseWriter, r *http.Request) {
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	page := parsePaginationParams(r)
	items, total, svcErr := h.service.List(r.Context(), wsID, crm.ListDealsInput{Limit: page.Limit, Offset: page.Offset})
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list deals: %v", svcErr))
		return
	}
	if !writePaginatedOr500(w, items, total, page) {
		return
	}
}

func (h *DealHandler) UpdateDeal(w http.ResponseWriter, r *http.Request) {
	handleEntityUpdate[
		crm.Deal,
		UpdateDealRequest,
		crm.UpdateDealInput,
		crm.Deal,
	](
		w,
		r,
		"deal not found",
		"failed to get deal: %v",
		"failed to update deal: %v",
		h.service.Get,
		buildUpdateDealInput,
		h.service.Update,
	)
}

// buildUpdateDealInput merges update request with existing deal values.
// Task 1.6.15: Extracted to reduce cyclomatic complexity of UpdateDeal (was 11).
func buildUpdateDealInput(req UpdateDealRequest, existing *crm.Deal) crm.UpdateDealInput {
	return crm.UpdateDealInput{
		AccountID:     coalesce(req.AccountID, existing.AccountID),
		ContactID:     req.ContactID,
		PipelineID:    coalesce(req.PipelineID, existing.PipelineID),
		StageID:       coalesce(req.StageID, existing.StageID),
		OwnerID:       coalesce(req.OwnerID, existing.OwnerID),
		Title:         coalesce(req.Title, existing.Title),
		Amount:        req.Amount,
		Currency:      req.Currency,
		ExpectedClose: req.ExpectedClose,
		Status:        req.Status,
		Metadata:      req.Metadata,
	}
}

func (h *DealHandler) DeleteDeal(w http.ResponseWriter, r *http.Request) {
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, paramID)
	if delErr := h.service.Delete(r.Context(), wsID, id); delErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete deal: %v", delErr))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
