// Task 1.5: HTTP handlers for Lead CRUD endpoints
package handlers

import (
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
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}

	var req CreateLeadRequest
	if !decodeBodyJSON(w, r, &req) {
		return
	}

	// Validate required fields
	if req.OwnerID == "" {
		writeError(w, http.StatusBadRequest, "ownerId is required")
		return
	}

	// Create lead via service
	lead, svcErr := h.leadService.Create(ctx, crm.CreateLeadInput{
		WorkspaceID: wsID,
		ContactID:   req.ContactID,
		AccountID:   req.AccountID,
		Source:      req.Source,
		Status:      req.Status,
		OwnerID:     req.OwnerID,
		Score:       req.Score,
		Metadata:    req.Metadata,
	})
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create lead: %v", svcErr))
		return
	}

	// Write response
	w.WriteHeader(http.StatusCreated)
	if !writeJSONOr500(w, leadToResponse(lead)) {
		return
	}
}

// GetLead handles GET /api/v1/leads/{id}
// Task 1.5: Retrieve a single lead by ID (with multi-tenancy isolation)
func (h *LeadHandler) GetLead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}

	leadID := chi.URLParam(r, paramID)
	if leadID == "" {
		writeError(w, http.StatusBadRequest, errLeadIDRequired)
		return
	}

	// Get lead via service
	lead, svcErr := h.leadService.Get(ctx, wsID, leadID)
	if handleGetError(w, svcErr, errLeadNotFound, errFailedToGetLead) {
		return
	}

	// Write response
	if !writeJSONOr500(w, leadToResponse(lead)) {
		return
	}
}

// ListLeads handles GET /api/v1/leads with pagination and owner filter
// Task 1.5: List leads with pagination filters
func (h *LeadHandler) ListLeads(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
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
		var listErr error
		leads, listErr = h.leadService.ListByOwner(ctx, wsID, ownerID)
		if listErr != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list leads: %v", listErr))
			return
		}
		total = len(leads)
		// Apply pagination manually for list by owner
		leads = applyPagination(leads, page.Limit, page.Offset)
	} else {
		// List all with pagination
		var listErr error
		leads, total, listErr = h.leadService.List(ctx, wsID, crm.ListLeadsInput{
			Limit:  page.Limit,
			Offset: page.Offset,
		})
		if listErr != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list leads: %v", listErr))
			return
		}
	}

	// Build response
	responses := make([]LeadResponse, len(leads))
	for i, lead := range leads {
		responses[i] = leadToResponse(lead)
	}

	if !writePaginatedOr500(w, responses, total, page) {
		return
	}
}

// UpdateLead handles PUT /api/v1/leads/{id}
// Task 1.5: Update a lead (partial update allowed)
func (h *LeadHandler) UpdateLead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}

	leadID, existing, ok := h.getLeadForUpdate(w, r, wsID)
	if !ok {
		return
	}

	var req UpdateLeadRequest
	if !decodeBodyJSON(w, r, &req) {
		return
	}

	// Merge request with existing values
	updateInput := buildLeadUpdateInput(req, existing)

	// Update lead via service
	updated, upErr := h.leadService.Update(ctx, wsID, leadID, updateInput)
	if upErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update lead: %v", upErr))
		return
	}

	// Write response
	if !writeJSONOr500(w, leadToResponse(updated)) {
		return
	}
}

func (h *LeadHandler) getLeadForUpdate(w http.ResponseWriter, r *http.Request, wsID string) (string, *crm.Lead, bool) {
	return getEntityForUpdate[crm.Lead](w, r, wsID, errLeadIDRequired, errLeadNotFound, errFailedToGetLead, h.leadService.Get)
}

// DeleteLead handles DELETE /api/v1/leads/{id}
// Task 1.5: Soft delete a lead (sets deleted_at timestamp)
func (h *LeadHandler) DeleteLead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}

	leadID, ok := ensureEntityExistsBeforeDelete[crm.Lead](w, r, wsID, errLeadIDRequired, errLeadNotFound, errFailedToGetLead, h.leadService.Get)
	if !ok {
		return
	}

	// Delete lead via service (soft delete)
	if delErr := h.leadService.Delete(ctx, wsID, leadID); delErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete lead: %v", delErr))
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
		CreatedAt:   lead.CreatedAt.Format(timeFormatISO),
		UpdatedAt:   lead.UpdatedAt.Format(timeFormatISO),
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
