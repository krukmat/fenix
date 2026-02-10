// Task 1.5: HTTP handlers for Lead CRUD endpoints
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

// LeadHandler handles HTTP requests for lead CRUD operations.
type LeadHandler struct {
	leadService *crm.LeadService
}

// NewLeadHandler creates a new LeadHandler instance.
func NewLeadHandler(leadService *crm.LeadService) *LeadHandler {
	return &LeadHandler{
		leadService: leadService,
	}
}

// CreateLeadRequest is the request body for creating a lead.
type CreateLeadRequest struct {
	ContactID string   `json:"contactId,omitempty"`
	AccountID string   `json:"accountId,omitempty"`
	Source    string   `json:"source,omitempty"`
	Status    string   `json:"status,omitempty"`
	OwnerID   string   `json:"ownerId"`
	Score     *float64 `json:"score,omitempty"`
	Metadata  string   `json:"metadata,omitempty"`
}

// UpdateLeadRequest is the request body for updating a lead.
type UpdateLeadRequest struct {
	ContactID string   `json:"contactId,omitempty"`
	AccountID string   `json:"accountId,omitempty"`
	Source    string   `json:"source,omitempty"`
	Status    string   `json:"status,omitempty"`
	OwnerID   string   `json:"ownerId,omitempty"`
	Score     *float64 `json:"score,omitempty"`
	Metadata  string   `json:"metadata,omitempty"`
}

// LeadResponse is the response body for lead operations.
type LeadResponse struct {
	ID          string   `json:"id"`
	WorkspaceID string   `json:"workspaceId"`
	ContactID   *string  `json:"contactId,omitempty"`
	AccountID   *string  `json:"accountId,omitempty"`
	Source      *string  `json:"source,omitempty"`
	Status      string   `json:"status"`
	OwnerID     string   `json:"ownerId"`
	Score       *float64 `json:"score,omitempty"`
	Metadata    *string  `json:"metadata,omitempty"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
	DeletedAt   *string  `json:"deletedAt,omitempty"`
}

// PaginatedResponse is a generic paginated response structure.
type PaginatedResponse struct {
	Data []LeadResponse `json:"data"`
	Meta Meta           `json:"meta"`
}

// CreateLead handles POST /api/v1/leads
// Task 1.5: Create a new lead (CRUD + Multi-tenancy)
func (h *LeadHandler) CreateLead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID, err := getWorkspaceID(ctx)
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}

	var req CreateLeadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate required fields
	if req.OwnerID == "" {
		writeError(w, http.StatusBadRequest, "ownerId is required")
		return
	}

	// Create lead via service
	lead, err := h.leadService.Create(ctx, crm.CreateLeadInput{
		WorkspaceID: wsID,
		ContactID:   req.ContactID,
		AccountID:   req.AccountID,
		Source:      req.Source,
		Status:      req.Status,
		OwnerID:     req.OwnerID,
		Score:       req.Score,
		Metadata:    req.Metadata,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create lead: %v", err))
		return
	}

	// Write response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(leadToResponse(lead)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
}

// GetLead handles GET /api/v1/leads/{id}
// Task 1.5: Retrieve a single lead by ID (with multi-tenancy isolation)
func (h *LeadHandler) GetLead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID, err := getWorkspaceID(ctx)
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}

	leadID := chi.URLParam(r, "id")
	if leadID == "" {
		writeError(w, http.StatusBadRequest, "lead id is required")
		return
	}

	// Get lead via service
	lead, err := h.leadService.Get(ctx, wsID, leadID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "lead not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get lead: %v", err))
		return
	}

	// Write response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(leadToResponse(lead)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
}

// ListLeads handles GET /api/v1/leads with pagination and owner filter
// Task 1.5: List leads with pagination filters
func (h *LeadHandler) ListLeads(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID, err := getWorkspaceID(ctx)
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}

	// Parse pagination params
	page := parsePaginationParams(r)

	// Check for owner_id filter
	ownerID := r.URL.Query().Get("owner_id")

	var leads []*crm.Lead
	var total int

	if ownerID != "" {
		// List by owner
		leads, err = h.leadService.ListByOwner(ctx, wsID, ownerID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list leads: %v", err))
			return
		}
		total = len(leads)
		// Apply pagination manually for list by owner
		leads = applyPagination(leads, page.Limit, page.Offset)
	} else {
		// List all with pagination
		leads, total, err = h.leadService.List(ctx, wsID, crm.ListLeadsInput{
			Limit:  page.Limit,
			Offset: page.Offset,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list leads: %v", err))
			return
		}
	}

	// Build response
	responses := make([]LeadResponse, len(leads))
	for i, lead := range leads {
		responses[i] = leadToResponse(lead)
	}

	resp := PaginatedResponse{
		Data: responses,
		Meta: Meta{Total: total, Limit: page.Limit, Offset: page.Offset},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
}

// UpdateLead handles PUT /api/v1/leads/{id}
// Task 1.5: Update a lead (partial update allowed)
func (h *LeadHandler) UpdateLead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID, err := getWorkspaceID(ctx)
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}

	leadID, existing, ok := h.getLeadForUpdate(w, r, wsID)
	if !ok {
		return
	}

	var req UpdateLeadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Merge request with existing values
	updateInput := buildLeadUpdateInput(req, existing)

	// Update lead via service
	updated, err := h.leadService.Update(ctx, wsID, leadID, updateInput)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update lead: %v", err))
		return
	}

	// Write response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(leadToResponse(updated)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
}

func (h *LeadHandler) getLeadForUpdate(w http.ResponseWriter, r *http.Request, wsID string) (string, *crm.Lead, bool) {
	ctx := r.Context()
	leadID := chi.URLParam(r, "id")
	if leadID == "" {
		writeError(w, http.StatusBadRequest, "lead id is required")
		return "", nil, false
	}

	existing, err := h.leadService.Get(ctx, wsID, leadID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "lead not found")
		return "", nil, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get lead: %v", err))
		return "", nil, false
	}

	return leadID, existing, true
}

// DeleteLead handles DELETE /api/v1/leads/{id}
// Task 1.5: Soft delete a lead (sets deleted_at timestamp)
func (h *LeadHandler) DeleteLead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID, err := getWorkspaceID(ctx)
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}

	leadID := chi.URLParam(r, "id")
	if leadID == "" {
		writeError(w, http.StatusBadRequest, "lead id is required")
		return
	}

	// Verify lead exists (and is not already soft-deleted) before deleting
	_, err = h.leadService.Get(ctx, wsID, leadID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "lead not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get lead: %v", err))
		return
	}

	// Delete lead via service (soft delete)
	if err := h.leadService.Delete(ctx, wsID, leadID); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete lead: %v", err))
		return
	}

	// Write response (204 No Content)
	w.WriteHeader(http.StatusNoContent)
}

// --- helpers ---

// leadToResponse converts a domain Lead to a LeadResponse.
func leadToResponse(lead *crm.Lead) LeadResponse {
	return LeadResponse{
		ID:          lead.ID,
		WorkspaceID: lead.WorkspaceID,
		ContactID:   lead.ContactID,
		AccountID:   lead.AccountID,
		Source:      lead.Source,
		Status:      lead.Status,
		OwnerID:     lead.OwnerID,
		Score:       lead.Score,
		Metadata:    lead.Metadata,
		CreatedAt:   lead.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   lead.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		DeletedAt:   formatDeletedAt(lead.DeletedAt),
	}
}

// buildLeadUpdateInput merges the update request with existing values.
func buildLeadUpdateInput(req UpdateLeadRequest, existing *crm.Lead) crm.UpdateLeadInput {
	input := crm.UpdateLeadInput{
		ContactID: req.ContactID,
		AccountID: req.AccountID,
		Source:    req.Source,
		Status:    req.Status,
		OwnerID:   req.OwnerID,
		Score:     req.Score,
		Metadata:  req.Metadata,
	}

	// Default required fields to existing values if omitted
	if input.Status == "" {
		input.Status = existing.Status
	}
	if input.OwnerID == "" {
		input.OwnerID = existing.OwnerID
	}

	return input
}

// applyPagination applies pagination to a slice of leads.
func applyPagination(leads []*crm.Lead, limit, offset int) []*crm.Lead {
	if offset >= len(leads) {
		return []*crm.Lead{}
	}
	end := offset + limit
	if end > len(leads) {
		end = len(leads)
	}
	return leads[offset:end]
}
