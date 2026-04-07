package handlers

// W1-T4 (mobile_wedge_harmonization_plan): GET /api/v1/governance/summary
// Returns recent usage events + enriched quota states (policy metadata + current state) for a workspace.
// Designed for the mobile Governance screen — one call, no client-side joins.

import (
	"net/http"
	"time"

	usagedomain "github.com/matiasleandrokruk/fenix/internal/domain/usage"
)

const (
	errFailedToListPolicies  = "failed to list quota policies"
	errFailedToListUsageEvts = "failed to list usage events for governance"
	governanceUsageLimit     = 20
)

// GovernanceHandler serves the governance summary endpoint.
type GovernanceHandler struct {
	service *usagedomain.Service
}

// NewGovernanceHandler constructs a GovernanceHandler.
func NewGovernanceHandler(service *usagedomain.Service) *GovernanceHandler {
	return &GovernanceHandler{service: service}
}

// quotaStateItemResponse is the enriched response — policy metadata + current state merged.
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
	// false when no state row exists yet for the current period (currentValue is 0)
	StatePresent bool `json:"statePresent"`
}

type governanceSummaryResponse struct {
	RecentUsage []usageEventResponse     `json:"recentUsage"`
	QuotaStates []quotaStateItemResponse `json:"quotaStates"`
}

// GetGovernanceSummary handles GET /api/v1/governance/summary.
func (h *GovernanceHandler) GetGovernanceSummary(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}

	// Recent usage events (last 20, no run filter)
	events, err := h.service.ListEvents(r.Context(), workspaceID, nil, governanceUsageLimit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errFailedToListUsageEvts)
		return
	}

	recentUsage := make([]usageEventResponse, 0, len(events))
	for _, e := range events {
		recentUsage = append(recentUsage, usageEventToResponse(e))
	}

	// Active quota policies
	policies, err := h.service.ListActivePolicies(r.Context(), workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errFailedToListPolicies)
		return
	}

	// For each policy, attempt to resolve the current period state.
	// If no state row exists yet, return a synthetic zero-value item (statePresent=false).
	now := time.Now().UTC()
	quotaStates := make([]quotaStateItemResponse, 0, len(policies))
	for _, p := range policies {
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

		state, stateErr := h.service.GetState(r.Context(), workspaceID, p.ID, periodStart, periodEnd)
		if stateErr == nil {
			item.CurrentValue = state.CurrentValue
			item.StatePresent = true
			if state.LastEventAt != nil {
				v := state.LastEventAt.UTC().Format(time.RFC3339)
				item.LastEventAt = &v
			}
		}
		// ErrQuotaStateNotFound → keep synthetic zero; other errors → same (best-effort)
		quotaStates = append(quotaStates, item)
	}

	_ = writeJSONOr500(w, governanceSummaryResponse{
		RecentUsage: recentUsage,
		QuotaStates: quotaStates,
	})
}

// currentPeriodBounds computes [start, end) for the given reset period relative to now.
func currentPeriodBounds(now time.Time, resetPeriod string) (time.Time, time.Time) {
	switch resetPeriod {
	case "daily":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 0, 1)
	case "weekly":
		// ISO week: Monday = day 0
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 0, 7)
	default: // "monthly" and anything else
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 1, 0)
	}
}
