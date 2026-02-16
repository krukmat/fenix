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
	wsID, wsErr := getWorkspaceID(r.Context())
	if wsErr != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}
	var req CreateAttachmentRequest
	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if !isAttachmentRequestValid(req) {
		writeError(w, http.StatusBadRequest, "entityType, entityId, uploaderId, filename and storagePath are required")
		return
	}
	out, svcErr := h.service.Create(r.Context(), crm.CreateAttachmentInput{
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
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create attachment: %v", svcErr))
		return
	}
	w.WriteHeader(http.StatusCreated)
	if encodeErr := json.NewEncoder(w).Encode(out); encodeErr != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
}

func (h *AttachmentHandler) GetAttachment(w http.ResponseWriter, r *http.Request) {
	wsID, wsErr := getWorkspaceID(r.Context())
	if wsErr != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}
	id := chi.URLParam(r, "id")
	out, svcErr := h.service.Get(r.Context(), wsID, id)
	if errors.Is(svcErr, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "attachment not found")
		return
	}
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get attachment: %v", svcErr))
		return
	}
	if encodeErr := json.NewEncoder(w).Encode(out); encodeErr != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
}

func (h *AttachmentHandler) ListAttachments(w http.ResponseWriter, r *http.Request) {
	wsID, wsErr := getWorkspaceID(r.Context())
	if wsErr != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}
	page := parsePaginationParams(r)
	items, total, svcErr := h.service.List(r.Context(), wsID, crm.ListAttachmentsInput{Limit: page.Limit, Offset: page.Offset})
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list attachments: %v", svcErr))
		return
	}
	if encodeErr := json.NewEncoder(w).Encode(map[string]any{"data": items, "meta": Meta{Total: total, Limit: page.Limit, Offset: page.Offset}}); encodeErr != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
}

func (h *AttachmentHandler) DeleteAttachment(w http.ResponseWriter, r *http.Request) {
	wsID, wsErr := getWorkspaceID(r.Context())
	if wsErr != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}
	id := chi.URLParam(r, "id")
	if svcErr := h.service.Delete(r.Context(), wsID, id); svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete attachment: %v", svcErr))
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
