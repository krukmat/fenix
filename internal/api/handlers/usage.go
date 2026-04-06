package handlers

import (
	"errors"
	"net/http"
	"time"

	usagedomain "github.com/matiasleandrokruk/fenix/internal/domain/usage"
)

const (
	errQuotaPolicyIDRequired = "quota_policy_id is required"
	errInvalidPeriodStart    = "invalid period_start, expected RFC3339"
	errInvalidPeriodEnd      = "invalid period_end, expected RFC3339"
	errFailedToListUsage     = "failed to list usage events"
	errFailedToGetQuotaState = "failed to get quota state"
	queryRunID               = "run_id"
	queryQuotaPolicyID       = "quota_policy_id"
	queryPeriodStart         = "period_start"
	queryPeriodEnd           = "period_end"
)

type UsageHandler struct {
	service *usagedomain.Service
}

func NewUsageHandler(service *usagedomain.Service) *UsageHandler {
	return &UsageHandler{service: service}
}

type usageEventResponse struct {
	ID            string  `json:"id"`
	WorkspaceID   string  `json:"workspaceId"`
	ActorID       string  `json:"actorId"`
	ActorType     string  `json:"actorType"`
	RunID         *string `json:"runId,omitempty"`
	ToolName      *string `json:"toolName,omitempty"`
	ModelName     *string `json:"modelName,omitempty"`
	InputUnits    int64   `json:"inputUnits"`
	OutputUnits   int64   `json:"outputUnits"`
	EstimatedCost float64 `json:"estimatedCost"`
	LatencyMs     *int64  `json:"latencyMs,omitempty"`
	CreatedAt     string  `json:"createdAt"`
}

type quotaStateResponse struct {
	ID            string  `json:"id"`
	WorkspaceID   string  `json:"workspaceId"`
	QuotaPolicyID string  `json:"quotaPolicyId"`
	CurrentValue  float64 `json:"currentValue"`
	PeriodStart   string  `json:"periodStart"`
	PeriodEnd     string  `json:"periodEnd"`
	LastEventAt   *string `json:"lastEventAt,omitempty"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`
}

func (h *UsageHandler) ListUsage(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}

	page := parsePaginationParams(r)
	runID := optionalTrimmedQuery(r, queryRunID)
	items, err := h.service.ListEvents(r.Context(), workspaceID, runID, page.Limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errFailedToListUsage)
		return
	}

	out := make([]usageEventResponse, 0, len(items))
	for _, item := range items {
		out = append(out, usageEventToResponse(item))
	}
	_ = writePaginatedOr500(w, out, len(out), page)
}

func (h *UsageHandler) GetQuotaState(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}

	policyID, ok := requireQuotaPolicyID(w, r)
	if !ok {
		return
	}

	periodStart, periodEnd, err := quotaPeriodFromRequest(r)
	if err != nil {
		writeQuotaPeriodError(w, err)
		return
	}

	state, err := h.service.GetState(r.Context(), workspaceID, policyID, periodStart, periodEnd)
	if errors.Is(err, usagedomain.ErrQuotaStateNotFound) {
		writeError(w, http.StatusNotFound, "quota state not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, errFailedToGetQuotaState)
		return
	}

	_ = writeJSONOr500(w, map[string]any{"data": quotaStateToResponse(state)})
}

func requireQuotaPolicyID(w http.ResponseWriter, r *http.Request) (string, bool) {
	policyID := r.URL.Query().Get(queryQuotaPolicyID)
	if policyID == "" {
		writeError(w, http.StatusBadRequest, errQuotaPolicyIDRequired)
		return "", false
	}
	return policyID, true
}

func writeQuotaPeriodError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errBadPeriodStart):
		writeError(w, http.StatusBadRequest, errInvalidPeriodStart)
	case errors.Is(err, errBadPeriodEnd):
		writeError(w, http.StatusBadRequest, errInvalidPeriodEnd)
	default:
		writeError(w, http.StatusBadRequest, errInvalidBody)
	}
}

func usageEventToResponse(event *usagedomain.Event) usageEventResponse {
	return usageEventResponse{
		ID:            event.ID,
		WorkspaceID:   event.WorkspaceID,
		ActorID:       event.ActorID,
		ActorType:     event.ActorType,
		RunID:         event.RunID,
		ToolName:      event.ToolName,
		ModelName:     event.ModelName,
		InputUnits:    event.InputUnits,
		OutputUnits:   event.OutputUnits,
		EstimatedCost: event.EstimatedCost,
		LatencyMs:     event.LatencyMs,
		CreatedAt:     event.CreatedAt.Format(time.RFC3339),
	}
}

func quotaStateToResponse(state *usagedomain.State) quotaStateResponse {
	var lastEventAt *string
	if state.LastEventAt != nil {
		value := state.LastEventAt.UTC().Format(time.RFC3339)
		lastEventAt = &value
	}
	return quotaStateResponse{
		ID:            state.ID,
		WorkspaceID:   state.WorkspaceID,
		QuotaPolicyID: state.QuotaPolicyID,
		CurrentValue:  state.CurrentValue,
		PeriodStart:   state.PeriodStart.UTC().Format(time.RFC3339),
		PeriodEnd:     state.PeriodEnd.UTC().Format(time.RFC3339),
		LastEventAt:   lastEventAt,
		CreatedAt:     state.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     state.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

var (
	errBadPeriodStart = errors.New("bad period_start")
	errBadPeriodEnd   = errors.New("bad period_end")
)

func quotaPeriodFromRequest(r *http.Request) (time.Time, time.Time, error) {
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	if rawStart := r.URL.Query().Get(queryPeriodStart); rawStart != "" {
		parsed, err := time.Parse(time.RFC3339, rawStart)
		if err != nil {
			return time.Time{}, time.Time{}, errBadPeriodStart
		}
		start = parsed.UTC()
	}
	if rawEnd := r.URL.Query().Get(queryPeriodEnd); rawEnd != "" {
		parsed, err := time.Parse(time.RFC3339, rawEnd)
		if err != nil {
			return time.Time{}, time.Time{}, errBadPeriodEnd
		}
		end = parsed.UTC()
	}
	return start, end, nil
}

func optionalTrimmedQuery(r *http.Request, key string) *string {
	value := r.URL.Query().Get(key)
	if value == "" {
		return nil
	}
	return &value
}
