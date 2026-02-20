package handlers

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
)

type NoteHandler struct{ service *crm.NoteService }

func NewNoteHandler(service *crm.NoteService) *NoteHandler { return &NoteHandler{service: service} }

type CreateNoteRequest struct {
	EntityType string `json:"entityType"`
	EntityID   string `json:"entityId"`
	AuthorID   string `json:"authorId"`
	Content    string `json:"content"`
	IsInternal bool   `json:"isInternal"`
	Metadata   string `json:"metadata,omitempty"`
}

type UpdateNoteRequest struct {
	Content    string `json:"content"`
	IsInternal bool   `json:"isInternal"`
	Metadata   string `json:"metadata,omitempty"`
}

func (h *NoteHandler) CreateNote(w http.ResponseWriter, r *http.Request) {
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	var req CreateNoteRequest
	if !decodeBodyJSON(w, r, &req) {
		return
	}
	if !isNoteRequestValid(req) {
		writeError(w, http.StatusBadRequest, "entityType, entityId, authorId and content are required")
		return
	}
	out, svcErr := h.service.Create(r.Context(), crm.CreateNoteInput{
		WorkspaceID: wsID,
		EntityType:  req.EntityType,
		EntityID:    req.EntityID,
		AuthorID:    req.AuthorID,
		Content:     req.Content,
		IsInternal:  req.IsInternal,
		Metadata:    req.Metadata,
	})
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create note: %v", svcErr))
		return
	}
	w.WriteHeader(http.StatusCreated)
	if !writeJSONOr500(w, out) {
		return
	}
}

func (h *NoteHandler) GetNote(w http.ResponseWriter, r *http.Request) {
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, paramID)
	out, svcErr := h.service.Get(r.Context(), wsID, id)
	if handleGetError(w, svcErr, "note not found", "failed to get note: %v") {
		return
	}
	if !writeJSONOr500(w, out) {
		return
	}
}

func (h *NoteHandler) ListNotes(w http.ResponseWriter, r *http.Request) {
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	page := parsePaginationParams(r)
	items, total, svcErr := h.service.List(r.Context(), wsID, crm.ListNotesInput{Limit: page.Limit, Offset: page.Offset})
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list notes: %v", svcErr))
		return
	}
	if !writePaginatedOr500(w, items, total, page) {
		return
	}
}

func (h *NoteHandler) UpdateNote(w http.ResponseWriter, r *http.Request) {
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, paramID)
	_, svcErr := h.service.Get(r.Context(), wsID, id)
	if handleGetError(w, svcErr, "note not found", "failed to get note: %v") {
		return
	}
	var req UpdateNoteRequest
	if !decodeBodyJSON(w, r, &req) {
		return
	}
	out, svcErr := h.service.Update(r.Context(), wsID, id, crm.UpdateNoteInput{Content: req.Content, IsInternal: req.IsInternal, Metadata: req.Metadata})
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update note: %v", svcErr))
		return
	}
	if !writeJSONOr500(w, out) {
		return
	}
}

func (h *NoteHandler) DeleteNote(w http.ResponseWriter, r *http.Request) {
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, paramID)
	if svcErr := h.service.Delete(r.Context(), wsID, id); svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete note: %v", svcErr))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// isNoteRequestValid checks required fields for CreateNote.
// Task 1.6.15: Extracted to reduce cyclomatic complexity of CreateNote (was 8).
func isNoteRequestValid(req CreateNoteRequest) bool {
	return req.EntityType != "" && req.EntityID != "" && req.AuthorID != "" && req.Content != ""
}
