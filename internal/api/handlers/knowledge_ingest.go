// Task 2.2: HTTP handler for knowledge ingestion.
// POST /api/v1/knowledge/ingest — creates a knowledge_item + embedding_document chunks.
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
)

// KnowledgeIngestHandler handles knowledge ingestion HTTP requests (Task 2.2).
type KnowledgeIngestHandler struct {
	ingestService *knowledge.IngestService
}

// NewKnowledgeIngestHandler creates a KnowledgeIngestHandler.
func NewKnowledgeIngestHandler(svc *knowledge.IngestService) *KnowledgeIngestHandler {
	return &KnowledgeIngestHandler{ingestService: svc}
}

// ingestRequest is the JSON request body for POST /api/v1/knowledge/ingest.
type ingestRequest struct {
	SourceSystem      *string `json:"sourceSystem,omitempty"`
	SourceType        string  `json:"sourceType"`
	SourceObjectID    *string `json:"sourceObjectId,omitempty"`
	RefreshStrategy   *string `json:"refreshStrategy,omitempty"`
	DeleteBehavior    *string `json:"deleteBehavior,omitempty"`
	PermissionContext *string `json:"permissionContext,omitempty"`
	Title             string  `json:"title"`
	RawContent        string  `json:"rawContent"`
	EntityType        *string `json:"entityType,omitempty"`
	EntityID          *string `json:"entityId,omitempty"`
	Metadata          *string `json:"metadata,omitempty"`
}

// ingestResponse is the JSON response body for a successful ingest.
type ingestResponse struct {
	ID                string  `json:"id"`
	WorkspaceID       string  `json:"workspaceId"`
	SourceSystem      *string `json:"sourceSystem,omitempty"`
	SourceType        string  `json:"sourceType"`
	SourceObjectID    *string `json:"sourceObjectId,omitempty"`
	RefreshStrategy   *string `json:"refreshStrategy,omitempty"`
	DeleteBehavior    *string `json:"deleteBehavior,omitempty"`
	PermissionContext *string `json:"permissionContext,omitempty"`
	Title             string  `json:"title"`
	EntityType        *string `json:"entityType,omitempty"`
	EntityID          *string `json:"entityId,omitempty"`
	CreatedAt         string  `json:"createdAt"`
}

// Ingest handles POST /api/v1/knowledge/ingest.
func (h *KnowledgeIngestHandler) Ingest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	wsID, wsErr := getWorkspaceID(ctx)
	if wsErr != nil {
		writeError(w, http.StatusUnauthorized, errMissingWorkspaceContext)
		return
	}

	var req ingestRequest
	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		writeError(w, http.StatusBadRequest, errInvalidBody)
		return
	}

	if valErr := validateIngestRequest(req); valErr != nil {
		writeError(w, http.StatusBadRequest, valErr.Error())
		return
	}

	input := knowledge.CreateKnowledgeItemInput{
		WorkspaceID:       wsID,
		SourceSystem:      req.SourceSystem,
		SourceType:        knowledge.SourceType(req.SourceType),
		SourceObjectID:    req.SourceObjectID,
		RefreshStrategy:   req.RefreshStrategy,
		DeleteBehavior:    req.DeleteBehavior,
		PermissionContext: req.PermissionContext,
		Title:             req.Title,
		RawContent:        req.RawContent,
		EntityType:        req.EntityType,
		EntityID:          req.EntityID,
		Metadata:          req.Metadata,
	}

	item, ingestErr := h.ingestService.Ingest(ctx, input)
	if ingestErr != nil {
		writeError(w, http.StatusInternalServerError, "failed to ingest knowledge item")
		return
	}

	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(http.StatusCreated)
	if encodeErr := json.NewEncoder(w).Encode(ingestResponse{
		ID:                item.ID,
		WorkspaceID:       item.WorkspaceID,
		SourceSystem:      item.SourceSystem,
		SourceType:        string(item.SourceType),
		SourceObjectID:    item.SourceObjectID,
		RefreshStrategy:   item.RefreshStrategy,
		DeleteBehavior:    item.DeleteBehavior,
		PermissionContext: item.PermissionContext,
		Title:             item.Title,
		EntityType:        item.EntityType,
		EntityID:          item.EntityID,
		CreatedAt:         item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}); encodeErr != nil {
		http.Error(w, errFailedToEncodeJSON, http.StatusInternalServerError)
	}
}

// validateIngestRequest checks that the required fields are present and valid.
func validateIngestRequest(req ingestRequest) error {
	if req.Title == "" {
		return errorf("title is required")
	}
	if req.SourceType == "" {
		return errorf("sourceType is required")
	}
	if hasValue(req.SourceObjectID) && !hasValue(req.SourceSystem) {
		return errorf("sourceSystem is required when sourceObjectId is provided")
	}
	if !isValidSourceType(req.SourceType) {
		return errorf("invalid sourceType: must be one of document, email, call, note, case, ticket, kb_article, api, other")
	}
	return nil
}

func hasValue(s *string) bool {
	return s != nil && *s != ""
}

// isValidSourceType returns true if s is a recognised SourceType value.
func isValidSourceType(s string) bool {
	switch knowledge.SourceType(s) {
	case knowledge.SourceTypeDocument,
		knowledge.SourceTypeEmail,
		knowledge.SourceTypeCall,
		knowledge.SourceTypeNote,
		knowledge.SourceTypeCase,
		knowledge.SourceTypeTicket,
		knowledge.SourceTypeKBArticle,
		knowledge.SourceTypeAPI,
		knowledge.SourceTypeOther:
		return true
	}
	return false
}

// errorf returns a simple error value with a formatted message.
func errorf(msg string) error {
	return &ingestValidationError{msg: msg}
}

type ingestValidationError struct{ msg string }

func (e *ingestValidationError) Error() string { return e.msg }
