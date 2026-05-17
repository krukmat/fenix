package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/relationship"
	usagedomain "github.com/matiasleandrokruk/fenix/internal/domain/usage"
)

const (
	errFailedToListPolicies          = "failed to list quota policies"
	errFailedToListUsageEvts         = "failed to list usage events for governance"
	errFailedToEraseRelationshipData = "failed to erase relationship memory"
	errEntityTypeRequired            = "entityType is required"
	errEntityIDRequired              = "entityId is required"
	governanceUsageLimit             = 20
)

type governanceUsageReader interface {
	ListEvents(ctx context.Context, workspaceID string, runID *string, limit int) ([]*usagedomain.Event, error)
	ListActivePolicies(ctx context.Context, workspaceID string) ([]*usagedomain.Policy, error)
	GetState(ctx context.Context, workspaceID, policyID string, periodStart, periodEnd time.Time) (*usagedomain.State, error)
}

type relationshipMemoryEraser interface {
	EraseEntityMemory(ctx context.Context, workspaceID string, entityType relationship.EntityType, entityID string) error
}

// GovernanceHandler serves governance summary and compliance actions.
type GovernanceHandler struct {
	usage     governanceUsageReader
	lifecycle relationshipMemoryEraser
}

func NewGovernanceHandler(usage governanceUsageReader, lifecycle relationshipMemoryEraser) *GovernanceHandler {
	return &GovernanceHandler{usage: usage, lifecycle: lifecycle}
}

type quotaStateItemResponse struct {
	PolicyID        string  `json:"policyId"`
	PolicyType      string  `json:"policyType"`
	MetricName      string  `json:"metricName"`
	LimitValue      float64 `json:"limitValue"`
	ResetPeriod     string  `json:"resetPeriod"`
	EnforcementMode string  `json:"enforcementMode"`
	CurrentValue    float64 `json:"currentValue"`
	PeriodStart     string  `json:"periodStart"`
	PeriodEnd       string  `json:"periodEnd"`
	LastEventAt     *string `json:"lastEventAt,omitempty"`
	StatePresent    bool    `json:"statePresent"`
}

type governanceSummaryResponse struct {
	RecentUsage []usageEventResponse     `json:"recentUsage"`
	QuotaStates []quotaStateItemResponse `json:"quotaStates"`
}

type eraseRelationshipMemoryRequest struct {
	EntityType string `json:"entityType"`
	EntityID   string `json:"entityId"`
}

func (h *GovernanceHandler) GetGovernanceSummary(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	if h.usage == nil {
		writeError(w, http.StatusInternalServerError, errFailedToListUsageEvts)
		return
	}

	events, err := h.usage.ListEvents(r.Context(), workspaceID, nil, governanceUsageLimit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errFailedToListUsageEvts)
		return
	}

	recentUsage := make([]usageEventResponse, 0, len(events))
	for _, e := range events {
		recentUsage = append(recentUsage, usageEventToResponse(e))
	}

	policies, err := h.usage.ListActivePolicies(r.Context(), workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errFailedToListPolicies)
		return
	}

	now := time.Now().UTC()
	quotaStates := make([]quotaStateItemResponse, 0, len(policies))
	for _, p := range policies {
		quotaStates = append(quotaStates, h.enrichPolicyState(r, workspaceID, p, now))
	}

	_ = writeJSONOr500(w, governanceSummaryResponse{
		RecentUsage: recentUsage,
		QuotaStates: quotaStates,
	})
}

func (h *GovernanceHandler) EraseRelationshipMemory(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	if h.lifecycle == nil {
		writeError(w, http.StatusInternalServerError, errFailedToEraseRelationshipData)
		return
	}

	var req eraseRelationshipMemoryRequest
	if !decodeBodyJSON(w, r, &req) {
		return
	}
	if req.EntityType == "" {
		writeError(w, http.StatusBadRequest, errEntityTypeRequired)
		return
	}
	if req.EntityID == "" {
		writeError(w, http.StatusBadRequest, errEntityIDRequired)
		return
	}

	entityType := relationship.EntityType(req.EntityType)
	if err := h.lifecycle.EraseEntityMemory(r.Context(), workspaceID, entityType, req.EntityID); err != nil {
		writeError(w, http.StatusInternalServerError, errFailedToEraseRelationshipData)
		return
	}

	_ = writeJSONOr500(w, map[string]any{
		"status":      "erased",
		"workspaceId": workspaceID,
		"entityType":  req.EntityType,
		"entityId":    req.EntityID,
	})
}

func (h *GovernanceHandler) enrichPolicyState(
	r *http.Request,
	workspaceID string,
	p *usagedomain.Policy,
	now time.Time,
) quotaStateItemResponse {
	periodStart, periodEnd := currentPeriodBounds(now, p.ResetPeriod)
	item := quotaStateItemResponse{
		PolicyID:        p.ID,
		PolicyType:      p.PolicyType,
		MetricName:      p.MetricName,
		LimitValue:      p.LimitValue,
		ResetPeriod:     p.ResetPeriod,
		EnforcementMode: p.EnforcementMode,
		CurrentValue:    0,
		PeriodStart:     periodStart.UTC().Format(time.RFC3339),
		PeriodEnd:       periodEnd.UTC().Format(time.RFC3339),
		StatePresent:    false,
	}
	state, stateErr := h.usage.GetState(r.Context(), workspaceID, p.ID, periodStart, periodEnd)
	if stateErr != nil {
		return item
	}
	item.CurrentValue = state.CurrentValue
	item.StatePresent = true
	if state.LastEventAt != nil {
		v := state.LastEventAt.UTC().Format(time.RFC3339)
		item.LastEventAt = &v
	}
	return item
}

func currentPeriodBounds(now time.Time, resetPeriod string) (time.Time, time.Time) {
	switch resetPeriod {
	case "daily":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 0, 1)
	case "weekly":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 0, 7)
	default:
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 1, 0)
	}
}
