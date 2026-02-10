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

type AttachmentHandler struct{ service *crm.AttachmentService }

func NewAttachmentHandler(service *crm.AttachmentService) *AttachmentHandler {
	return &AttachmentHandler{service: service}
}

type CreateAttachmentRequest struct {
	EntityType  string `json:"entityType"`
	EntityID    string `json:"entityId"`
	UploaderID  string `json:"uploaderId"`
	Filename    string `json:"filename"`
	ContentType string `json:"contentType,omitempty"`
	SizeBytes   *int64 `json:"sizeBytes,omitempty"`
	StoragePath string `json:"storagePath"`
	Sensitivity string `json:"sensitivity,omitempty"`
	Metadata    string `json:"metadata,omitempty"`
}

func (h *AttachmentHandler) CreateAttachment(w http.ResponseWriter, r *http.Request) {
	wsID, err := getWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}
	var req CreateAttachmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if !isAttachmentRequestValid(req) {
		writeError(w, http.StatusBadRequest, "entityType, entityId, uploaderId, filename and storagePath are required")
		return
	}
	out, err := h.service.Create(r.Context(), crm.CreateAttachmentInput{
		WorkspaceID: wsID,
		EntityType:  req.EntityType,
		EntityID:    req.EntityID,
		UploaderID:  req.UploaderID,
		Filename:    req.Filename,
		ContentType: req.ContentType,
		SizeBytes:   req.SizeBytes,
		StoragePath: req.StoragePath,
		Sensitivity: req.Sensitivity,
		Metadata:    req.Metadata,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create attachment: %v", err))
		return
	}
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(out); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
}

func (h *AttachmentHandler) GetAttachment(w http.ResponseWriter, r *http.Request) {
	wsID, err := getWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}
	id := chi.URLParam(r, "id")
	out, err := h.service.Get(r.Context(), wsID, id)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "attachment not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get attachment: %v", err))
		return
	}
	if err := json.NewEncoder(w).Encode(out); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
}

func (h *AttachmentHandler) ListAttachments(w http.ResponseWriter, r *http.Request) {
	wsID, err := getWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}
	page := parsePaginationParams(r)
	items, total, err := h.service.List(r.Context(), wsID, crm.ListAttachmentsInput{Limit: page.Limit, Offset: page.Offset})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list attachments: %v", err))
		return
	}
	if err := json.NewEncoder(w).Encode(map[string]any{"data": items, "meta": Meta{Total: total, Limit: page.Limit, Offset: page.Offset}}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
}

func (h *AttachmentHandler) DeleteAttachment(w http.ResponseWriter, r *http.Request) {
	wsID, err := getWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.service.Delete(r.Context(), wsID, id); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete attachment: %v", err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// isAttachmentRequestValid checks required fields for CreateAttachment.
// Task 1.6.15: Extracted to reduce cyclomatic complexity of CreateAttachment (was 9).
func isAttachmentRequestValid(req CreateAttachmentRequest) bool {
	return req.EntityType != "" && req.EntityID != "" && req.UploaderID != "" &&
		req.Filename != "" && req.StoragePath != ""
}
