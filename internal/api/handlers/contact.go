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
	wsID, wsErr := getWorkspaceID(ctx)
	if wsErr != nil {
		writeError(w, http.StatusBadRequest, errMissingWorkspaceID)
		return
	}

	var req CreateContactRequest
	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		writeError(w, http.StatusBadRequest, errInvalidBody)
		return
	}

	if !isContactRequestValid(req) {
		writeError(w, http.StatusBadRequest, "accountId, firstName, lastName and ownerId are required")
		return
	}

	contact, svcErr := h.contactService.Create(ctx, crm.CreateContactInput{
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
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create contact: %v", svcErr))
		return
	}

	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(http.StatusCreated)
	if encodeErr := json.NewEncoder(w).Encode(contactToResponse(contact)); encodeErr != nil {
		writeError(w, http.StatusInternalServerError, errFailedToEncode)
		return
	}
}

// GetContact handles GET /api/v1/contacts/{id}
func (h *ContactHandler) GetContact(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID, wsErr := getWorkspaceID(ctx)
	if wsErr != nil {
		writeError(w, http.StatusBadRequest, errMissingWorkspaceID)
		return
	}

	contactID := chi.URLParam(r, paramID)
	if contactID == "" {
		writeError(w, http.StatusBadRequest, errContactIDRequired)
		return
	}

	contact, svcErr := h.contactService.Get(ctx, wsID, contactID)
	if errors.Is(svcErr, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, errContactNotFound)
		return
	}
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf(errFailedToGetContact, svcErr))
		return
	}

	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(http.StatusOK)
	if encodeErr := json.NewEncoder(w).Encode(contactToResponse(contact)); encodeErr != nil {
		writeError(w, http.StatusInternalServerError, errFailedToEncode)
		return
	}
}

// ListContacts handles GET /api/v1/contacts
func (h *ContactHandler) ListContacts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID, wsErr := getWorkspaceID(ctx)
	if wsErr != nil {
		writeError(w, http.StatusBadRequest, errMissingWorkspaceID)
		return
	}

	page := parsePaginationParams(r)

	contacts, total, svcErr := h.contactService.List(ctx, wsID, crm.ListContactsInput{
		Limit:  page.Limit,
		Offset: page.Offset,
	})
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list contacts: %v", svcErr))
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

	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(http.StatusOK)
	if encodeErr := json.NewEncoder(w).Encode(resp); encodeErr != nil {
		writeError(w, http.StatusInternalServerError, errFailedToEncode)
		return
	}
}

// ListContactsByAccount handles GET /api/v1/accounts/{account_id}/contacts
func (h *ContactHandler) ListContactsByAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID, wsErr := getWorkspaceID(ctx)
	if wsErr != nil {
		writeError(w, http.StatusBadRequest, errMissingWorkspaceID)
		return
	}

	accountID := chi.URLParam(r, "account_id")
	if accountID == "" {
		writeError(w, http.StatusBadRequest, "account_id is required")
		return
	}

	contacts, svcErr := h.contactService.ListByAccount(ctx, wsID, accountID)
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list contacts by account: %v", svcErr))
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

	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(http.StatusOK)
	if encodeErr := json.NewEncoder(w).Encode(resp); encodeErr != nil {
		writeError(w, http.StatusInternalServerError, errFailedToEncode)
		return
	}
}

// UpdateContact handles PUT /api/v1/contacts/{id}
func (h *ContactHandler) UpdateContact(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID, wsErr := getWorkspaceID(ctx)
	if wsErr != nil {
		writeError(w, http.StatusBadRequest, errMissingWorkspaceID)
		return
	}

	contactID, existing, ok := h.getContactForUpdate(w, r, wsID)
	if !ok {
		return
	}

	var req UpdateContactRequest
	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		writeError(w, http.StatusBadRequest, errInvalidBody)
		return
	}

	updated, svcErr := h.contactService.Update(ctx, wsID, contactID, buildUpdateContactInput(req, existing))
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update contact: %v", svcErr))
		return
	}

	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(http.StatusOK)
	if encodeErr := json.NewEncoder(w).Encode(contactToResponse(updated)); encodeErr != nil {
		writeError(w, http.StatusInternalServerError, errFailedToEncode)
		return
	}
}

func (h *ContactHandler) getContactForUpdate(w http.ResponseWriter, r *http.Request, wsID string) (string, *crm.Contact, bool) {
	ctx := r.Context()
	contactID := chi.URLParam(r, paramID)
	if contactID == "" {
		writeError(w, http.StatusBadRequest, errContactIDRequired)
		return "", nil, false
	}

	existing, svcErr := h.contactService.Get(ctx, wsID, contactID)
	if errors.Is(svcErr, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, errContactNotFound)
		return "", nil, false
	}
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf(errFailedToGetContact, svcErr))
		return "", nil, false
	}

	return contactID, existing, true
}

// DeleteContact handles DELETE /api/v1/contacts/{id}
func (h *ContactHandler) DeleteContact(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID, wsErr := getWorkspaceID(ctx)
	if wsErr != nil {
		writeError(w, http.StatusBadRequest, errMissingWorkspaceID)
		return
	}

	contactID := chi.URLParam(r, paramID)
	if contactID == "" {
		writeError(w, http.StatusBadRequest, errContactIDRequired)
		return
	}

	_, svcErr := h.contactService.Get(ctx, wsID, contactID)
	if errors.Is(svcErr, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, errContactNotFound)
		return
	}
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf(errFailedToGetContact, svcErr))
		return
	}

	if delErr := h.contactService.Delete(ctx, wsID, contactID); delErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete contact: %v", delErr))
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
		CreatedAt:   c.CreatedAt.Format(timeFormatISO),
		UpdatedAt:   c.UpdatedAt.Format(timeFormatISO),
		DeletedAt:   formatDeletedAt(c.DeletedAt),
	}
}
