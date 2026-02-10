// Task 1.3.7: HTTP handlers for Account CRUD endpoints
package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
)

// AccountHandler handles HTTP requests for account CRUD operations.
type AccountHandler struct {
	accountService *crm.AccountService
}

// NewAccountHandler creates a new AccountHandler instance.
func NewAccountHandler(accountService *crm.AccountService) *AccountHandler {
	return &AccountHandler{
		accountService: accountService,
	}
}

// CreateAccountRequest is the request body for creating an account.
type CreateAccountRequest struct {
	Name        string `json:"name"`
	Domain      string `json:"domain,omitempty"`
	Industry    string `json:"industry,omitempty"`
	SizeSegment string `json:"sizeSegment,omitempty"`
	OwnerID     string `json:"ownerId"`
	Address     string `json:"address,omitempty"`
	Metadata    string `json:"metadata,omitempty"`
}

// UpdateAccountRequest is the request body for updating an account.
type UpdateAccountRequest struct {
	Name        string `json:"name,omitempty"`
	Domain      string `json:"domain,omitempty"`
	Industry    string `json:"industry,omitempty"`
	SizeSegment string `json:"sizeSegment,omitempty"`
	OwnerID     string `json:"ownerId,omitempty"`
	Address     string `json:"address,omitempty"`
	Metadata    string `json:"metadata,omitempty"`
}

// AccountResponse is the response body for account operations.
type AccountResponse struct {
	ID          string      `json:"id"`
	WorkspaceID string      `json:"workspaceId"`
	Name        string      `json:"name"`
	Domain      *string     `json:"domain,omitempty"`
	Industry    *string     `json:"industry,omitempty"`
	SizeSegment *string     `json:"sizeSegment,omitempty"`
	OwnerID     string      `json:"ownerId"`
	Address     *string     `json:"address,omitempty"`
	Metadata    *string     `json:"metadata,omitempty"`
	CreatedAt   string      `json:"createdAt"`
	UpdatedAt   string      `json:"updatedAt"`
	DeletedAt   *string     `json:"deletedAt,omitempty"`
}

// ListAccountsResponse is the response body for listing accounts.
type ListAccountsResponse struct {
	Data []AccountResponse `json:"data"`
	Meta Meta             `json:"meta"`
}

// Meta contains pagination metadata.
type Meta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// CreateAccount handles POST /api/v1/accounts
// Task 1.3.7: Create a new account (CRUD + Multi-tenancy)
func (h *AccountHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID, err := getWorkspaceID(ctx)
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}

	var req CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate required fields
	if req.Name == "" || req.OwnerID == "" {
		writeError(w, http.StatusBadRequest, "name and ownerId are required")
		return
	}

	// Create account via service
	account, err := h.accountService.Create(ctx, crm.CreateAccountInput{
		WorkspaceID: wsID,
		Name:        req.Name,
		Domain:      req.Domain,
		Industry:    req.Industry,
		SizeSegment: req.SizeSegment,
		OwnerID:     req.OwnerID,
		Address:     req.Address,
		Metadata:    req.Metadata,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create account: %v", err))
		return
	}

	// Write response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(accountToResponse(account))
}

// GetAccount handles GET /api/v1/accounts/{id}
// Task 1.3.7: Retrieve a single account by ID (with multi-tenancy isolation)
func (h *AccountHandler) GetAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID, err := getWorkspaceID(ctx)
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}

	accountID := chi.URLParam(r, "id")
	if accountID == "" {
		writeError(w, http.StatusBadRequest, "account id is required")
		return
	}

	// Get account via service
	account, err := h.accountService.Get(ctx, wsID, accountID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get account: %v", err))
		return
	}

	// Write response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(accountToResponse(account))
}

// ListAccounts handles GET /api/v1/accounts with pagination
// Task 1.3.7: List accounts with pagination filters
func (h *AccountHandler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID, err := getWorkspaceID(ctx)
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}

	// Parse + validate pagination params (extracted to reduce cyclomatic complexity)
	page := parsePaginationParams(r)

	// List accounts via service
	accounts, total, err := h.accountService.List(ctx, wsID, crm.ListAccountsInput{
		Limit:  page.Limit,
		Offset: page.Offset,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list accounts: %v", err))
		return
	}

	// Build response
	responses := make([]AccountResponse, len(accounts))
	for i, acc := range accounts {
		responses[i] = accountToResponse(acc)
	}

	resp := ListAccountsResponse{
		Data: responses,
		Meta: Meta{Total: total, Limit: page.Limit, Offset: page.Offset},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// UpdateAccount handles PUT /api/v1/accounts/{id}
// Task 1.3.7: Update an account (partial update allowed)
// KNOWN_LIMITATION (TD-5): Get + Update are two separate SQL calls.
// Under concurrent requests, another writer could modify/delete the account
// between Get and Update. For MVP (SQLite single-writer) this is acceptable.
// Fix: use a DB transaction with SELECT FOR UPDATE when migrating to Postgres.
func (h *AccountHandler) UpdateAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID, err := getWorkspaceID(ctx)
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}

	accountID := chi.URLParam(r, "id")
	if accountID == "" {
		writeError(w, http.StatusBadRequest, "account id is required")
		return
	}

	// First fetch existing account to preserve unmodified fields
	existing, err := h.accountService.Get(ctx, wsID, accountID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get account: %v", err))
		return
	}

	var req UpdateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Merge request with existing values (extracted to reduce cyclomatic complexity)
	updateInput := buildUpdateInput(req, existing)

	// Update account via service
	updated, err := h.accountService.Update(ctx, wsID, accountID, updateInput)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update account: %v", err))
		return
	}

	// Write response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(accountToResponse(updated))
}

// DeleteAccount handles DELETE /api/v1/accounts/{id}
// Task 1.3.7: Soft delete an account (sets deleted_at timestamp)
// TD-3 fix: returns 404 if account does not exist or is already deleted
func (h *AccountHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	wsID, err := getWorkspaceID(ctx)
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}

	accountID := chi.URLParam(r, "id")
	if accountID == "" {
		writeError(w, http.StatusBadRequest, "account id is required")
		return
	}

	// Verify account exists (and is not already soft-deleted) before deleting (TD-3)
	_, err = h.accountService.Get(ctx, wsID, accountID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get account: %v", err))
		return
	}

	// Delete account via service (soft delete)
	if err := h.accountService.Delete(ctx, wsID, accountID); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete account: %v", err))
		return
	}

	// Write response (204 No Content)
	w.WriteHeader(http.StatusNoContent)
}

// --- helpers ---

// accountToResponse converts a domain Account to an AccountResponse.
func accountToResponse(acc *crm.Account) AccountResponse {
	return AccountResponse{
		ID:          acc.ID,
		WorkspaceID: acc.WorkspaceID,
		Name:        acc.Name,
		Domain:      acc.Domain,
		Industry:    acc.Industry,
		SizeSegment: acc.SizeSegment,
		OwnerID:     acc.OwnerID,
		Address:     acc.Address,
		Metadata:    acc.Metadata,
		CreatedAt:   acc.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   acc.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		DeletedAt:   formatDeletedAt(acc.DeletedAt),
	}
}

// formatDeletedAt formats deleted_at timestamp as string or nil.
func formatDeletedAt(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format("2006-01-02T15:04:05Z")
	return &s
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
