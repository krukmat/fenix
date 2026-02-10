// Task 1.4: HTTP handlers for Contact CRUD endpoints
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

// ContactHandler handles HTTP requests for contact CRUD operations.
type ContactHandler struct {
	contactService *crm.ContactService
}

// NewContactHandler creates a new ContactHandler instance.
func NewContactHandler(contactService *crm.ContactService) *ContactHandler {
	return &ContactHandler{contactService: contactService}
}

// CreateContactRequest is the request body for creating a contact.
type CreateContactRequest struct {
	AccountID string `json:"accountId"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email,omitempty"`
	Phone     string `json:"phone,omitempty"`
	Title     string `json:"title,omitempty"`
	Status    string `json:"status,omitempty"`
	OwnerID   string `json:"ownerId"`
	Metadata  string `json:"metadata,omitempty"`
}

// UpdateContactRequest is the request body for updating a contact.
type UpdateContactRequest struct {
	AccountID string `json:"accountId,omitempty"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	Email     string `json:"email,omitempty"`
	Phone     string `json:"phone,omitempty"`
	Title     string `json:"title,omitempty"`
	Status    string `json:"status,omitempty"`
	OwnerID   string `json:"ownerId,omitempty"`
	Metadata  string `json:"metadata,omitempty"`
}

// ContactResponse is the response body for contact operations.
type ContactResponse struct {
	ID          string  `json:"id"`
	WorkspaceID string  `json:"workspaceId"`
	AccountID   string  `json:"accountId"`
	FirstName   string  `json:"firstName"`
	LastName    string  `json:"lastName"`
	Email       *string `json:"email,omitempty"`
	Phone       *string `json:"phone,omitempty"`
	Title       *string `json:"title,omitempty"`
	Status      string  `json:"status"`
	OwnerID     string  `json:"ownerId"`
	Metadata    *string `json:"metadata,omitempty"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
	DeletedAt   *string `json:"deletedAt,omitempty"`
}

// ListContactsResponse is the response body for listing contacts.
type ListContactsResponse struct {
	Data []ContactResponse `json:"data"`
	Meta Meta              `json:"meta"`
}

// CreateContact handles POST /api/v1/contacts
func (h *ContactHandler) CreateContact(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID, err := getWorkspaceID(ctx)
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}

	var req CreateContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !isContactRequestValid(req) {
		writeError(w, http.StatusBadRequest, "accountId, firstName, lastName and ownerId are required")
		return
	}

	contact, err := h.contactService.Create(ctx, crm.CreateContactInput{
		WorkspaceID: wsID,
		AccountID:   req.AccountID,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Email:       req.Email,
		Phone:       req.Phone,
		Title:       req.Title,
		Status:      req.Status,
		OwnerID:     req.OwnerID,
		Metadata:    req.Metadata,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create contact: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(contactToResponse(contact)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
}

// GetContact handles GET /api/v1/contacts/{id}
func (h *ContactHandler) GetContact(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID, err := getWorkspaceID(ctx)
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}

	contactID := chi.URLParam(r, "id")
	if contactID == "" {
		writeError(w, http.StatusBadRequest, "contact id is required")
		return
	}

	contact, err := h.contactService.Get(ctx, wsID, contactID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "contact not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get contact: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(contactToResponse(contact)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
}

// ListContacts handles GET /api/v1/contacts
func (h *ContactHandler) ListContacts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID, err := getWorkspaceID(ctx)
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}

	page := parsePaginationParams(r)

	contacts, total, err := h.contactService.List(ctx, wsID, crm.ListContactsInput{
		Limit:  page.Limit,
		Offset: page.Offset,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list contacts: %v", err))
		return
	}

	responses := make([]ContactResponse, len(contacts))
	for i, c := range contacts {
		responses[i] = contactToResponse(c)
	}

	resp := ListContactsResponse{
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

// ListContactsByAccount handles GET /api/v1/accounts/{account_id}/contacts
func (h *ContactHandler) ListContactsByAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID, err := getWorkspaceID(ctx)
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}

	accountID := chi.URLParam(r, "account_id")
	if accountID == "" {
		writeError(w, http.StatusBadRequest, "account_id is required")
		return
	}

	contacts, err := h.contactService.ListByAccount(ctx, wsID, accountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list contacts by account: %v", err))
		return
	}

	responses := make([]ContactResponse, len(contacts))
	for i, c := range contacts {
		responses[i] = contactToResponse(c)
	}

	resp := ListContactsResponse{
		Data: responses,
		Meta: Meta{Total: len(responses), Limit: len(responses), Offset: 0},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
}

// UpdateContact handles PUT /api/v1/contacts/{id}
func (h *ContactHandler) UpdateContact(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID, err := getWorkspaceID(ctx)
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}

	contactID := chi.URLParam(r, "id")
	if contactID == "" {
		writeError(w, http.StatusBadRequest, "contact id is required")
		return
	}

	existing, err := h.contactService.Get(ctx, wsID, contactID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "contact not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get contact: %v", err))
		return
	}

	var req UpdateContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updated, err := h.contactService.Update(ctx, wsID, contactID, buildUpdateContactInput(req, existing))
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update contact: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(contactToResponse(updated)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
}

// DeleteContact handles DELETE /api/v1/contacts/{id}
func (h *ContactHandler) DeleteContact(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID, err := getWorkspaceID(ctx)
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}

	contactID := chi.URLParam(r, "id")
	if contactID == "" {
		writeError(w, http.StatusBadRequest, "contact id is required")
		return
	}

	_, err = h.contactService.Get(ctx, wsID, contactID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "contact not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get contact: %v", err))
		return
	}

	if err := h.contactService.Delete(ctx, wsID, contactID); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete contact: %v", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// isContactRequestValid checks required fields for CreateContact.
// Task 1.6.15: Extracted to reduce cyclomatic complexity of CreateContact (was 8).
func isContactRequestValid(req CreateContactRequest) bool {
	return req.AccountID != "" && req.FirstName != "" && req.LastName != "" && req.OwnerID != ""
}

// buildUpdateContactInput merges the update request with existing values.
// Task 1.6.15: Refactored using coalesce/coalescePtr to reduce cyclomatic complexity (was 14).
func buildUpdateContactInput(req UpdateContactRequest, existing *crm.Contact) crm.UpdateContactInput {
	return crm.UpdateContactInput{
		AccountID: coalesce(req.AccountID, existing.AccountID),
		FirstName: coalesce(req.FirstName, existing.FirstName),
		LastName:  coalesce(req.LastName, existing.LastName),
		Email:     coalescePtr(req.Email, existing.Email),
		Phone:     coalescePtr(req.Phone, existing.Phone),
		Title:     coalescePtr(req.Title, existing.Title),
		Status:    coalesce(req.Status, existing.Status),
		OwnerID:   coalesce(req.OwnerID, existing.OwnerID),
		Metadata:  coalescePtr(req.Metadata, existing.Metadata),
	}
}

func contactToResponse(c *crm.Contact) ContactResponse {
	return ContactResponse{
		ID:          c.ID,
		WorkspaceID: c.WorkspaceID,
		AccountID:   c.AccountID,
		FirstName:   c.FirstName,
		LastName:    c.LastName,
		Email:       c.Email,
		Phone:       c.Phone,
		Title:       c.Title,
		Status:      c.Status,
		OwnerID:     c.OwnerID,
		Metadata:    c.Metadata,
		CreatedAt:   c.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   c.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		DeletedAt:   formatDeletedAt(c.DeletedAt),
	}
}
