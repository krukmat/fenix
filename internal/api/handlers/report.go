// Task 4.5e — FR-003: Reporting HTTP handlers.
package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
)

const (
	mimeCSV = "text/csv"
)

// ReportHandler serves reporting endpoints.
// Task 4.5e — wraps ReportService for JSON + CSV HTTP APIs.
type ReportHandler struct {
	reportService *crm.ReportService
}

func NewReportHandler(reportService *crm.ReportService) *ReportHandler {
	return &ReportHandler{reportService: reportService}
}

func (h *ReportHandler) GetSalesFunnel(w http.ResponseWriter, r *http.Request) {
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}

	report, err := h.reportService.GetSalesFunnel(r.Context(), wsID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to build sales funnel report: %v", err))
		return
	}
	_ = writeJSONOr500(w, report)
}

func (h *ReportHandler) GetDealAging(w http.ResponseWriter, r *http.Request) {
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	rows, err := h.reportService.GetDealAging(r.Context(), wsID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to build deal aging report: %v", err))
		return
	}
	_ = writeJSONOr500(w, rows)
}

func (h *ReportHandler) GetSupportBacklog(w http.ResponseWriter, r *http.Request) {
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	agingDays, ok := parseAgingDays(w, r)
	if !ok {
		return
	}

	report, err := h.reportService.GetSupportBacklog(r.Context(), wsID, agingDays)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to build support backlog report: %v", err))
		return
	}
	_ = writeJSONOr500(w, report)
}

func (h *ReportHandler) GetSupportVolume(w http.ResponseWriter, r *http.Request) {
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}

	rows, err := h.reportService.GetCaseVolume(r.Context(), wsID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to build support volume report: %v", err))
		return
	}
	_ = writeJSONOr500(w, rows)
}

func (h *ReportHandler) ExportSalesFunnelCSV(w http.ResponseWriter, r *http.Request) {
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	exportCSVReport(w, r, wsID, "sales_funnel.csv", "failed to export sales funnel csv",
		func() (io.Reader, error) { return h.reportService.ExportSalesFunnelCSV(r.Context(), wsID) })
}

func (h *ReportHandler) ExportSupportBacklogCSV(w http.ResponseWriter, r *http.Request) {
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	exportCSVReport(w, r, wsID, "support_backlog.csv", "failed to export support backlog csv",
		func() (io.Reader, error) { return h.reportService.ExportSupportBacklogCSV(r.Context(), wsID) })
}

func exportCSVReport(w http.ResponseWriter, r *http.Request, _ string, filename, errMsg string, load func() (io.Reader, error)) {
	if r.URL.Query().Get("format") != "csv" {
		writeError(w, http.StatusBadRequest, "format must be csv")
		return
	}
	reader, err := load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("%s: %v", errMsg, err))
		return
	}
	w.Header().Set(headerContentType, mimeCSV)
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, reader)
}

func parseAgingDays(w http.ResponseWriter, r *http.Request) (int, bool) {
	const defaultAgingDays = 30
	v := r.URL.Query().Get("aging_days")
	if v == "" {
		return defaultAgingDays, true
	}
	agingDays, err := strconv.Atoi(v)
	if err != nil {
		writeError(w, http.StatusBadRequest, "aging_days must be an integer")
		return 0, false
	}
	return agingDays, true
}
