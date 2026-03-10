package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	signaldomain "github.com/matiasleandrokruk/fenix/internal/domain/signal"
)

type SignalService interface {
	List(ctx context.Context, workspaceID string, filters signaldomain.Filters) ([]*signaldomain.Signal, error)
	GetByEntity(ctx context.Context, workspaceID, entityType, entityID string) ([]*signaldomain.Signal, error)
	Dismiss(ctx context.Context, workspaceID, signalID, actorID string) error
}

type SignalHandler struct {
	service SignalService
	authz   ActionAuthorizer
}

type SignalResponse struct {
	ID          string                 `json:"id"`
	WorkspaceID string                 `json:"workspace_id"`
	EntityType  string                 `json:"entity_type"`
	EntityID    string                 `json:"entity_id"`
	SignalType  string                 `json:"signal_type"`
	Confidence  float64                `json:"confidence"`
	EvidenceIDs []string               `json:"evidence_ids"`
	SourceType  string                 `json:"source_type"`
	SourceID    string                 `json:"source_id"`
	Metadata    map[string]any         `json:"metadata"`
	Status      string                 `json:"status"`
	DismissedBy *string                `json:"dismissed_by,omitempty"`
	DismissedAt *string                `json:"dismissed_at,omitempty"`
	ExpiresAt   *string                `json:"expires_at,omitempty"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
}

func NewSignalHandler(service SignalService) *SignalHandler {
	return &SignalHandler{service: service}
}

func NewSignalHandlerWithAuthorizer(service SignalService, authz ActionAuthorizer) *SignalHandler {
	return &SignalHandler{service: service, authz: authz}
}

func (h *SignalHandler) List(w http.ResponseWriter, r *http.Request) {
	if !checkActionAuthorization(w, r, h.authz, resourceAPI, "signals.list") {
		return
	}

	workspaceID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}

	filters, err := decodeSignalFilters(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	out, err := fetchSignals(r.Context(), h.service, workspaceID, filters)
	if err != nil {
		writeSignalError(w, err)
		return
	}

	response := make([]*SignalResponse, 0, len(out))
	for _, item := range out {
		response = append(response, signalToResponse(item))
	}
	_ = writeJSONOr500(w, map[string]any{"data": response})
}

func (h *SignalHandler) Dismiss(w http.ResponseWriter, r *http.Request) {
	if !checkActionAuthorization(w, r, h.authz, resourceAPI, "signals.dismiss") {
		return
	}

	workspaceID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}

	id := chi.URLParam(r, paramID)
	if id == "" {
		writeError(w, http.StatusBadRequest, "signal id is required")
		return
	}

	actorID, _ := r.Context().Value(ctxkeys.UserID).(string)
	if actorID == "" {
		writeError(w, http.StatusBadRequest, "missing user_id in context")
		return
	}

	if err := h.service.Dismiss(r.Context(), workspaceID, id, actorID); err != nil {
		writeSignalError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func fetchSignals(ctx context.Context, svc SignalService, workspaceID string, filters signaldomain.Filters) ([]*signaldomain.Signal, error) {
	if filters.EntityType != "" && filters.EntityID != "" {
		return svc.GetByEntity(ctx, workspaceID, filters.EntityType, filters.EntityID)
	}
	return svc.List(ctx, workspaceID, filters)
}

func decodeSignalFilters(r *http.Request) (signaldomain.Filters, error) {
	var filters signaldomain.Filters
	if entityType := r.URL.Query().Get("entity_type"); entityType != "" {
		filters.EntityType = entityType
	}
	if entityID := r.URL.Query().Get("entity_id"); entityID != "" {
		filters.EntityID = entityID
	}
	if status := r.URL.Query().Get("status"); status != "" {
		parsed := signaldomain.Status(status)
		switch parsed {
		case signaldomain.StatusActive, signaldomain.StatusDismissed, signaldomain.StatusExpired:
			filters.Status = &parsed
		default:
			return filters, errors.New("invalid signal status")
		}
	}
	if (filters.EntityType == "") != (filters.EntityID == "") {
		return filters, errors.New("entity_type and entity_id must be provided together")
	}
	return filters, nil
}

func writeSignalError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, signaldomain.ErrSignalNotFound):
		writeError(w, http.StatusNotFound, "signal not found")
	case errors.Is(err, signaldomain.ErrInvalidSignalInput):
		writeError(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, signaldomain.ErrSignalDismissInvalid):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func signalToResponse(in *signaldomain.Signal) *SignalResponse {
	if in == nil {
		return nil
	}

	return &SignalResponse{
		ID:          in.ID,
		WorkspaceID: in.WorkspaceID,
		EntityType:  in.EntityType,
		EntityID:    in.EntityID,
		SignalType:  in.SignalType,
		Confidence:  in.Confidence,
		EvidenceIDs: in.EvidenceIDs,
		SourceType:  in.SourceType,
		SourceID:    in.SourceID,
		Metadata:    decodeSignalMetadata(in.Metadata),
		Status:      string(in.Status),
		DismissedBy: in.DismissedBy,
		DismissedAt: formatOptionalSignalTime(in.DismissedAt),
		ExpiresAt:   formatOptionalSignalTime(in.ExpiresAt),
		CreatedAt:   in.CreatedAt.Format(timeFormatISO),
		UpdatedAt:   in.UpdatedAt.Format(timeFormatISO),
	}
}

func decodeSignalMetadata(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func formatOptionalSignalTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format(timeFormatISO)
	return &formatted
}
