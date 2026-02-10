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
	wsID, err := getWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}
	var req CreateNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if !isNoteRequestValid(req) {
		writeError(w, http.StatusBadRequest, "entityType, entityId, authorId and content are required")
		return
	}
	out, err := h.service.Create(r.Context(), crm.CreateNoteInput{
		WorkspaceID: wsID,
		EntityType:  req.EntityType,
		EntityID:    req.EntityID,
		AuthorID:    req.AuthorID,
		Content:     req.Content,
		IsInternal:  req.IsInternal,
		Metadata:    req.Metadata,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create note: %v", err))
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(out)
}

func (h *NoteHandler) GetNote(w http.ResponseWriter, r *http.Request) {
	wsID, err := getWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}
	id := chi.URLParam(r, "id")
	out, err := h.service.Get(r.Context(), wsID, id)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "note not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get note: %v", err))
		return
	}
	json.NewEncoder(w).Encode(out)
}

func (h *NoteHandler) ListNotes(w http.ResponseWriter, r *http.Request) {
	wsID, err := getWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}
	page := parsePaginationParams(r)
	items, total, err := h.service.List(r.Context(), wsID, crm.ListNotesInput{Limit: page.Limit, Offset: page.Offset})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list notes: %v", err))
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"data": items, "meta": Meta{Total: total, Limit: page.Limit, Offset: page.Offset}})
}

func (h *NoteHandler) UpdateNote(w http.ResponseWriter, r *http.Request) {
	wsID, err := getWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}
	id := chi.URLParam(r, "id")
	if _, err := h.service.Get(r.Context(), wsID, id); errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "note not found")
		return
	}
	var req UpdateNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	out, err := h.service.Update(r.Context(), wsID, id, crm.UpdateNoteInput{Content: req.Content, IsInternal: req.IsInternal, Metadata: req.Metadata})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update note: %v", err))
		return
	}
	json.NewEncoder(w).Encode(out)
}

func (h *NoteHandler) DeleteNote(w http.ResponseWriter, r *http.Request) {
	wsID, err := getWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.service.Delete(r.Context(), wsID, id); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete note: %v", err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// isNoteRequestValid checks required fields for CreateNote.
// Task 1.6.15: Extracted to reduce cyclomatic complexity of CreateNote (was 8).
func isNoteRequestValid(req CreateNoteRequest) bool {
	return req.EntityType != "" && req.EntityID != "" && req.AuthorID != "" && req.Content != ""
}
