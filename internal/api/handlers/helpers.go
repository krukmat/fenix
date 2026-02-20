// Task 1.3.7 / TD-1 fix: Handler helper functions and context management
package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
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

// requireWorkspaceID obtiene workspace_id desde contexto y responde 400 cuando falta.
func requireWorkspaceID(w http.ResponseWriter, r *http.Request) (string, bool) {
	wsID, wsErr := getWorkspaceID(r.Context())
	if wsErr != nil {
		writeError(w, http.StatusBadRequest, errMissingWorkspaceID)
		return "", false
	}
	return wsID, true
}

// decodeBodyJSON decodifica body JSON y responde 400 si es inválido.
func decodeBodyJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if decodeErr := json.NewDecoder(r.Body).Decode(dst); decodeErr != nil {
		writeError(w, http.StatusBadRequest, errInvalidBody)
		return false
	}
	return true
}

// writeJSONOr500 escribe payload JSON y responde 500 en caso de fallo.
func writeJSONOr500(w http.ResponseWriter, payload any) bool {
	if encodeErr := json.NewEncoder(w).Encode(payload); encodeErr != nil {
		writeError(w, http.StatusInternalServerError, errFailedToEncode)
		return false
	}
	return true
}

// writePaginatedOr500 escribe respuesta estándar paginada {data, meta}.
func writePaginatedOr500(w http.ResponseWriter, items any, total int, page paginationParams) bool {
	return writeJSONOr500(w, map[string]any{
		"data": items,
		"meta": Meta{Total: total, Limit: page.Limit, Offset: page.Offset},
	})
}

// handleGetError unifica manejo ErrNoRows + error interno para endpoints Get.
func handleGetError(w http.ResponseWriter, err error, notFoundMsg, internalFmt string) bool {
	if errorsIsNoRows(err) {
		writeError(w, http.StatusNotFound, notFoundMsg)
		return true
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf(internalFmt, err))
		return true
	}
	return false
}

func errorsIsNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

// getEntityForUpdate centraliza patrón: leer id path param, cargar entidad, mapear errores.
func getEntityForUpdate[T any](
	w http.ResponseWriter,
	r *http.Request,
	wsID string,
	idRequiredMsg string,
	notFoundMsg string,
	internalFmt string,
	getter func(context.Context, string, string) (*T, error),
) (string, *T, bool) {
	ctx := r.Context()
	id := chiURLParamID(r)
	if id == "" {
		writeError(w, http.StatusBadRequest, idRequiredMsg)
		return "", nil, false
	}

	existing, err := getter(ctx, wsID, id)
	if errorsIsNoRows(err) {
		writeError(w, http.StatusNotFound, notFoundMsg)
		return "", nil, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf(internalFmt, err))
		return "", nil, false
	}

	return id, existing, true
}

// ensureEntityExistsBeforeDelete centraliza verificación previa de existencia para DELETE.
func ensureEntityExistsBeforeDelete[T any](
	w http.ResponseWriter,
	r *http.Request,
	wsID string,
	idRequiredMsg string,
	notFoundMsg string,
	internalFmt string,
	getter func(context.Context, string, string) (*T, error),
) (string, bool) {
	ctx := r.Context()
	id := chiURLParamID(r)
	if id == "" {
		writeError(w, http.StatusBadRequest, idRequiredMsg)
		return "", false
	}

	_, err := getter(ctx, wsID, id)
	if errorsIsNoRows(err) {
		writeError(w, http.StatusNotFound, notFoundMsg)
		return "", false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf(internalFmt, err))
		return "", false
	}

	return id, true
}

func chiURLParamID(r *http.Request) string {
	return chi.URLParam(r, paramID)
}

// handleListWithPagination centraliza listados estándar con meta paginada.
func handleListWithPagination[T any](
	w http.ResponseWriter,
	r *http.Request,
	errFmt string,
	listFn func(context.Context, string, int, int) ([]*T, int, error),
) {
	wsID, wsErr := getWorkspaceID(r.Context())
	if wsErr != nil {
		writeError(w, http.StatusBadRequest, errMissingWorkspaceID)
		return
	}

	page := parsePaginationParams(r)
	items, total, err := listFn(r.Context(), wsID, page.Limit, page.Offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf(errFmt, err))
		return
	}

	_ = writePaginatedOr500(w, items, total, page)
}

// handleEntityUpdate centraliza el flujo común de UPDATE por entidad:
// workspace + carga previa + decode + update + respuesta JSON.
func handleEntityUpdate[Entity any, Req any, In any, Out any](
	w http.ResponseWriter,
	r *http.Request,
	notFoundMsg string,
	getErrFmt string,
	updateErrFmt string,
	getter func(context.Context, string, string) (*Entity, error),
	buildInput func(Req, *Entity) In,
	updater func(context.Context, string, string, In) (*Out, error),
) {
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}

	id := chiURLParamID(r)
	existing, svcErr := getter(r.Context(), wsID, id)
	if handleGetError(w, svcErr, notFoundMsg, getErrFmt) {
		return
	}

	var req Req
	if !decodeBodyJSON(w, r, &req) {
		return
	}

	out, upErr := updater(r.Context(), wsID, id, buildInput(req, existing))
	if upErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf(updateErrFmt, upErr))
		return
	}

	_ = writeJSONOr500(w, out)
}
