// Task 1.3.7 / TD-1 fix: Handler helper functions and context management
package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
)

// paginationParams holds parsed limit and offset values.
type paginationParams struct {
	Limit  int
	Offset int
}

const (
	defaultPaginationLimit = 25
	maxPaginationLimit     = 100
)

// HTTP response string constants (extracted to satisfy goconst lint gate).
const (
	headerContentType = "Content-Type"
	mimeJSON          = "application/json"
	timeFormatISO     = "2006-01-02T15:04:05Z"

	// Error messages — workspace / auth
	errMissingWorkspaceID      = "missing workspace_id in context"
	errMissingWorkspaceContext = "missing workspace context"
	errMissingWorkspaceShort   = "missing workspace_id"

	// Error messages — request
	errInvalidBody = "invalid request body"

	// Error messages — encode
	errFailedToEncode     = "failed to encode response"
	errFailedToEncodeJSON = `{"error":"failed to encode response"}`
	errEmptyJSON          = "{}"

	// Error messages — account
	errAccountIDRequired  = "account id is required"
	errAccountNotFound    = "account not found"
	errFailedToGetAccount = "failed to get account: %v"

	// Error messages — contact
	errContactIDRequired  = "contact id is required"
	errContactNotFound    = "contact not found"
	errFailedToGetContact = "failed to get contact: %v"

	// Error messages — lead
	errLeadIDRequired  = "lead id is required"
	errLeadNotFound    = "lead not found"
	errFailedToGetLead = "failed to get lead: %v"

	// Error messages — case
	errCaseNotFound = "case not found"

	// Error messages — agent
	errAgentRunNotFound = "agent run not found"

	// URL param names
	paramID         = "id"
	paramStageID    = "stage_id"
	paramEntityID   = "entity_id"
	paramEntityType = "entity_type"
)

// getWorkspaceID retrieves workspace_id from context.
// Uses ctxkeys.WorkspaceID — same type+value as WorkspaceMiddleware injection.
// This eliminates the silent type mismatch between different context key types (TD-1).
func getWorkspaceID(ctx context.Context) (string, error) {
	wsID, ok := ctx.Value(ctxkeys.WorkspaceID).(string)
	if !ok || wsID == "" {
		return "", fmt.Errorf("workspace_id not found in context")
	}
	return wsID, nil
}

// parsePaginationParams extracts and validates limit/offset from URL query params.
// Extracted to reduce cyclomatic complexity of ListAccounts (was 11, now isolated here).
func parsePaginationParams(r *http.Request) paginationParams {
	limit := defaultPaginationLimit
	offset := 0

	if lim, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && lim > 0 {
		if lim > maxPaginationLimit {
			lim = maxPaginationLimit
		}
		limit = lim
	}

	if off, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && off >= 0 {
		offset = off
	}

	return paginationParams{Limit: limit, Offset: offset}
}

// coalesce returns val if non-empty, otherwise returns fallback.
// Task 1.6.15: Used across Update handlers to replace repetitive if-empty-use-existing branches.
func coalesce(val, fallback string) string {
	if val == "" {
		return fallback
	}
	return val
}

// coalescePtr returns val if non-empty, otherwise dereferences the pointer fallback if non-nil.
// Task 1.6.15: Used for nullable fields (e.g. email *string) in Update handlers.
func coalescePtr(val string, fallback *string) string {
	if val == "" && fallback != nil {
		return *fallback
	}
	return val
}

// buildUpdateInput merges the update request with existing values.
// Extracted to reduce cyclomatic complexity of UpdateAccount (was 9, now isolated here).
// Required fields (Name, OwnerID) default to existing values if omitted.
func buildUpdateInput(req UpdateAccountRequest, existing *crm.Account) crm.UpdateAccountInput {
	input := crm.UpdateAccountInput{
		Name:        req.Name,
		Domain:      req.Domain,
		Industry:    req.Industry,
		SizeSegment: req.SizeSegment,
		OwnerID:     req.OwnerID,
		Address:     req.Address,
		Metadata:    req.Metadata,
	}
	if input.Name == "" {
		input.Name = existing.Name
	}
	if input.OwnerID == "" {
		input.OwnerID = existing.OwnerID
	}
	return input
}
