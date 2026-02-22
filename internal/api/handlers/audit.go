// Task 4.6 — FR-070/071: Audit Advanced (query + export)
package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	domainaudit "github.com/matiasleandrokruk/fenix/internal/domain/audit"
)

// AuditHandler serves audit query and export endpoints.
// Task 4.6 — wraps AuditService for query/get/export HTTP APIs.
type AuditHandler struct {
	auditService *domainaudit.AuditService
}

const (
	errAuditEventIDRequired = "audit event id is required"
	errFailedToQueryAudit   = "failed to query audit events: %v"
	errFailedToGetAudit     = "failed to get audit event: %v"
	errFormatMustBeCSV      = "format must be csv"
	queryParamFormat        = "format"
	formatCSV               = "csv"
)

func NewAuditHandler(auditService *domainaudit.AuditService) *AuditHandler {
	return &AuditHandler{auditService: auditService}
}

// Query handles GET /api/v1/audit/events.
// Task 4.6 — FR-070 advanced filters with pagination.
func (h *AuditHandler) Query(w http.ResponseWriter, r *http.Request) {
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	page := parsePaginationParams(r)

	items, err := h.auditService.Query(r.Context(), domainaudit.QueryInput{
		WorkspaceID: wsID,
		ActorID:     r.URL.Query().Get("actor_id"),
		EntityType:  r.URL.Query().Get(paramEntityType),
		Action:      r.URL.Query().Get("action"),
		Outcome:     r.URL.Query().Get("outcome"),
		DateFrom:    r.URL.Query().Get("date_from"),
		DateTo:      r.URL.Query().Get("date_to"),
		Limit:       page.Limit,
		Offset:      page.Offset,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf(errFailedToQueryAudit, err))
		return
	}

	_ = writePaginatedOr500(w, items, len(items), page)
}

// GetByID handles GET /api/v1/audit/events/{id}.
// Task 4.6 — FR-070 retrieve single audit event.
func (h *AuditHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, paramID)
	if id == "" {
		writeError(w, http.StatusBadRequest, errAuditEventIDRequired)
		return
	}

	event, err := h.auditService.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, errAuditEventNotFound)
			return
		}
		writeError(w, http.StatusInternalServerError, fmt.Sprintf(errFailedToGetAudit, err))
		return
	}
	if event.WorkspaceID != wsID {
		writeError(w, http.StatusNotFound, errAuditEventNotFound)
		return
	}

	_ = writeJSONOr500(w, event)
}

// Export handles POST /api/v1/audit/export.
// Task 4.6 — FR-071 CSV export.
func (h *AuditHandler) Export(w http.ResponseWriter, r *http.Request) {
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	if r.URL.Query().Get(queryParamFormat) != formatCSV {
		writeError(w, http.StatusBadRequest, errFormatMustBeCSV)
		return
	}

	reader, err := h.auditService.Export(r.Context(), domainaudit.ExportInput{
		WorkspaceID: wsID,
		ActorID:     r.URL.Query().Get("actor_id"),
		EntityType:  r.URL.Query().Get(paramEntityType),
		Action:      r.URL.Query().Get("action"),
		Outcome:     r.URL.Query().Get("outcome"),
		DateFrom:    r.URL.Query().Get("date_from"),
		DateTo:      r.URL.Query().Get("date_to"),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, errFailedToExportAudit)
		return
	}

	w.Header().Set(headerContentType, mimeCSV)
	w.Header().Set(headerContentDisposition, `attachment; filename="audit_events.csv"`)
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, reader)
}
