package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
)

type DealHandler struct{ service *crm.DealService }

func NewDealHandler(service *crm.DealService) *DealHandler { return &DealHandler{service: service} }

const errDealNotFound = "deal not found"

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
		if errors.Is(svcErr, crm.ErrInvalidDealInput) {
			writeError(w, http.StatusBadRequest, svcErr.Error())
			return
		}
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
	if handleGetError(w, svcErr, errDealNotFound, "failed to get deal: %v") {
		return
	}
	if !writeJSONOr500(w, out) {
		return
	}
}

func (h *DealHandler) ListDeals(w http.ResponseWriter, r *http.Request) {
	handleParsedListWithPagination(w, r, parseDealListInput, h.service.List, "failed to list deals: %v")
}

func parseDealListInput(r *http.Request, page paginationParams) (crm.ListDealsInput, error) {
	q := r.URL.Query()
	status := strings.TrimSpace(q.Get(queryStatus))
	ownerID := strings.TrimSpace(q.Get(queryOwnerID))
	accountID := strings.TrimSpace(q.Get(queryAccountID))
	pipelineID := strings.TrimSpace(q.Get("pipeline_id"))
	stageID := strings.TrimSpace(q.Get(paramStageID))
	sortParam := strings.TrimSpace(q.Get("sort"))
	if sortParam == "" {
		sortParam = querySortDesc
	}
	if sortParam != querySortDesc && sortParam != querySortAsc {
		return crm.ListDealsInput{}, fmt.Errorf("invalid sort. allowed: %s, %s", querySortDesc, querySortAsc)
	}

	filterCount := 0
	for _, v := range []string{status, ownerID, accountID, pipelineID, stageID} {
		if v != "" {
			filterCount++
		}
	}
	if filterCount > 1 {
		return crm.ListDealsInput{}, fmt.Errorf("only one filter is allowed: status, owner_id, account_id, pipeline_id, stage_id")
	}

	return crm.ListDealsInput{
		Limit:      page.Limit,
		Offset:     page.Offset,
		Status:     status,
		OwnerID:    ownerID,
		AccountID:  accountID,
		PipelineID: pipelineID,
		StageID:    stageID,
		Sort:       sortParam,
	}, nil
}

func (h *DealHandler) UpdateDeal(w http.ResponseWriter, r *http.Request) {
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}

	id := chiURLParamID(r)
	existing, svcErr := h.service.Get(r.Context(), wsID, id)
	if handleGetError(w, svcErr, errDealNotFound, "failed to get deal: %v") {
		return
	}

	var req UpdateDealRequest
	if !decodeBodyJSON(w, r, &req) {
		return
	}

	out, upErr := h.service.Update(r.Context(), wsID, id, buildUpdateDealInput(req, existing))
	if upErr != nil {
		if errors.Is(upErr, crm.ErrInvalidDealInput) {
			writeError(w, http.StatusBadRequest, upErr.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update deal: %v", upErr))
		return
	}

	_ = writeJSONOr500(w, out)
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
	handleDeleteWithNotFound(w, r, errDealNotFound, sql.ErrNoRows, errDealNotFound, "failed to delete deal: %v", h.service.Delete)
}
